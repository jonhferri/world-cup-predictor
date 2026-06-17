package migrations

import (
	"strings"

	"github.com/pocketbase/dbx"
  "github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/oyvhov/world-cup-pool/internal/scoring"
)

// Backfills firstTeam on tips where firstPlayer is set but firstTeam is empty,
// for all finished matches. Infers the team from the players collection.
func init() {
  m.Register(func(app core.App) error {
		finishedMatches, _ := app.FindRecordsByFilter("matches", "status = 'finished'", "", 0, 0)
		if len(finishedMatches) == 0 {
			return nil
		}
		finishedIDs := make([]string, len(finishedMatches))
		for i, match := range finishedMatches {
      finishedIDs[i] = `"` + match.Id + `"`
		}
		matchFilter := "match = (" + strings.Join(finishedIDs, ",") + ")"
		allTips, _ := app.FindRecordsByFilter("tips", "firstPlayer != '' && firstTeam = '' && "+matchFilter, "", 0, 0)
		for _, tip := range allTips {
			player, err := app.FindFirstRecordByFilter("players", "name = {:n}", map[string]any{"n": tip.GetString("firstPlayer")})
			if err != nil {
				continue
			}
			_, _ = app.DB().NewQuery("UPDATE tips SET firstTeam={:t} WHERE id={:id}").
				Bind(dbx.Params{"t": player.GetString("teamId"), "id": tip.Id}).
				Execute()
		}
		return scoring.Recompute(app)
	}, nil)
}
