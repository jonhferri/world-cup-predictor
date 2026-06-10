package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Add firstTeam and firstPlayer fields to tips.
		tips, err := app.FindCollectionByNameOrId(nTips)
		if err != nil {
			return err
		}
		tipChanged := false
		if tips.Fields.GetByName("firstTeam") == nil {
			tips.Fields.Add(&core.TextField{Name: "firstTeam", Max: 50})
			tipChanged = true
		}
		if tips.Fields.GetByName("firstPlayer") == nil {
			tips.Fields.Add(&core.TextField{Name: "firstPlayer", Max: 100})
			tipChanged = true
		}
		if tipChanged {
			if err := app.Save(tips); err != nil {
				return err
			}
		}

		// Add firstTeamScorer and firstPlayerScorer fields to matches
		// (set by the admin once the match is played).
		matches, err := app.FindCollectionByNameOrId(nMatches)
		if err != nil {
			return err
		}
		matchChanged := false
		if matches.Fields.GetByName("firstTeamScorer") == nil {
			matches.Fields.Add(&core.TextField{Name: "firstTeamScorer", Max: 50})
			matchChanged = true
		}
		if matches.Fields.GetByName("firstPlayerScorer") == nil {
			matches.Fields.Add(&core.TextField{Name: "firstPlayerScorer", Max: 100})
			matchChanged = true
		}
		if matchChanged {
			if err := app.Save(matches); err != nil {
				return err
			}
		}

		return patchFirstScorerConfig(app, true)
	}, func(app core.App) error {
		if tips, err := app.FindCollectionByNameOrId(nTips); err == nil {
			tips.Fields.RemoveByName("firstTeam")
			tips.Fields.RemoveByName("firstPlayer")
			_ = app.Save(tips)
		}
		if matches, err := app.FindCollectionByNameOrId(nMatches); err == nil {
			matches.Fields.RemoveByName("firstTeamScorer")
			matches.Fields.RemoveByName("firstPlayerScorer")
			_ = app.Save(matches)
		}
		return patchFirstScorerConfig(app, false)
	})
}

func patchFirstScorerConfig(app core.App, add bool) error {
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
		if add {
			match["firstTeamScorer"] = 5
			match["firstPlayerScorer"] = 10
		} else {
			delete(match, "firstTeamScorer")
			delete(match, "firstPlayerScorer")
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
