package clockfix_test

import (
	"errors"
	"testing"

	"task278-broadcastslot/internal/clockfix"
	"task278-broadcastslot/internal/model"
)

func TestCorrectFormulaAsiaShanghai(t *testing.T) {
	const printed = int64(72000000)
	const ref = int64(72000000)
	res, err := clockfix.CorrectOne(printed, "Asia/Shanghai", 120, ref, true)
	if err != nil {
		t.Fatal(err)
	}
	want := printed - 8*3600000
	if res != want {
		t.Fatalf("utc=%d want %d", res, want)
	}
}

func TestCorrectDriftApplied(t *testing.T) {
	const printed = int64(72120000)
	const ref = int64(72000000)
	res, err := clockfix.CorrectOne(printed, "UTC", 120, ref, true)
	if err != nil {
		t.Fatal(err)
	}
	want := clockfix.Apply(printed, 0, ref, 120)
	if res != want {
		t.Fatalf("utc=%d want %d", res, want)
	}
}

func TestUnknownTimezone(t *testing.T) {
	_, err := clockfix.CorrectOne(1000, "Europe/Berlin", 0, 0, false)
	if !errors.Is(err, model.ErrUnknownTimezone) {
		t.Fatalf("expected ErrUnknownTimezone, got %v", err)
	}
}

func TestSlotInvertedRejected(t *testing.T) {
	_, _, err := clockfix.CorrectSlot(5000, 5000, "UTC", 0, 0, false)
	if !errors.Is(err, model.ErrSlotInverted) {
		t.Fatalf("expected ErrSlotInverted, got %v", err)
	}
}
