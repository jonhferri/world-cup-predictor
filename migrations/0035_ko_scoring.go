package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/oyvhov/world-cup-pool/internal/scoring"
)

// Adds KO-specific FT/ET scoring fields to all existing scoring_configs
// and triggers a full recompute so existing KO tips get rescored.
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
			for _, key := range []string{
				"koFtGoalDiff", "koFtExactHome", "koFtExactAway", "koFtExact",
				"koEtGoalDiff", "koEtExactHome", "koEtExactAway", "koEtExact",
				"koAdvancer",
			} {
				if _, exists := match[key]; !exists {
					match[key] = 5
				}
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
