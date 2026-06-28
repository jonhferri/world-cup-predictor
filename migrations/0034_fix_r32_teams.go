package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Corrects homeTeam/awayTeam for Round of 32 matches (73–88) to match the
// actual bracket that emerged after the group stage.
func init() {
	m.Register(func(app core.App) error {
		// Authoritative R32 bracket from openfootball 2026/worldcup.json.
		r32 := []struct {
			num   int
			home  string
			away  string
		}{
			{73, "South Africa", "Canada"},
			{74, "Germany", "Paraguay"},
			{75, "Netherlands", "Morocco"},
			{76, "Brazil", "Japan"},
			{77, "France", "Sweden"},
			{78, "Ivory Coast", "Norway"},
			{79, "Mexico", "Ecuador"},
			{80, "England", "DR Congo"},
			{81, "USA", "Bosnia & Herzegovina"},
			{82, "Belgium", "Senegal"},
			{83, "Portugal", "Croatia"},
			{84, "Spain", "Austria"},
			{85, "Switzerland", "Algeria"},
			{86, "Argentina", "Cape Verde"},
			{87, "Colombia", "Ghana"},
			{88, "Australia", "Egypt"},
		}

		teamID := func(name string) (string, error) {
			r, err := app.FindFirstRecordByFilter("teams", "name = {:n}", map[string]any{"n": name})
			if err != nil || r == nil {
				return "", fmt.Errorf("team not found: %s", name)
			}
			return r.Id, nil
		}

		for _, fix := range r32 {
			extID := fmt.Sprintf("WC2026-K-%d", fix.num)
			rec, err := app.FindFirstRecordByFilter("matches", "extId = {:e}", map[string]any{"e": extID})
			if err != nil || rec == nil {
				continue // match not present in this environment
			}

			homeID, err := teamID(fix.home)
			if err != nil {
				return err
			}
			awayID, err := teamID(fix.away)
			if err != nil {
				return err
			}

			changed := false
			if rec.GetString("homeTeam") != homeID {
				rec.Set("homeTeam", homeID)
				changed = true
			}
			if rec.GetString("awayTeam") != awayID {
				rec.Set("awayTeam", awayID)
				changed = true
			}
			if changed {
				if err := app.Save(rec); err != nil {
					return fmt.Errorf("save match %s: %w", extID, err)
				}
			}
		}
		return nil
	}, nil)
}
