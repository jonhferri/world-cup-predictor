package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return patchScoringV3(app, map[string]int{
			"tendency":  5,
			"exact":     10,
			"totalGoals": 5,
			"goalDiff":  5,
		})
	}, func(app core.App) error {
		return patchScoringV3(app, map[string]int{
			"tendency":  3,
			"exact":     1,
			"totalGoals": 1,
			"goalDiff":  1,
		})
	})
}

func patchScoringV3(app core.App, values map[string]int) error {
	records, err := app.FindRecordsByFilter(nScoringConfigs, "id != ''", "", 0, 0)
	if err != nil {
		return nil
	}
	for _, record := range records {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(record.GetString("config")), &cfg); err != nil {
			continue
		}
		match, _ := cfg["match"].(map[string]any)
		if match == nil {
			match = map[string]any{}
			cfg["match"] = match
		}
		for key, val := range values {
			match[key] = val
		}
		raw, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			continue
		}
		record.Set("config", string(raw))
		_ = app.Save(record)
	}
	return nil
}
