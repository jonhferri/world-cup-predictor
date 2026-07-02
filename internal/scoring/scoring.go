// Package scoring computes match (Tip) and tournament (Forecast) points from
// a per-League scoring config, recomputes on every result change, and builds
// League leaderboards with the agreed tiebreakers.
//
// Scale is tiny (friends app: a handful of users, 104 matches), so every
// result change triggers a full, idempotent recompute — simplest and always
// correct.
package scoring

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/text/unicode/norm"

	"github.com/oyvhov/world-cup-pool/internal/bracket"
	"github.com/oyvhov/world-cup-pool/internal/standings"
	"github.com/oyvhov/world-cup-pool/internal/topscorer"
)

// normalizeAccents folds diacritics so "Julián" == "Julian" when comparing
// player names from different sources.
func normalizeAccents(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue // drop combining marks
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}


func normalizeName(s string) string {
    // decompose, drop combining marks, lowercase, strip punctuation, collapse spaces
    decomp := norm.NFD.String(s)
    var b strings.Builder
    for _, r := range decomp {
        if unicode.Is(unicode.Mn, r) { // skip diacritics
            continue
        }
        if unicode.IsPunct(r) || unicode.IsSymbol(r) {
            continue
        }
        b.WriteRune(unicode.ToLower(r))
    }
    out := strings.Join(strings.Fields(b.String()), " ")
    return strings.TrimSpace(out)
}
// ---- Config ----

type Config struct {
	Match struct {
		Tendency          int `json:"tendency"`          // correct result (group 1/X/2; KO = who advances)
		Exact             int `json:"exact"`             // exact reference score
		TotalGoals        int `json:"totalGoals"`        // correct total goals
		GoalDiff          int `json:"goalDiff"`          // correct goal difference
		FirstTeamScorer   int `json:"firstTeamScorer"`   // correct first team to score
		FirstPlayerScorer int `json:"firstPlayerScorer"` // correct first player to score
		// KO-specific FT scoring (applied to the 90' score)
		KOFtTendency  int `json:"koFtTendency"`  // correct FT outcome (H/D/A)
		KOFtGoalDiff  int `json:"koFtGoalDiff"`  // correct goal difference at FT
		KOFtExactHome int `json:"koFtExactHome"` // exact home goals at FT
		KOFtExactAway int `json:"koFtExactAway"` // exact away goals at FT
		KOFtExact     int `json:"koFtExact"`     // both FT goals exactly right
		// KO-specific ET scoring (only if match went to ET and user predicted FT draw)
		KOEtTendency  int `json:"koEtTendency"`  // correct ET outcome (H/D/A)
		KOEtGoalDiff  int `json:"koEtGoalDiff"`  // correct goal difference at ET
		KOEtExactHome int `json:"koEtExactHome"` // exact home goals at ET (cumulative)
		KOEtExactAway int `json:"koEtExactAway"` // exact away goals at ET (cumulative)
		KOEtExact     int `json:"koEtExact"`     // both ET goals exactly right
		// Advancer (KO only)
		KOAdvancer int `json:"koAdvancer"` // correct team to advance
	} `json:"match"`
	Forecast struct {
		GroupPosition     int            `json:"groupPosition"`     // per exact final position
		PerfectGroupBonus int            `json:"perfectGroupBonus"` // whole group perfect
		Advance           int            `json:"advance"`           // per predicted advancer that advances
		GoldenBootWinner  int            `json:"goldenBootWinner"`  // correct Golden Boot winner
		Round             map[string]int `json:"round"`             // predicted team reaching a KO round
	} `json:"forecast"`
}

func loadConfig(rec *core.Record) Config {
	var c Config
	_ = json.Unmarshal([]byte(rec.GetString("config")), &c)
	// Backward-compat defaults.
	if c.Forecast.Advance == 0 {
		c.Forecast.Advance = 1
	}
	if c.Forecast.GoldenBootWinner == 0 {
		c.Forecast.GoldenBootWinner = 15
	}
	// KO fields default to 5 if not present in older configs.
	if c.Match.KOFtTendency == 0 {
		c.Match.KOFtTendency = 5
	}
	if c.Match.KOFtGoalDiff == 0 {
		c.Match.KOFtGoalDiff = 5
	}
	if c.Match.KOFtExactHome == 0 {
		c.Match.KOFtExactHome = 5
	}
	if c.Match.KOFtExactAway == 0 {
		c.Match.KOFtExactAway = 5
	}
	if c.Match.KOFtExact == 0 {
		c.Match.KOFtExact = 5
	}
	if c.Match.KOEtTendency == 0 {
		c.Match.KOEtTendency = 5
	}
	if c.Match.KOEtGoalDiff == 0 {
		c.Match.KOEtGoalDiff = 5
	}
	if c.Match.KOEtExactHome == 0 {
		c.Match.KOEtExactHome = 5
	}
	if c.Match.KOEtExactAway == 0 {
		c.Match.KOEtExactAway = 5
	}
	if c.Match.KOEtExact == 0 {
		c.Match.KOEtExact = 5
	}
	if c.Match.KOAdvancer == 0 {
		c.Match.KOAdvancer = 10
	}
	return c
}

// configsInUse returns every scoring config referenced by a League plus the
// default, so per-(user,match,config) scores cover all Leagues.
func configsInUse(app core.App) (map[string]Config, string, error) {
	out := map[string]Config{}
	def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
	if err != nil {
		return nil, "", err
	}
	out[def.Id] = loadConfig(def)
	leagues, err := app.FindRecordsByFilter("leagues", "id != ''", "", 0, 0)
	if err != nil {
		return nil, "", err
	}
	for _, l := range leagues {
		cid := l.GetString("scoringConfig")
		if _, done := out[cid]; cid == "" || done {
			continue
		}
		if cr, err := app.FindRecordById("scoring_configs", cid); err == nil {
			out[cid] = loadConfig(cr)
		}
	}
	return out, def.Id, nil
}

func sign(n int) int {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}

// ---- Match (Tip) scoring ----

type tipComponents struct {
	Tendency          int  `json:"tendency"` // group: correct 1/X/2; not used for KO
	Exact             int  `json:"exact"`    // group only
	TotalGoals        int  `json:"totalGoals"` // group only (exact home or away goals)
	GoalDiff          int  `json:"goalDiff"`   // group only
	GdDev             int  `json:"gdDev"` // |predicted GD - actual GD| (tiebreaker only)
	FirstTeamScorer   int  `json:"firstTeamScorer"`
	FirstPlayerScorer int  `json:"firstPlayerScorer"`
	// KO-specific components
	KOFtTendency  int `json:"koFtTendency"`
	KOFtGoalDiff  int `json:"koFtGoalDiff"`
	KOFtExactHome int `json:"koFtExactHome"`
	KOFtExactAway int `json:"koFtExactAway"`
	KOFtExact     int `json:"koFtExact"`
	KOEtTendency  int `json:"koEtTendency"`
	KOEtGoalDiff  int `json:"koEtGoalDiff"`
	KOEtExactHome int `json:"koEtExactHome"`
	KOEtExactAway int `json:"koEtExactAway"`
	KOEtExact     int `json:"koEtExact"`
	KOAdvancer    int `json:"koAdvancer"`
	Turbo         bool `json:"turbo"` // points doubled (user turbo or FINAL/3RD auto)
}

func (c tipComponents) points() int {
	base := c.Tendency + c.Exact + c.TotalGoals + c.GoalDiff +
		c.FirstTeamScorer + c.FirstPlayerScorer +
		c.KOFtTendency + c.KOFtGoalDiff + c.KOFtExactHome + c.KOFtExactAway + c.KOFtExact +
		c.KOEtTendency + c.KOEtGoalDiff + c.KOEtExactHome + c.KOEtExactAway + c.KOEtExact +
		c.KOAdvancer
	if c.Turbo {
		return base * 2
	}
	return base
}

// MatchResult / TipPrediction are the plain inputs to the pure scorer, so the
// rules are unit-testable without a database.
type MatchResult struct {
	Stage             string
	FtH, FtA         int
	EtH, EtA         int
	Advancer          string
	FirstTeamScorer   string
	FirstPlayerScorer string
}
type TipPrediction struct {
	FtH, FtA   int
	EtH, EtA   int
	Advancer    string
	FirstTeam   string
	FirstPlayer string
}

// scoreValues is the pure scoring core (see scoring_test.go).
//
// Group matches: Tendency (correct 1/X/2) + exact home goals + exact away goals +
// exact score bonus + goal difference + first team/player scorer.
//
// Knockout matches use a separate FT+ET structure:
//   - FT (max 20): tendency (H/D/A at 90') + exact home goals + exact away goals + exact bonus
//   - ET (max 20, only when match went to ET AND user predicted FT draw): same 4 components
//     against cumulative ET scores
//   - Advancer (5): correct team to advance (pen winner counts)
//   - First team/player scorer: unchanged
func scoreValues(cfg Config, m MatchResult, p TipPrediction) tipComponents {
	var r tipComponents

	if m.Stage == "group" {
		// ---- Group match scoring ----
		if sign(p.FtH-p.FtA) == sign(m.FtH-m.FtA) {
			r.Tendency = cfg.Match.Tendency
		}
		if p.FtH == m.FtH && p.FtA == m.FtA {
			r.Exact = cfg.Match.Exact
		}
		if p.FtH == m.FtH {
			r.TotalGoals += cfg.Match.TotalGoals
		}
		if p.FtA == m.FtA {
			r.TotalGoals += cfg.Match.TotalGoals
		}
		if p.FtH-p.FtA == m.FtH-m.FtA {
			r.GoalDiff = cfg.Match.GoalDiff
		}
		if d := (p.FtH - p.FtA) - (m.FtH - m.FtA); d < 0 {
			r.GdDev = -d
		} else {
			r.GdDev = d
		}
	} else {
		// ---- Knockout match scoring ----

		// FT components (always scored, based on the actual 90' result)
		if sign(p.FtH-p.FtA) == sign(m.FtH-m.FtA) {
			r.KOFtTendency = cfg.Match.KOFtTendency
		}
		if p.FtH-p.FtA == m.FtH-m.FtA {
			r.KOFtGoalDiff = cfg.Match.KOFtGoalDiff
		}
		if p.FtH == m.FtH {
			r.KOFtExactHome = cfg.Match.KOFtExactHome
		}
		if p.FtA == m.FtA {
			r.KOFtExactAway = cfg.Match.KOFtExactAway
		}
		if p.FtH == m.FtH && p.FtA == m.FtA {
			r.KOFtExact = cfg.Match.KOFtExact
		}
		if d := (p.FtH - p.FtA) - (m.FtH - m.FtA); d < 0 {
			r.GdDev = -d
		} else {
			r.GdDev = d
		}

		// ET components: only if the match actually went to ET and the user predicted a FT draw
		wentET := m.EtH != 0 || m.EtA != 0
		userPredictedDraw := p.FtH == p.FtA
		if wentET && userPredictedDraw {
			if sign(p.EtH-p.EtA) == sign(m.EtH-m.EtA) {
				r.KOEtTendency = cfg.Match.KOEtTendency
			}
			if p.EtH-p.EtA == m.EtH-m.EtA {
				r.KOEtGoalDiff = cfg.Match.KOEtGoalDiff
			}
			if p.EtH == m.EtH {
				r.KOEtExactHome = cfg.Match.KOEtExactHome
			}
			if p.EtA == m.EtA {
				r.KOEtExactAway = cfg.Match.KOEtExactAway
			}
			if p.EtH == m.EtH && p.EtA == m.EtA {
				r.KOEtExact = cfg.Match.KOEtExact
			}
		}

		// Advancer
		if m.Advancer != "" && m.Advancer == p.Advancer {
			r.KOAdvancer = cfg.Match.KOAdvancer
		}
	}

	// First scorer (all match types)
	if m.FirstTeamScorer != "" && m.FirstTeamScorer == p.FirstTeam {
		r.FirstTeamScorer = cfg.Match.FirstTeamScorer
	}
	if m.FirstPlayerScorer != "" && normalizeName(m.FirstPlayerScorer) == normalizeName(p.FirstPlayer) {
		r.FirstPlayerScorer = cfg.Match.FirstPlayerScorer
	}
	return r
}

// resolvePlayerTeam looks up the team record ID for the given player name.
// It first tries an exact match, then falls back to normalized comparison
// (diacritics stripped, lowercase) so "Julián" matches "Julian".
func resolvePlayerTeam(app core.App, playerName string) string {
  if p, err := app.FindFirstRecordByFilter("players",
    "name = {:n}", map[string]any{"n": playerName}); err == nil {
    return p.GetString("teamId")
  }
  players, err := app.FindRecordsByFilter("players", "id != ''", "", 0, 0)
  if err != nil {
    return ""
  }
  norm := normalizeName(playerName)
  for _, p := range players {
    if normalizeName(p.GetString("name")) == norm {
      return p.GetString("teamId")
    }
  }
  return ""
}

func scoreTip(app core.App, cfg Config, match, tip *core.Record) tipComponents {
  firstTeam := tip.GetString("firstTeam")
  if firstTeam == "" {
    if player := tip.GetString("firstPlayer"); player != "" {
      firstTeam = resolvePlayerTeam(app, player)
    }
  }
	r := scoreValues(cfg,
		MatchResult{
			Stage:             match.GetString("stage"),
			FtH:               match.GetInt("ftHome"),
			FtA:               match.GetInt("ftAway"),
			EtH:               match.GetInt("etHome"),
			EtA:               match.GetInt("etAway"),
			Advancer:          match.GetString("advancer"),
			FirstTeamScorer:   match.GetString("firstTeamScorer"),
			FirstPlayerScorer: match.GetString("firstPlayerScorer"),
		},
		TipPrediction{
			FtH:         tip.GetInt("ftHome"),
			FtA:         tip.GetInt("ftAway"),
			EtH:         tip.GetInt("etHome"),
			EtA:         tip.GetInt("etAway"),
			Advancer:    tip.GetString("advancer"),
      		FirstTeam:   firstTeam,
			FirstPlayer: tip.GetString("firstPlayer"),
		},
	)
	stage := match.GetString("stage")
	r.Turbo = tip.GetBool("turbo") || stage == "FINAL" || stage == "3RD"
	return r
}

// ---- Group standings (final, from finalized group matches) ----

type teamAgg struct {
	id                 string
	pts, gd, gf, games int
}

// finalGroups returns, for each fully-finished group, the ordered team ids
// (1st..4th) and collects every finished group's third-placed team for the
// best-third rank. The FIFA tiebreaker order (including head-to-head) lives in
// internal/standings, shared with bracket resolution so the two never disagree.
func finalGroups(app core.App) (order map[string][]string, thirds []teamAgg) {
	ms, _ := app.FindRecordsByFilter("matches",
		"stage = 'group' && finalizedAt != ''", "", 0, 0)
	var ranked []standings.Row
	order, ranked, _ = standings.GroupTables(standings.FromRecords(ms))
	for _, r := range ranked {
		thirds = append(thirds, teamAgg{id: r.TeamID, pts: r.Pts, gd: r.GD, gf: r.GF, games: r.Games})
	}
	return order, thirds
}

func sortAggs(a []teamAgg) {
	sort.Slice(a, func(i, j int) bool {
		if a[i].pts != a[j].pts {
			return a[i].pts > a[j].pts
		}
		if a[i].gd != a[j].gd {
			return a[i].gd > a[j].gd
		}
		return a[i].gf > a[j].gf
	})
}

func bestThirdSet(thirds []teamAgg) map[string]bool {
	sortAggs(thirds)
	set := map[string]bool{}
	for i, t := range thirds {
		if i >= 8 {
			break
		}
		set[t.id] = true
	}
	return set
}

// ---- Auto-derive group standings from tips ----

// deriveGroupOrder computes predicted group standings from a user's tip
// records for group-stage matches, using pts → gd → gf sort (simplified;
// no head-to-head tiebreaker). Returns only groups where the user tipped
// every match (3 matches per group of 4 teams).
func deriveGroupOrder(groupMatches []*core.Record, tipByMatch map[string]*core.Record) map[string][]string {
	type row struct {
		teamID string
		pts, gd, gf, games int
	}
	groupRows := map[string]map[string]*row{}
	groupMatchCount := map[string]int{}

	for _, m := range groupMatches {
		groupMatchCount[m.GetString("groupLetter")]++
		tip, ok := tipByMatch[m.Id]
		if !ok {
			continue
		}
		g := m.GetString("groupLetter")
		if groupRows[g] == nil {
			groupRows[g] = map[string]*row{}
		}
		home := m.GetString("homeTeam")
		away := m.GetString("awayTeam")
		if home == "" || away == "" {
			continue
		}
		ftH := tip.GetInt("ftHome")
		ftA := tip.GetInt("ftAway")
		if groupRows[g][home] == nil {
			groupRows[g][home] = &row{teamID: home}
		}
		if groupRows[g][away] == nil {
			groupRows[g][away] = &row{teamID: away}
		}
		h := groupRows[g][home]
		a := groupRows[g][away]
		h.gf += ftH
		h.gd += ftH - ftA
		h.games++
		a.gf += ftA
		a.gd += ftA - ftH
		a.games++
		switch {
		case ftH > ftA:
			h.pts += 3
		case ftH < ftA:
			a.pts += 3
		default:
			h.pts++
			a.pts++
		}
	}

	result := map[string][]string{}
	for letter, rows := range groupRows {
		// Only include groups where the user tipped all matches.
		if len(rows) < 4 {
			continue
		}
		rowSlice := make([]*row, 0, len(rows))
		for _, r := range rows {
			rowSlice = append(rowSlice, r)
		}
		sort.Slice(rowSlice, func(i, j int) bool {
			if rowSlice[i].pts != rowSlice[j].pts {
				return rowSlice[i].pts > rowSlice[j].pts
			}
			if rowSlice[i].gd != rowSlice[j].gd {
				return rowSlice[i].gd > rowSlice[j].gd
			}
			return rowSlice[i].gf > rowSlice[j].gf
		})
		_ = groupMatchCount[letter] // referenced for clarity
		ids := make([]string, len(rowSlice))
		for i, r := range rowSlice {
			ids[i] = r.teamID
		}
		result[letter] = ids
	}
	return result
}

// ---- Forecast scoring ----

// actualRoundTeams maps stage -> set(teamId) of teams that actually reached
// that round, plus the actual champion.
func actualRoundTeams(app core.App) (map[string]map[string]bool, string) {
	res := map[string]map[string]bool{}
	champion := ""
	ms, _ := app.FindRecordsByFilter("matches", "stage != 'group'", "num", 0, 0)
	for _, m := range ms {
		st := m.GetString("stage")
		if res[st] == nil {
			res[st] = map[string]bool{}
		}
		for _, f := range []string{"homeTeam", "awayTeam"} {
			if id := m.GetString(f); id != "" {
				res[st][id] = true
			}
		}
		if st == "FINAL" && m.GetString("finalizedAt") != "" {
			champion = m.GetString("advancer")
		}
	}
	return res, champion
}

func tournamentComplete(app core.App) bool {
	finals, err := app.FindRecordsByFilter("matches", "stage = 'FINAL' && finalizedAt != ''", "", 1, 0)
	return err == nil && len(finals) > 0
}

type fcResolver struct {
	order      map[string][]string
	thirdByNum map[int]string // R32 match num -> chosen third teamId
	bracket    map[string]string
	ko         map[int]*core.Record
}

// assignThirds maps the user's chosen best thirds ({groupLetter: teamId})
// onto the 8 R32 third-slots. It uses FIFA's official Annex C allocation
// table for the given combination of 8 qualifying groups; if the combination
// isn't exactly 8 / not in the table it falls back to a deterministic
// backtracking matching. Identical logic on the frontend so the predicted
// Forecast bracket and its scoring always agree.
func assignThirds(koList []*core.Record, thirds map[string]string) map[int]string {
	type slot struct {
		num     int
		winner  string
		allowed []string
	}
	var slots []slot
	for _, mt := range koList {
		if mt.GetString("stage") != "R32" {
			continue
		}
		home, away := mt.GetString("homeLabel"), mt.GetString("awayLabel")
		for _, lbl := range []string{home, away} {
			if strings.HasPrefix(lbl, "3") && strings.Contains(lbl, "/") {
				w, _ := bracket.WinnerLetter(home, away)
				slots = append(slots, slot{
					num:     mt.GetInt("num"),
					winner:  w,
					allowed: strings.Split(strings.TrimPrefix(lbl, "3"), "/"),
				})
			}
		}
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].num < slots[j].num })

	chosen := make([]string, 0, len(thirds))
	for letter := range thirds {
		chosen = append(chosen, letter)
	}
	sort.Strings(chosen)

	// Official FIFA table for this exact set of 8 qualifying groups.
	if m, ok := bracket.Lookup(chosen); ok {
		out := map[int]string{}
		for _, s := range slots {
			if g, ok := m[s.winner]; ok {
				out[s.num] = thirds[g]
			}
		}
		return out
	}

	// Fallback: deterministic backtracking perfect matching.
	assign := make([]string, len(slots))
	var solve func(i int) bool
	solve = func(i int) bool {
		if i == len(slots) {
			return true
		}
		for _, letter := range chosen {
			taken := false
			for _, a := range assign {
				if a == letter {
					taken = true
					break
				}
			}
			if taken {
				continue
			}
			allowed := false
			for _, a := range slots[i].allowed {
				if a == letter {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
			assign[i] = letter
			if solve(i + 1) {
				return true
			}
			assign[i] = ""
		}
		return false
	}
	solve(0)

	out := map[int]string{}
	for i, s := range slots {
		if assign[i] != "" {
			out[s.num] = thirds[assign[i]]
		}
	}
	return out
}

func (r *fcResolver) resolve(label string, forNum int, seen map[int]bool) string {
	if label == "" {
		return ""
	}
	switch label[0] {
	case '1', '2':
		idx := 0
		if label[0] == '2' {
			idx = 1
		}
		o := r.order[label[1:]]
		if len(o) > idx {
			return o[idx]
		}
		return ""
	case '3':
		return r.thirdByNum[forNum]
	case 'W', 'L':
		n, _ := strconv.Atoi(label[1:])
		if seen[n] {
			return ""
		}
		seen[n] = true
		w := r.bracket[strconv.Itoa(n)]
		if label[0] == 'W' {
			return w
		}
		src := r.ko[n]
		if src == nil || w == "" {
			return ""
		}
		h := r.resolve(src.GetString("homeLabel"), n, seen)
		a := r.resolve(src.GetString("awayLabel"), n, seen)
		if w == h {
			return a
		}
		if w == a {
			return h
		}
		return ""
	}
	return ""
}

func koStableKey(m *core.Record) string {
	if n := m.GetInt("num"); n > 0 {
		return strconv.Itoa(n)
	}
	return m.GetString("stage")
}

type fcBreakdown struct {
	// Points.
	Groups     int `json:"groups"`   // exact final positions (+ perfect bonus)
	Advance    int `json:"advance"`  // predicted advancers that actually advanced
	Knockout   int `json:"knockout"` // predicted teams reaching KO rounds
	Champion   int `json:"champion"`
	GoldenBoot int `json:"goldenBoot"`
	// Correct-pick counts (for the Forecast leaderboard view).
	GroupsCorrect     int            `json:"groupsCorrect"`
	AdvanceCorrect    int            `json:"advanceCorrect"`
	RoundCorrect      map[string]int `json:"roundCorrect"` // R32..FINAL
	ChampionCorrect   int            `json:"championCorrect"`
	GoldenBootCorrect int            `json:"goldenBootCorrect"`
}

func (b fcBreakdown) total() int {
	return b.Groups + b.Advance + b.Knockout + b.Champion + b.GoldenBoot
}

// scoreForecast derives group / advancement / KO predictions from the user's
// match tips and scores them against actual results. The forecast record is
// only used for the golden boot pick.
func scoreForecast(app core.App, cfg Config, fc *core.Record) (fcBreakdown, int) {
	b := fcBreakdown{RoundCorrect: map[string]int{}}
	userID := fc.GetString("user")

	// Load user's tips keyed by match ID.
	userTips, _ := app.FindRecordsByFilter("tips", "user = {:u}", "", 0, 0,
		map[string]any{"u": userID})
	tipByMatch := make(map[string]*core.Record, len(userTips))
	for _, t := range userTips {
		tipByMatch[t.GetString("match")] = t
	}

	// ---- Group scoring (auto-derived from tips) ----
	groupMatches, _ := app.FindRecordsByFilter("matches",
		"stage = 'group' && finalizedAt != ''", "", 0, 0)
	derivedOrder := deriveGroupOrder(groupMatches, tipByMatch)

	actualOrder, thirdAggs := finalGroups(app)
	for g, actual := range actualOrder {
		pred := derivedOrder[g]
		allCorrect := len(pred) == 4
		for i := 0; i < 4 && i < len(actual); i++ {
			if i < len(pred) && pred[i] == actual[i] {
				b.Groups += cfg.Forecast.GroupPosition
				b.GroupsCorrect++
			} else {
				allCorrect = false
			}
		}
		if allCorrect && len(pred) == 4 {
			b.Groups += cfg.Forecast.PerfectGroupBonus
		}
	}

	// ---- Advance scoring (top 2 from derived groups + best derived thirds) ----
	actualAdv := map[string]bool{}
	for _, actual := range actualOrder {
		if len(actual) >= 2 {
			actualAdv[actual[0]] = true
			actualAdv[actual[1]] = true
		}
	}
	bestActual := map[string]bool{}
	if len(thirdAggs) >= 12 {
		bestActual = bestThirdSet(thirdAggs)
		for id := range bestActual {
			actualAdv[id] = true
		}
	}

	// Derive best-8 thirds from user's predicted 3rd-place teams.
	derivedThirds := make([]teamAgg, 0, 12)
	for _, pred := range derivedOrder {
		if len(pred) >= 3 {
			derivedThirds = append(derivedThirds, teamAgg{id: pred[2]})
		}
	}
	derivedBestThird := map[string]bool{}
	if len(derivedThirds) >= 8 {
		derivedBestThird = bestThirdSet(derivedThirds)
	}

	for g, pred := range derivedOrder {
		_ = g
		if len(pred) >= 2 {
			for _, pid := range pred[:2] {
				if actualAdv[pid] {
					b.Advance += cfg.Forecast.Advance
					b.AdvanceCorrect++
				}
			}
		}
		if len(pred) >= 3 && derivedBestThird[pred[2]] && actualAdv[pred[2]] {
			b.Advance += cfg.Forecast.Advance
			b.AdvanceCorrect++
		}
	}

	// ---- KO round scoring (from KO match tips, advancer field) ----
	actualRounds, actualChamp := actualRoundTeams(app)
	koMatches, _ := app.FindRecordsByFilter("matches",
		"stage != 'group' && finalizedAt != ''", "", 0, 0)
	for _, m := range koMatches {
		st := m.GetString("stage")
		w := cfg.Forecast.Round[st]
		if w == 0 {
			continue
		}
		tip := tipByMatch[m.Id]
		if tip == nil {
			continue
		}
		adv := tip.GetString("advancer")
		if adv == "" {
			continue
		}
		if actualRounds[st] != nil && actualRounds[st][adv] {
			b.Knockout += w
			b.RoundCorrect[st]++
		}
		// Champion: correct advancer in the FINAL = predicted champion.
		if st == "FINAL" && actualChamp != "" && adv == actualChamp {
			b.Champion += cfg.Forecast.Round["CHAMPION"]
			b.ChampionCorrect = 1
		}
	}

	// ---- Golden boot (from forecast record) ----
	if tournamentComplete(app) {
		winnerID := topscorer.WinnerID(app)
		winnerName := topscorer.WinnerName(app)
		pick := topscorer.PickFromForecast(fc)
		pickName := fc.GetString("goldenBootPlayerName")
		matched := (winnerID != "" && pick == winnerID) ||
			(winnerName != "" && (strings.EqualFold(pick, winnerName) || strings.EqualFold(pickName, winnerName)))
		if matched {
			b.GoldenBoot = cfg.Forecast.GoldenBootWinner
			b.GoldenBootCorrect = 1
		}
	}

	return b, b.total()
}
