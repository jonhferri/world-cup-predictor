package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/oyvhov/world-cup-pool/internal/scoring"
)

// Adds koFtTendency + koEtTendency fields to all existing scoring_configs
// and bumps koAdvancer from 5 → 10, then triggers a full recompute.
func init() {
	m.Register(func(app core.App) error {
		records, err := app.FindRecordsByFilter(nScoringConfigs, "id != ''", "", 0, 0)
		if err != nil {
			return err
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
			// Add new tendency fields if absent.
			for _, key := range []string{"koFtTendency", "koEtTendency"} {
				if _, exists := match[key]; !exists {
					match[key] = 5
				}
			}
			// Bump koAdvancer to 10 if it's still at the old default of 5.
			if v, _ := match["koAdvancer"].(float64); v == 5 {
				match["koAdvancer"] = 10
			}
			raw, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				continue
			}
			record.Set("config", string(raw))
			_ = app.Save(record)
		}
		return scoring.Recompute(app)
	}, nil)
}
