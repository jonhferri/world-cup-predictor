package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return patchTiebreaker(app, "goalDiffDeviation", "forecastPoints")
	}, func(app core.App) error {
		return patchTiebreaker(app, "forecastPoints", "goalDiffDeviation")
	})
}

func patchTiebreaker(app core.App, from, to string) error {
	records, err := app.FindRecordsByFilter(nScoringConfigs, "id != ''", "", 0, 0)
	if err != nil {
		return nil
	}
	for _, record := range records {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(record.GetString("config")), &cfg); err != nil {
			continue
		}
		tbs, _ := cfg["tiebreakers"].([]any)
		changed := false
		for i, tb := range tbs {
			if s, ok := tb.(string); ok && s == from {
				tbs[i] = to
				changed = true
			}
		}
		if !changed {
			continue
		}
		cfg["tiebreakers"] = tbs
		raw, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			continue
		}
		record.Set("config", string(raw))
		_ = app.Save(record)
	}
	return nil
}
