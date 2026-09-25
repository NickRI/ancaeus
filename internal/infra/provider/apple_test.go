package provider

import (
	"testing"

	"ancaeus/internal/domain"
)

func TestPickAppleLocation(t *testing.T) {
	ap := domain.AccessPoint{Name: "TP-Link", MacAddress: "f0:09:0d:54:1f:bb"}

	t.Run("exact match pads apple bssid", func(t *testing.T) {
		lookups := []domain.Location{
			{Latitude: -180, Longitude: -180, BSSID: "aa:aa:aa:aa:aa:01"},
			{Latitude: 42.1, Longitude: 24.7, BSSID: "f0:9:d:54:1f:bb"},
		}
		got := pickAppleLocation(ap, lookups)
		if got == nil || got.SSID != "TP-Link" || got.BSSID != "f0:09:0d:54:1f:bb" {
			t.Fatalf("got=%+v", got)
		}
		if got.Latitude != 42.1 {
			t.Fatalf("lat=%v", got.Latitude)
		}
	})

	t.Run("neighbor fallback attributes queried ap", func(t *testing.T) {
		lookups := []domain.Location{
			{Latitude: 42.1, Longitude: 24.7, BSSID: "12:9:d:54:1f:bb"},
		}
		got := pickAppleLocation(ap, lookups)
		if got == nil || got.SSID != "TP-Link" || got.BSSID != "f0:09:0d:54:1f:bb" {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("no correct coords", func(t *testing.T) {
		lookups := []domain.Location{
			{Latitude: -180, Longitude: -180, BSSID: "f0:09:0d:54:1f:bb"},
		}
		if got := pickAppleLocation(ap, lookups); got != nil {
			t.Fatalf("got=%+v", got)
		}
	})
}
