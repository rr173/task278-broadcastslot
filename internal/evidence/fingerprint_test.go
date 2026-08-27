package evidence_test

import (
	"errors"
	"testing"

	"task278-broadcastslot/internal/evidence"
	"task278-broadcastslot/internal/model"
)

func TestFingerprintDeterministic(t *testing.T) {
	a := evidence.Fingerprint("晚间新闻", "SH-RADIO", 72000000, 73800000, "page-A")
	b := evidence.Fingerprint("晚间新闻", "SH-RADIO", 72000000, 73800000, "page-A")
	if a != b || len(a) != 16 {
		t.Fatalf("fingerprint=%q len=%d", a, len(a))
	}
}

func TestFingerprintIdempotentMatch(t *testing.T) {
	in := model.ProgramEntry{
		Title: "T", Callsign: "C", PrintedStartMS: 1, PrintedEndMS: 2, PageID: "p", Transmitter: "tx",
	}
	if err := evidence.Fill(&in); err != nil {
		t.Fatal(err)
	}
	existing := in
	res, err := evidence.Resolve(&existing, in)
	if err != nil || res.ID == 0 && res.Fingerprint != in.Fingerprint {
		t.Fatalf("resolve: %v %+v", err, res)
	}
}

func TestDuplicateFingerprintDifferentFields(t *testing.T) {
	in := model.ProgramEntry{
		Title: "T2", Callsign: "C", PrintedStartMS: 1, PrintedEndMS: 2, PageID: "p", Transmitter: "tx",
	}
	if err := evidence.Fill(&in); err != nil {
		t.Fatal(err)
	}
	existing := model.ProgramEntry{
		Fingerprint: in.Fingerprint, Title: "T1", Callsign: "C",
		PrintedStartMS: 1, PrintedEndMS: 2, PageID: "p", Transmitter: "tx",
	}
	_, err := evidence.Resolve(&existing, in)
	if !errors.Is(err, model.ErrDuplicateFingerprint) {
		t.Fatalf("expected duplicate fingerprint, got %v", err)
	}
}
