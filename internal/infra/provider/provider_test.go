package provider_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ancaeus/internal/domain"
	"ancaeus/internal/infra/provider"
)

func TestNew(t *testing.T) {
	t.Run("beacondb default", func(t *testing.T) {
		p, err := provider.New(provider.Config{})
		if err != nil {
			t.Fatal(err)
		}
		if p.Name() != "beacondb" {
			t.Fatalf("name=%q", p.Name())
		}
	})

	t.Run("apple", func(t *testing.T) {
		p, err := provider.New(provider.Config{Provider: "apple"})
		if err != nil {
			t.Fatal(err)
		}
		if p.Name() != "apple" {
			t.Fatalf("name=%q", p.Name())
		}
	})

	t.Run("google requires token file", func(t *testing.T) {
		_, err := provider.New(provider.Config{Provider: "google"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("google with token file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "token")
		if err := os.WriteFile(path, []byte("secret\n"), 0600); err != nil {
			t.Fatal(err)
		}
		p, err := provider.New(provider.Config{Provider: "google", GoogleGeoTokenFile: path})
		if err != nil {
			t.Fatal(err)
		}
		if p.Name() != "google" {
			t.Fatalf("name=%q", p.Name())
		}
	})

	t.Run("unknown", func(t *testing.T) {
		_, err := provider.New(provider.Config{Provider: "nope"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestBeaconDBLocate(t *testing.T) {
	aps := []domain.AccessPoint{{Name: "Home", MacAddress: "aa:aa:aa:aa:aa:01", SignalStrength: -50}}

	t.Run("ok annotates primary ap", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"location": map[string]float64{"lat": 42.1, "lng": 24.7},
				"accuracy": 30,
			})
		}))
		t.Cleanup(srv.Close)

		p, err := provider.New(provider.Config{Provider: "beacondb", BeaconDBURL: srv.URL})
		if err != nil {
			t.Fatal(err)
		}
		loc, err := p.Locate(t.Context(), aps)
		if err != nil {
			t.Fatal(err)
		}
		if loc.Latitude != 42.1 || loc.SSID != "Home" || loc.BSSID != "aa:aa:aa:aa:aa:01" {
			t.Fatalf("loc=%+v", loc)
		}
	})

	t.Run("not found", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}))
		t.Cleanup(srv.Close)

		p, err := provider.New(provider.Config{Provider: "beacondb", BeaconDBURL: srv.URL})
		if err != nil {
			t.Fatal(err)
		}
		_, err = p.Locate(t.Context(), aps)
		if !errors.Is(err, domain.ErrLocationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}
