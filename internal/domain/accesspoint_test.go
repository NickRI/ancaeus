package domain_test

import (
	"testing"

	"ancaeus/internal/domain"
)

func TestFilterAccessPointsNomap(t *testing.T) {
	in := []domain.AccessPoint{
		{Name: "Home", MacAddress: "AA:BB:CC:DD:EE:01", SignalStrength: -40},
		{Name: "Guest_nomap", MacAddress: "AA:BB:CC:DD:EE:02", SignalStrength: -50},
		{Name: "Cafe", MacAddress: "aa-bb-cc-dd-ee-01", SignalStrength: -30},
	}
	out := domain.FilterAccessPoints(in)
	if len(out) != 1 {
		t.Fatalf("got %d aps, want 1 (dedupe+nomap)", len(out))
	}
	if out[0].MacAddress != "aa:bb:cc:dd:ee:01" {
		t.Fatalf("mac=%q", out[0].MacAddress)
	}
}

func TestNormalizeMAC(t *testing.T) {
	got := domain.NormalizeMAC("12:9:d:54:1f:bb")
	if got != "12:09:0d:54:1f:bb" {
		t.Fatalf("got=%q", got)
	}
}

func TestLocationCorrect(t *testing.T) {
	ok := domain.Location{Latitude: 55.7, Longitude: 37.6}
	bad := domain.Location{Latitude: -180, Longitude: -180}
	if !ok.Correct() || bad.Correct() {
		t.Fatal("Correct() mismatch")
	}
}

func TestSortedByPriority(t *testing.T) {
	in := []domain.AccessPoint{
		{MacAddress: "aa:aa:aa:aa:aa:01", SignalStrength: -50},
		{MacAddress: "bb:bb:bb:bb:bb:02", InUse: true, SignalStrength: -70},
		{MacAddress: "cc:cc:cc:cc:cc:03", SignalStrength: -40},
	}
	got := domain.SortedByPriority(in, true)
	if len(got) != 3 || !got[0].InUse || got[1].MacAddress != "cc:cc:cc:cc:cc:03" {
		t.Fatalf("got=%+v", got)
	}
}
