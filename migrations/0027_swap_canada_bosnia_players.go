package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The players2026.csv had Canada's and Bosnia's player lists in swapped columns,
// so all seeded players ended up assigned to the wrong team. This migration
// swaps their teamId values to correct the association.
func init() {
	m.Register(func(app core.App) error {
		return swapPlayerTeams(app, "Canada", "Bosnia & Herzegovina")
	}, func(app core.App) error {
		return swapPlayerTeams(app, "Canada", "Bosnia & Herzegovina")
	})
}

func swapPlayerTeams(app core.App, teamA, teamB string) error {
	recA, err := app.FindFirstRecordByFilter("teams", "name = {:n}", map[string]any{"n": teamA})
	if err != nil {
		return nil // team doesn't exist yet; nothing to swap
	}
	recB, err := app.FindFirstRecordByFilter("teams", "name = {:n}", map[string]any{"n": teamB})
	if err != nil {
		return nil
	}
	idA, idB := recA.Id, recB.Id

	// Load both groups first so we're not updating records mid-iteration.
	playersA, _ := app.FindRecordsByFilter("players", "teamId = {:t}", "", 0, 0, map[string]any{"t": idA})
	playersB, _ := app.FindRecordsByFilter("players", "teamId = {:t}", "", 0, 0, map[string]any{"t": idB})

	for _, p := range playersA {
		p.Set("teamId", idB)
		if err := app.Save(p); err != nil {
			return err
		}
	}
	for _, p := range playersB {
		p.Set("teamId", idA)
		if err := app.Save(p); err != nil {
			return err
		}
	}
	return nil
}
