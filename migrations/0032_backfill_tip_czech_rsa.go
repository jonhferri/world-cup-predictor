package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Backfills a 1-1 tip for w3zm0gjgsex44g4 on Czech Republic vs South Africa
// (match 9byg3lfeity8t3u): Czech Republic first, Patrik Schick first scorer.
func init() {
	m.Register(func(app core.App) error {
		const userID = "w3zm0gjgsex44g4"
		const matchID = "9byg3lfeity8t3u"

		if _, err := app.FindRecordById("users", userID); err != nil {
			return nil // not present in this environment; skip
		}

		czechTeam, err := app.FindFirstRecordByFilter("teams", "name = {:n}", map[string]any{"n": "Czech Republic"})
		if err != nil {
			return nil
		}

		// Resolve Patrik Schick from the players collection.
		firstPlayerVal := "Patrik Schick"
		if p, err := app.FindFirstRecordByFilter("players", "name = {:n}", map[string]any{"n": "Patrik Schick"}); err == nil {
			firstPlayerVal = p.Id
		}

		tip, err := app.FindFirstRecordByFilter("tips",
			"user = {:u} && match = {:m}",
			map[string]any{"u": userID, "m": matchID})
		if err != nil {
			col, err2 := app.FindCollectionByNameOrId("tips")
			if err2 != nil {
				return err2
			}
			tip = core.NewRecord(col)
			tip.Set("user", userID)
			tip.Set("match", matchID)
		}
		tip.Set("ftHome", 1)
		tip.Set("ftAway", 1)
		tip.Set("etHome", 0)
		tip.Set("etAway", 0)
		tip.Set("penWinner", "")
		tip.Set("advancer", "")
		tip.Set("firstTeam", czechTeam.Id)
		tip.Set("firstPlayer", firstPlayerVal)
		return app.Save(tip)
	}, nil)
}
