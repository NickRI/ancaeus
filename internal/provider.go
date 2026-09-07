package internal

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// GeoProvider resolves a location from nearby Wi-Fi access points.
type GeoProvider interface {
	Name() string
	Locate(ctx context.Context, aps []AccessPoint) (*LookupResult, error)
}

type LookupResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	BSSID     string  `json:"bssid,omitempty"`
	Accuracy  float64 `json:"accuracy,omitempty"`
}

func (r LookupResult) Correct() bool {
	return r.Latitude != -180 && r.Longitude != -180
}

// ProviderConfig selects and configures a GeoProvider.
type ProviderConfig struct {
	Provider           string // beacondb | google | apple
	BeaconDBURL        string
	GoogleGeoTokenFile string
}

func NewGeoProvider(cfg ProviderConfig) (GeoProvider, error) {
	switch strings.ToLower(cfg.Provider) {
	case "", "beacondb":
		url := cfg.BeaconDBURL
		if url == "" {
			url = DefaultBeaconDBURL
		}

		return NewBeaconDBProvider(url), nil
	case "google":
		if cfg.GoogleGeoTokenFile == "" {
			return nil, fmt.Errorf("provider google requires -google-geo-token-file")
		}

		token, err := os.ReadFile(cfg.GoogleGeoTokenFile)
		if err != nil {
			return nil, err
		}

		return NewGoogleProvider(strings.TrimSpace(string(token))), nil
	case "apple":
		return NewAppleProvider(), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (beacondb|google|apple)", cfg.Provider)
	}
}
