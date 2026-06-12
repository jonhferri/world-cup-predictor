package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return patchScoringV3(app, map[string]int{"exact": 5})
	}, func(app core.App) error {
		return patchScoringV3(app, map[string]int{"exact": 10})
	})
}
