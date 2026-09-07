package internal

import (
	"strings"
	"testing"
)

func TestFilterAccessPointsNomap(t *testing.T) {
	in := []AccessPoint{
		{Name: "Home", MacAddress: "AA:BB:CC:DD:EE:01", SignalStrength: -40},
		{Name: "Guest_nomap", MacAddress: "AA:BB:CC:DD:EE:02", SignalStrength: -50},
		{Name: "Cafe", MacAddress: "aa-bb-cc-dd-ee-01", SignalStrength: -30},
	}
	out := filterAccessPoints(in)
	if len(out) != 1 {
		t.Fatalf("got %d aps, want 1 (dedupe+nomap)", len(out))
	}
	if out[0].MacAddress != "aa:bb:cc:dd:ee:01" {
		t.Fatalf("mac=%q", out[0].MacAddress)
	}
}

func TestCacheKeyStable(t *testing.T) {
	a := []AccessPoint{{MacAddress: "bb:bb:bb:bb:bb:bb"}, {MacAddress: "aa:aa:aa:aa:aa:aa"}}
	b := []AccessPoint{{MacAddress: "AA:AA:AA:AA:AA:AA"}, {MacAddress: "BB:BB:BB:BB:BB:BB"}}
	if cacheKey(a) != cacheKey(b) {
		t.Fatalf("%q vs %q", cacheKey(a), cacheKey(b))
	}
	if !strings.HasPrefix(cacheKey(a), "aa:") {
		t.Fatalf("expected sorted key, got %q", cacheKey(a))
	}
}

func TestLookupResultCorrect(t *testing.T) {
	ok := LookupResult{Latitude: 55.7, Longitude: 37.6}
	bad := LookupResult{Latitude: -180, Longitude: -180}
	if !ok.Correct() || bad.Correct() {
		t.Fatal("Correct() mismatch")
	}
}
