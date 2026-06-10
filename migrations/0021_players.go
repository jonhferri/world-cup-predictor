package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const nPlayers = "players"

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId(nPlayers); err == nil {
			return nil // already exists
		}
		players := core.NewBaseCollection(nPlayers)
		players.ListRule = ptr("")
		players.ViewRule = ptr("")
		players.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
		players.Fields.Add(&core.TextField{Name: "position", Max: 10})
		players.Fields.Add(&core.TextField{Name: "teamId", Max: 50})
		return app.Save(players)
	}, func(app core.App) error {
		if col, err := app.FindCollectionByNameOrId(nPlayers); err == nil {
			return app.Delete(col)
		}
		return nil
	})
}
