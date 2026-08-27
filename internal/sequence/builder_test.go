package sequence_test

import (
	"testing"

	"task278-broadcastslot/internal/sequence"
)

func TestBuilderCopiesResult(t *testing.T) {
	var b sequence.Builder
	in := []sequence.Item{{
		EntryID: 1, Transmitter: "TX-1", Callsign: "A",
		UTCStart: 100, UTCEnd: 200,
		Sources: []string{"clip:1"},
	}}
	first := b.Build(in)
	first[0].Sources[0] = "mutated"
	first[0].UTCStart = 9999

	second := b.Build(in)
	if second[0].Sources[0] != "clip:1" {
		t.Fatalf("builder reused sources slice: %v", second[0].Sources)
	}
	if second[0].UTCStart != 100 {
		t.Fatalf("builder reused slot data: %d", second[0].UTCStart)
	}
}

func TestTransmitterOverlapMarksInfeasible(t *testing.T) {
	var b sequence.Builder
	in := []sequence.Item{
		{EntryID: 1, Transmitter: "TX-1", UTCStart: 0, UTCEnd: 100},
		{EntryID: 2, Transmitter: "TX-1", UTCStart: 50, UTCEnd: 150},
	}
	slots := b.Build(in)
	if slots[0].Feasible && slots[1].Feasible {
		t.Fatal("expected one slot infeasible due to overlap")
	}
}
