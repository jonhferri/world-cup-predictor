// Package tips enforces the per-match prediction rules server-side:
//   - a Tip can only be created/edited while now < match.kickoff (lock)
//   - knockout Tips are only allowed once both teams are resolved
//   - the knockout advancer is derived from the phased prediction
//   - other players' Tips are visible only AFTER kickoff and only to people
//     who share a League (the /api/tips/others/{matchId} endpoint)
package tips

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
)

// groupStageSiblingFilter builds a PocketBase filter that matches all group-stage
// matches in the same round-set as roundLabel. WC 2026 has 17 matchdays split:
//   - Matchday 1–7  → group set 1  (first match each team plays)
//   - Matchday 8–13 → group set 2  (second match)
//   - Matchday 14–17 → group set 3  (third match)
func groupStageSiblingFilter(roundLabel string) (filter string, params map[string]any) {
	n, _ := strconv.Atoi(strings.TrimPrefix(roundLabel, "Matchday "))
	var lo, hi int
	switch {
	case n >= 14:
		lo, hi = 14, 17
	case n >= 8:
		lo, hi = 8, 13
	default:
		lo, hi = 1, 7
	}
	params = make(map[string]any)
	parts := make([]string, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		key := fmt.Sprintf("rl%d", i)
		params[key] = fmt.Sprintf("Matchday %d", i)
		parts = append(parts, fmt.Sprintf("roundLabel = {:%s}", key))
	}
	filter = "(" + strings.Join(parts, " || ") + `) && id != {:mid}`
	return filter, params
}

func matchKickoff(m *core.Record) time.Time {
	return m.GetDateTime("kickoff").Time()
}

func isLocked(now, kickoff time.Time) bool {
	return !now.Before(kickoff)
}

func locked(app core.App, m *core.Record) bool {
	return isLocked(clock.Now(app), matchKickoff(m))
}

// bypass lets the dev bot generator insert tips for every match regardless
// of lock / knockout-resolution. Never set in production (dev-only path).
var bypass atomic.Bool

// SetBypass toggles the dev-only validation bypass.
func SetBypass(b bool) { bypass.Store(b) }

// validateTurbo enforces the turbo rules:
//   - turbo cannot be unset once applied (immutable after first save)
//   - max 1 turbo per stage-group (matchday for group stage; round for KO)
//   - FINAL and 3RD are auto-doubled in scoring — turbo flag rejected for them
func validateTurbo(app core.App, tip *core.Record, match *core.Record) error {
	stage := match.GetString("stage")

	if !tip.GetBool("turbo") {
		return nil
	}

	// FINAL and 3RD are automatically doubled in scoring; no user turbo needed.
	if stage == "FINAL" || stage == "3RD" {
		tip.Set("turbo", false)
		return nil
	}

	userID := tip.GetString("user")

	// Find sibling matches in the same stage group.
	var matchFilter string
	var filterParams map[string]any
	if stage == "group" {
		matchFilter, filterParams = groupStageSiblingFilter(match.GetString("roundLabel"))
		filterParams["mid"] = match.Id
	} else {
		matchFilter = `stage = {:s} && id != {:mid}`
		filterParams = map[string]any{"s": stage, "mid": match.Id}
	}
	siblings, _ := app.FindRecordsByFilter("matches", matchFilter, "", 0, 0, filterParams)
	if len(siblings) == 0 {
		return nil
	}

	// Build a single OR filter to check whether any sibling already has turbo.
	parts := make([]string, len(siblings))
	for i, gm := range siblings {
		parts[i] = fmt.Sprintf(`match = "%s"`, gm.Id)
	}
	filter := fmt.Sprintf(`user = {:u} && turbo = true && (%s)`, strings.Join(parts, " || "))
	existing, _ := app.FindFirstRecordByFilter("tips", filter, map[string]any{"u": userID})
	if existing != nil {
		return apis.NewBadRequestError("turbo already used in this round", nil)
	}
	return nil
}

// validateAndDerive applies lock + validation and sets the derived advancer.
func validateAndDerive(app core.App, tip *core.Record) error {
	if bypass.Load() {
		return nil
	}
	match, err := app.FindRecordById("matches", tip.GetString("match"))
	if err != nil {
		return apis.NewBadRequestError("unknown match", nil)
	}
	if locked(app, match) {
		return apis.NewBadRequestError("this match is locked (kickoff passed)", nil)
	}

	ftH := tip.GetInt("ftHome")
	ftA := tip.GetInt("ftAway")
	if tip.Get("ftHome") == nil || tip.Get("ftAway") == nil {
		return apis.NewBadRequestError("full-time score is required", nil)
	}
	if ftH < 0 || ftA < 0 || ftH > 99 || ftA > 99 {
		return apis.NewBadRequestError("scores out of range", nil)
	}

	if err := validateTurbo(app, tip, match); err != nil {
		return err
	}

	if match.GetString("stage") == "group" {
		tip.Set("etHome", 0)
		tip.Set("etAway", 0)
		tip.Set("penWinner", "")
		tip.Set("advancer", "")
		return nil
	}

	// Knockout.
	home := match.GetString("homeTeam")
	away := match.GetString("awayTeam")
	if home == "" || away == "" {
		return apis.NewBadRequestError("this matchup is not set yet", nil)
	}

	if ftH != ftA {
		if ftH > ftA {
			tip.Set("advancer", home)
		} else {
			tip.Set("advancer", away)
		}
		tip.Set("etHome", 0)
		tip.Set("etAway", 0)
		tip.Set("penWinner", "")
		return nil
	}

	// Drawn after 90' -> extra time required (cumulative >= FT).
	etH := tip.GetInt("etHome")
	etA := tip.GetInt("etAway")
	if tip.Get("etHome") == nil || tip.Get("etAway") == nil {
		return apis.NewBadRequestError("predict the score after extra time", nil)
	}
	if etH < ftH || etA < ftA {
		return apis.NewBadRequestError("extra-time score must include the 90' goals", nil)
	}
	if etH != etA {
		if etH > etA {
			tip.Set("advancer", home)
		} else {
			tip.Set("advancer", away)
		}
		tip.Set("penWinner", "")
		return nil
	}

	// Still level -> penalty winner required.
	pw := tip.GetString("penWinner")
	if pw != home && pw != away {
		return apis.NewBadRequestError("pick who wins the penalty shootout", nil)
	}
	tip.Set("advancer", pw)
	return nil
}

// Register wires the Tip validation hooks and the friends-tips endpoint.
func Register(app core.App, se *core.ServeEvent) {
	app.OnRecordCreate("tips").BindFunc(func(e *core.RecordEvent) error {
		if err := validateAndDerive(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate("tips").BindFunc(func(e *core.RecordEvent) error {
		if err := validateAndDerive(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordDelete("tips").BindFunc(func(e *core.RecordEvent) error {
		if m, err := e.App.FindRecordById("matches", e.Record.GetString("match")); err == nil && locked(e.App, m) {
			return apis.NewBadRequestError("this match is locked", nil)
		}
		return e.Next()
	})

	// GET /api/tips/scores — the signed-in user's points per match under the
	// default scoring config (for the per-match "+N pt" badge).
	se.Router.GET("/api/tips/scores", func(e *core.RequestEvent) error {
		out := map[string]int{}
		def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
		if err == nil {
			rows, _ := app.FindRecordsByFilter("match_scores",
				"user = {:u} && config = {:c}", "", 0, 0,
				map[string]any{"u": e.Auth.Id, "c": def.Id})
			for _, r := range rows {
				out[r.GetString("match")] = r.GetInt("points")
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"scores": out})
	}).Bind(apis.RequireAuth())

	// GET /api/tips/others/{matchId} — all league members' Tips for a match,
	// but only after kickoff. The requesting user's own tip is included first
	// (isMe: true). Each row includes the points earned under the default config.
	se.Router.GET("/api/tips/others/{matchId}", func(e *core.RequestEvent) error {
		matchID := e.Request.PathValue("matchId")
		match, err := app.FindRecordById("matches", matchID)
		if err != nil {
			return apis.NewNotFoundError("match not found", nil)
		}
		if !locked(app, match) {
			// Not started: never reveal anyone's picks.
			return e.JSON(http.StatusOK, map[string]any{"locked": false, "tips": []any{}})
		}

		// Default scoring config for points display.
		def, _ := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
		type scoreRow struct {
			points     int
			components string
		}
		scoreFor := func(uid string) scoreRow {
			if def == nil {
				return scoreRow{points: -1}
			}
			s, err := app.FindFirstRecordByFilter("match_scores",
				"user = {:u} && match = {:m} && config = {:c}",
				map[string]any{"u": uid, "m": matchID, "c": def.Id})
			if err != nil {
				return scoreRow{points: -1}
			}
			return scoreRow{points: s.GetInt("points"), components: s.GetString("components")}
		}

		coMembers, err := sharedLeagueUserIDs(app, e.Auth.Id)
		if err != nil {
			return err
		}
		allTips, err := app.FindRecordsByFilter("tips",
			"match = {:m}", "", 0, 0, map[string]any{"m": matchID})
		if err != nil {
			return err
		}

		var myRow *map[string]any
		otherRows := make([]map[string]any, 0, len(allTips))
		for _, t := range allTips {
			uid := t.GetString("user")
			isMe := uid == e.Auth.Id
			if !isMe && !coMembers[uid] {
				continue
			}
			u, err := app.FindRecordById("users", uid)
			if err != nil {
				continue
			}
			sr := scoreFor(uid)
			row := map[string]any{
				"userId":      uid,
				"name":        u.GetString("name"),
				"isMe":        isMe,
				"ftHome":      t.GetInt("ftHome"),
				"ftAway":      t.GetInt("ftAway"),
				"etHome":      t.GetInt("etHome"),
				"etAway":      t.GetInt("etAway"),
				"penWinner":   t.GetString("penWinner"),
				"advancer":    t.GetString("advancer"),
				"firstTeam":   t.GetString("firstTeam"),
				"firstPlayer": t.GetString("firstPlayer"),
				"turbo":       t.GetBool("turbo"),
				"points":      sr.points,
				"components":  sr.components,
			}
			if isMe {
				r := row
				myRow = &r
			} else {
				otherRows = append(otherRows, row)
			}
		}
		out := make([]map[string]any, 0, len(otherRows)+1)
		if myRow != nil {
			out = append(out, *myRow)
		}
		out = append(out, otherRows...)
		return e.JSON(http.StatusOK, map[string]any{"locked": true, "tips": out})
	}).Bind(apis.RequireAuth())

	// GET /api/tips/crowd/{matchId} — global tip distribution (Home/Draw/Away)
	// across ALL users for a single match. Revealed only after kickoff so we
	// never leak picks before tips lock.
	se.Router.GET("/api/tips/crowd/{matchId}", func(e *core.RequestEvent) error {
		matchID := e.Request.PathValue("matchId")
		match, err := app.FindRecordById("matches", matchID)
		if err != nil {
			return apis.NewNotFoundError("match not found", nil)
		}
		if !locked(app, match) {
			return e.JSON(http.StatusOK, map[string]any{"locked": false})
		}
		dist, err := crowdDistribution(app, match)
		if err != nil {
			return err
		}
		dist["locked"] = true
		return e.JSON(http.StatusOK, dist)
	}).Bind(apis.RequireAuth())
}

// crowdDistribution aggregates every tip for the given match into
// Home / Draw / Away buckets and returns counts plus integer percentages
// that always sum to 100 (largest bucket absorbs any rounding drift).
//
// Group stage: outcome = sign(ftHome - ftAway).
// Knockout: outcome = advancer == homeTeam ? home : away (no draws possible).
func crowdDistribution(app core.App, match *core.Record) (map[string]any, error) {
	tips, err := app.FindRecordsByFilter("tips",
		"match = {:m}", "", 0, 0, map[string]any{"m": match.Id})
	if err != nil {
		return nil, err
	}
	stage := match.GetString("stage")
	isKO := stage != "group"
	home := match.GetString("homeTeam")
	away := match.GetString("awayTeam")
	var hC, dC, aC int
	for _, t := range tips {
		if isKO {
			adv := t.GetString("advancer")
			switch adv {
			case home:
				hC++
			case away:
				aC++
			}
			continue
		}
		// Group stage. Skip malformed rows (missing FT score).
		if t.Get("ftHome") == nil || t.Get("ftAway") == nil {
			continue
		}
		ftH := t.GetInt("ftHome")
		ftA := t.GetInt("ftAway")
		switch {
		case ftH > ftA:
			hC++
		case ftH < ftA:
			aC++
		default:
			dC++
		}
	}
	total := hC + dC + aC
	hP, dP, aP := pctSplit(hC, dC, aC, total)
	return map[string]any{
		"total": total,
		"isKO":  isKO,
		"outcomes": map[string]any{
			"home": map[string]any{"count": hC, "pct": hP},
			"draw": map[string]any{"count": dC, "pct": dP},
			"away": map[string]any{"count": aC, "pct": aP},
		},
	}, nil
}

// pctSplit returns integer percentages for (home, draw, away) that sum to 100
// when total > 0; returns zeros when total == 0. The largest raw bucket
// absorbs any rounding drift so the bar always renders cleanly.
func pctSplit(h, d, a, total int) (int, int, int) {
	if total <= 0 {
		return 0, 0, 0
	}
	hP := (h * 100) / total
	dP := (d * 100) / total
	aP := (a * 100) / total
	diff := 100 - (hP + dP + aP)
	if diff != 0 {
		// Give the leftover to whichever bucket has the most votes.
		switch {
		case h >= d && h >= a:
			hP += diff
		case d >= h && d >= a:
			dP += diff
		default:
			aP += diff
		}
	}
	return hP, dP, aP
}

// sharedLeagueUserIDs returns the set of user ids that share at least one
// League with the given user.
func sharedLeagueUserIDs(app core.App, userID string) (map[string]bool, error) {
	mine, err := app.FindRecordsByFilter("league_members",
		"user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, lm := range mine {
		peers, err := app.FindRecordsByFilter("league_members",
			"league = {:l}", "", 0, 0, map[string]any{"l": lm.GetString("league")})
		if err != nil {
			return nil, err
		}
		for _, p := range peers {
			out[p.GetString("user")] = true
		}
	}
	return out, nil
}
