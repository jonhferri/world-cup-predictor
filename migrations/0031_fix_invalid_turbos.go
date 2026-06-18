package migrations

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/oyvhov/world-cup-pool/internal/scoring"
)

func init() {
	m.Register(func(app core.App) error {
		return fixInvalidTurbos(app)
	}, nil)
}

// tipBucket returns the turbo-slot key for a tip, mirroring the logic in
// internal/tips/tips.go so the same rules apply during cleanup.
func tipBucket(matchRoundLabel, matchStage string) string {
	if matchStage != "group" {
		return matchStage
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(matchRoundLabel, "Matchday "))
	switch {
	case n >= 14:
		return "group-3"
	case n >= 8:
		return "group-2"
	default:
		return "group-1"
	}
}

// fixInvalidTurbos finds every (user, bucket) pair with more than one turbo tip
// and clears ALL turbos in that pair. Raw SQL bypasses the lock hook so
// finished-match tips can be corrected. Recompute is called at the end so
// leaderboard scores reflect the corrected data.
func fixInvalidTurbos(app core.App) error {
	turboTips, err := app.FindRecordsByFilter("tips", "turbo = true", "", 0, 0)
	if err != nil || len(turboTips) == 0 {
		return nil
	}

	// Build match-ID → (roundLabel, stage) index.
	type matchMeta struct{ roundLabel, stage string }
	matchCache := map[string]matchMeta{}
	for _, t := range turboTips {
		mid := t.GetString("match")
		if _, ok := matchCache[mid]; ok {
			continue
		}
		if rec, err := app.FindRecordById("matches", mid); err == nil {
			matchCache[mid] = matchMeta{
				roundLabel: rec.GetString("roundLabel"),
				stage:      rec.GetString("stage"),
			}
		}
	}

	// Group tip IDs by (user, bucket).
	type key struct{ user, bucket string }
	byKey := map[key][]string{}
	for _, t := range turboTips {
		mm, ok := matchCache[t.GetString("match")]
		if !ok {
			continue
		}
		k := key{t.GetString("user"), tipBucket(mm.roundLabel, mm.stage)}
		byKey[k] = append(byKey[k], t.Id)
	}

	// Collect tip IDs to clear (any bucket with more than one turbo).
	var toFix []string
	for _, ids := range byKey {
		if len(ids) > 1 {
			toFix = append(toFix, ids...)
		}
	}
	if len(toFix) == 0 {
		return nil
	}

	// Raw SQL update — bypasses OnRecordUpdate hooks so locked matches can be
	// corrected without triggering the "match is locked" error.
	for _, id := range toFix {
		if _, err := app.DB().NewQuery(
			"UPDATE tips SET turbo={:t} WHERE id={:id}",
		).Bind(dbx.Params{"t": 0, "id": id}).Execute(); err != nil {
			return fmt.Errorf("fix turbo tip %s: %w", id, err)
		}
	}

	// Rebuild all match_scores and forecast_scores with corrected turbo flags.
	return scoring.Recompute(app)
}
