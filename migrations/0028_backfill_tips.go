package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/oyvhov/world-cup-pool/internal/scoring"
)

// Backfills tips that users submitted in writing when the server was down.
// Also removes an incorrectly applied turbo boost.
func init() {
	m.Register(func(app core.App) error {
		// --- Helper: look up a team record ID by name ---
		teamID := func(name string) (string, error) {
			r, err := app.FindFirstRecordByFilter("teams", "name = {:n}", map[string]any{"n": name})
			if err != nil {
				return "", err
			}
			return r.Id, nil
		}

		// --- Helper: find the match between two teams ---
		matchByTeams := func(home, away string) (*core.Record, error) {
			hID, err := teamID(home)
			if err != nil {
				return nil, err
			}
			aID, err := teamID(away)
			if err != nil {
				return nil, err
			}
			return app.FindFirstRecordByFilter("matches",
				"homeTeam = {:h} && awayTeam = {:a}",
				map[string]any{"h": hID, "a": aID})
		}

		// --- Helper: find a player by name ---
		playerName := func(name string) string {
			r, err := app.FindFirstRecordByFilter("players", "name = {:n}", map[string]any{"n": name})
			if err != nil || r == nil {
				return name // fall back to plain name string if not found
			}
			return r.Id
		}

		// --- Helper: upsert a group-stage tip ---
		upsertGroupTip := func(userID, home, away string, ftH, ftA int, firstPlayer, firstTeamName string) error {
			match, err := matchByTeams(home, away)
			if err != nil {
				return err
			}
			firstTeamID, err := teamID(firstTeamName)
			if err != nil {
				return err
			}
			tip, err := app.FindFirstRecordByFilter("tips",
				"user = {:u} && match = {:m}",
				map[string]any{"u": userID, "m": match.Id})
			if err != nil {
				col, err2 := app.FindCollectionByNameOrId("tips")
				if err2 != nil {
					return err2
				}
				tip = core.NewRecord(col)
				tip.Set("user", userID)
				tip.Set("match", match.Id)
			}
			tip.Set("ftHome", ftH)
			tip.Set("ftAway", ftA)
			tip.Set("etHome", 0)
			tip.Set("etAway", 0)
			tip.Set("penWinner", "")
			tip.Set("advancer", "")
			tip.Set("firstTeam", firstTeamID)
			tip.Set("firstPlayer", playerName(firstPlayer))
			return app.Save(tip)
		}

		// ── User y8wc1ffk7780ck0 ────────────────────────────────────────────
		// Argentina 2–1 Algeria, first scorer: Lautaro Martínez (Argentina)
		if err := upsertGroupTip(
			"y8wc1ffk7780ck0",
			"Argentina", "Algeria",
			2, 1,
			"Lautaro Martínez", "Argentina",
		); err != nil {
			return err
		}

		// Austria 2–0 Jordan, first scorer: Marcel Sabitzer (Austria)
		if err := upsertGroupTip(
			"y8wc1ffk7780ck0",
			"Austria", "Jordan",
			2, 0,
			"Marcel Sabitzer", "Austria",
		); err != nil {
			return err
		}

		// ── User w3zm0gjgsex44g4 ──────────────────────────────────────────
		// Austria 3–1 Jordan, first scorer: Marko Arnautović (Austria)
		if err := upsertGroupTip(
			"w3zm0gjgsex44g4",
			"Austria", "Jordan",
			3, 1,
			"Marko Arnautović", "Austria",
		); err != nil {
			return err
		}

		// Remove turbo boost from tip bgugcb6a1ao14mu
		boostTip, err := app.FindRecordById("tips", "bgugcb6a1ao14mu")
		if err != nil {
			return err
		}
		boostTip.Set("turbo", false)
		if err := app.Save(boostTip); err != nil {
			return err
		}

		// Fix tips that have a firstPlayer set but no firstTeam.
		// Infer the team from the player's teamId field.
		finishedMatches, _ := app.FindRecordsByFilter("matches", "finalizedAt != ''", "", 0, 0)
		finishedIDs := make([]string, len(finishedMatches))
		for i, m := range finishedMatches {
			finishedIDs[i] = `"` + m.Id + `"`
		}
		if len(finishedIDs) == 0 {
			return scoring.Recompute(app)
		}
		matchFilter := "match = (" + strings.Join(finishedIDs, ",") + ")"
		allTips, _ := app.FindRecordsByFilter("tips", "firstPlayer != '' && firstTeam = '' && "+matchFilter, "", 0, 0)
		for _, tip := range allTips {
			playerID := tip.GetString("firstPlayer")
			player, err := app.FindRecordById("players", playerID)
			if err != nil {
				continue // firstPlayer may be a plain name string, not an ID — skip
			}
			tip.Set("firstTeam", player.GetString("teamId"))
			_ = app.Save(tip)
		}

		return scoring.Recompute(app)
	}, nil)
}
