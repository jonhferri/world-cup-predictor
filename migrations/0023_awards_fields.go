package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		forecasts, err := app.FindCollectionByNameOrId(nForecasts)
		if err != nil {
			return err
		}
		changed := false
		for _, name := range []string{
			"goldenBootPlayerName",
			"goldenBallPlayer",
			"goldenGlovePlayer",
			"bestYoungPlayer",
			"mostAssistsPlayer",
		} {
			if forecasts.Fields.GetByName(name) == nil {
				forecasts.Fields.Add(&core.TextField{Name: name, Max: 200})
				changed = true
			}
		}
		if changed {
			return app.Save(forecasts)
		}
		return nil
	}, func(app core.App) error {
		forecasts, err := app.FindCollectionByNameOrId(nForecasts)
		if err != nil {
			return err
		}
		for _, name := range []string{
			"goldenBootPlayerName",
			"goldenBallPlayer",
			"goldenGlovePlayer",
			"bestYoungPlayer",
			"mostAssistsPlayer",
		} {
			forecasts.Fields.RemoveByName(name)
		}
		return app.Save(forecasts)
	})
}
