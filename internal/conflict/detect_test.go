package conflict_test

import (
	"errors"
	"testing"

	"task278-broadcastslot/internal/conflict"
	"task278-broadcastslot/internal/model"
)

func TestDetectCycleRejects(t *testing.T) {
	existing := []model.SourceCitation{
		{FromRef: "a", ToRef: "b"},
		{FromRef: "b", ToRef: "c"},
	}
	err := conflict.DetectCycle(existing, "c", "a")
	if !errors.Is(err, model.ErrSourceCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestDetectCycleAllowsDAG(t *testing.T) {
	existing := []model.SourceCitation{{FromRef: "clip:1", ToRef: "entry:1"}}
	if err := conflict.DetectCycle(existing, "clip:1", "entry:2"); err != nil {
		t.Fatal(err)
	}
}

func TestAdDelayedConflict(t *testing.T) {
	slots := []conflict.Window{{
		EntryID: 2, PageID: "page-B", Callsign: "SH-RADIO",
		PrintedStart: 72120000, UTCStart: 1000, UTCEnd: 2000,
	}}
	ads := []model.NewspaperAd{{PageID: "page-B", PrintedStartMS: 72840000}}
	out := conflict.Detect(slots, ads)
	if len(out) != 1 || out[0].Kind != model.ConflictAdDelayed {
		t.Fatalf("expected ad_delayed, got %+v", out)
	}
}

func TestCallsignOverlap(t *testing.T) {
	slots := []conflict.Window{
		{EntryID: 1, Callsign: "X", UTCStart: 0, UTCEnd: 100},
		{EntryID: 2, Callsign: "X", UTCStart: 50, UTCEnd: 150},
	}
	out := conflict.Detect(slots, nil)
	found := false
	for _, c := range out {
		if c.Kind == model.ConflictCallsignOverlap {
			found = true
		}
	}
	if !found {
		t.Fatal("expected callsign_overlap")
	}
}
