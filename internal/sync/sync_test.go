package sync

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func testMatchRecord(stage string) *core.Record {
	collection := core.NewBaseCollection("matches")
	collection.Fields.Add(&core.TextField{Name: "stage", Max: 16})
	collection.Fields.Add(&core.TextField{Name: "status", Max: 16})
	collection.Fields.Add(&core.TextField{Name: "homeTeam", Max: 32})
	collection.Fields.Add(&core.TextField{Name: "awayTeam", Max: 32})
	collection.Fields.Add(&core.TextField{Name: "penWinner", Max: 32})
	collection.Fields.Add(&core.TextField{Name: "advancer", Max: 32})
	collection.Fields.Add(&core.NumberField{Name: "ftHome", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "ftAway", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "etHome", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "etAway", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "penHome", OnlyInt: true})
	collection.Fields.Add(&core.NumberField{Name: "penAway", OnlyInt: true})
	collection.Fields.Add(&core.DateField{Name: "finalizedAt"})

	record := core.NewRecord(collection)
	record.Set("stage", stage)
	record.Set("homeTeam", "home")
	record.Set("awayTeam", "away")
	return record
}

func TestApplyResultStoresFinishedGroupResult(t *testing.T) {
	record := testMatchRecord("group")

	applyResult(record, "finished", pi(2), pi(1), nil, nil, nil, nil)

	if record.GetString("status") != "finished" {
		t.Fatalf("status = %q, want finished", record.GetString("status"))
	}
	if record.GetInt("ftHome") != 2 || record.GetInt("ftAway") != 1 {
		t.Fatalf("full-time score = %d-%d, want 2-1", record.GetInt("ftHome"), record.GetInt("ftAway"))
	}
	if record.GetInt("etHome") != 0 || record.GetInt("etAway") != 0 || record.GetInt("penHome") != 0 || record.GetInt("penAway") != 0 {
		t.Fatalf("extra-time/penalty defaults were not cleared")
	}
	if record.GetDateTime("finalizedAt").Time().IsZero() {
		t.Fatal("finalizedAt was not set")
	}
	if record.GetString("advancer") != "" {
		t.Fatalf("group advancer = %q, want empty", record.GetString("advancer"))
	}
}

func TestApplyResultDerivesKnockoutAdvancerFromPenalties(t *testing.T) {
	record := testMatchRecord("FINAL")

	applyResult(record, "finished", pi(1), pi(1), pi(2), pi(2), pi(4), pi(3))

	if record.GetString("advancer") != "home" {
		t.Fatalf("advancer = %q, want home", record.GetString("advancer"))
	}
	if record.GetString("penWinner") != "home" {
		t.Fatalf("penWinner = %q, want home", record.GetString("penWinner"))
	}
}

func TestApplyResultDoesNotDeriveAdvancerBeforeFinished(t *testing.T) {
	record := testMatchRecord("R32")

	applyResult(record, "live", pi(2), pi(0), nil, nil, nil, nil)

	if record.GetString("advancer") != "" {
		t.Fatalf("advancer = %q, want empty while live", record.GetString("advancer"))
	}
	if !record.GetDateTime("finalizedAt").Time().IsZero() {
		t.Fatal("finalizedAt was set for a live match")
	}
}
