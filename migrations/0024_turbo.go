package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		tips, err := app.FindCollectionByNameOrId("tips")
		if err != nil {
			return err
		}
		if tips.Fields.GetByName("turbo") != nil {
			return nil
		}
		tips.Fields.Add(&core.BoolField{Name: "turbo"})
		return app.Save(tips)
	}, func(app core.App) error {
		tips, err := app.FindCollectionByNameOrId("tips")
		if err != nil {
			return err
		}
		tips.Fields.RemoveByName("turbo")
		return app.Save(tips)
	})
}
