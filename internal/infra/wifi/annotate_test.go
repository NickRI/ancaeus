package wifi

import (
	"testing"

	"ancaeus/internal/domain"
)

func TestAnnotateLocation(t *testing.T) {
	primary := domain.AccessPoint{Name: "Primary", MacAddress: "aa:aa:aa:aa:aa:01"}
	other := domain.AccessPoint{Name: "Other", MacAddress: "bb:bb:bb:bb:bb:02"}
	aps := []domain.AccessPoint{primary, other}

	t.Run("matches bssid from provider", func(t *testing.T) {
		loc := &domain.Location{Latitude: 1, Longitude: 2, BSSID: "bb:bb:bb:bb:bb:02"}
		annotateLocation(loc, aps, primary)
		if loc.SSID != "Other" || loc.BSSID != "bb:bb:bb:bb:bb:02" {
			t.Fatalf("loc=%+v", loc)
		}
	})

	t.Run("pads apple-style bssid", func(t *testing.T) {
		loc := &domain.Location{Latitude: 1, Longitude: 2, BSSID: "bb:bb:bb:bb:bb:2"}
		annotateLocation(loc, aps, primary)
		if loc.SSID != "Other" || loc.BSSID != "bb:bb:bb:bb:bb:02" {
			t.Fatalf("loc=%+v", loc)
		}
	})

	t.Run("falls back to primary when unknown bssid", func(t *testing.T) {
		loc := &domain.Location{Latitude: 1, Longitude: 2, BSSID: "12:9:d:54:1f:bb"}
		annotateLocation(loc, aps, primary)
		if loc.SSID != "Primary" || loc.BSSID != "12:09:0d:54:1f:bb" {
			t.Fatalf("loc=%+v", loc)
		}
	})

	t.Run("empty bssid uses primary mac", func(t *testing.T) {
		loc := &domain.Location{Latitude: 1, Longitude: 2}
		annotateLocation(loc, aps, primary)
		if loc.SSID != "Primary" || loc.BSSID != "aa:aa:aa:aa:aa:01" {
			t.Fatalf("loc=%+v", loc)
		}
	})
}
