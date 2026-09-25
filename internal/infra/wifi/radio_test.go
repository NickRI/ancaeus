package wifi

import (
	"testing"

	"ancaeus/internal/domain"
)

func TestMergeAPs(t *testing.T) {
	base := map[string]domain.AccessPoint{
		"aa:aa:aa:aa:aa:01": {MacAddress: "aa:aa:aa:aa:aa:01", Name: "old", SignalStrength: -80},
	}
	got := mergeAPs(base,
		domain.AccessPoint{MacAddress: "AA-AA-AA-AA-AA-01", Name: "new", InUse: true, SignalStrength: -40},
		domain.AccessPoint{MacAddress: "bb:bb:bb:bb:bb:02", Name: "other", SignalStrength: -50},
		domain.AccessPoint{MacAddress: "cc:cc:cc:cc:cc:03", Name: "x_nomap"},
	)

	if _, ok := base["bb:bb:bb:bb:bb:02"]; ok {
		t.Fatal("mergeAPs mutated input map")
	}
	a := got["aa:aa:aa:aa:aa:01"]
	if !a.InUse || a.Name != "new" || a.SignalStrength != -40 {
		t.Fatalf("merged=%+v", a)
	}
	if _, ok := got["bb:bb:bb:bb:bb:02"]; !ok {
		t.Fatal("missing bb")
	}
	if _, ok := got["cc:cc:cc:cc:cc:03"]; ok {
		t.Fatal("nomap should be dropped")
	}
}

func TestSignalStrength(t *testing.T) {
	station := map[string]int{"aa:aa:aa:aa:aa:01": -42}
	if got := signalStrength(station, "aa:aa:aa:aa:aa:01", -5500); got != -42 {
		t.Fatalf("station prefer: got=%d", got)
	}
	if got := signalStrength(station, "bb:bb:bb:bb:bb:02", -5500); got != -55 {
		t.Fatalf("bss mBm: got=%d", got)
	}
}
