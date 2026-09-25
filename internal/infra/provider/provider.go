package provider

import (
	"fmt"
	"os"
	"strings"

	"ancaeus/internal/port"
)

const DefaultBeaconDBURL = "https://api.beacondb.net/v1/geolocate"

// Config selects and configures a GeoProvider.
type Config struct {
	Provider           string // beacondb | google | apple
	BeaconDBURL        string
	GoogleGeoTokenFile string
}

// New returns a GeoProvider for cfg.
func New(cfg Config) (port.GeoProvider, error) {
	switch strings.ToLower(cfg.Provider) {
	case "", "beacondb":
		url := cfg.BeaconDBURL
		if url == "" {
			url = DefaultBeaconDBURL
		}
		return newBeaconDB(url), nil
	case "google":
		if cfg.GoogleGeoTokenFile == "" {
			return nil, fmt.Errorf("provider google requires -google-geo-token-file")
		}
		token, err := os.ReadFile(cfg.GoogleGeoTokenFile)
		if err != nil {
			return nil, err
		}
		return newGoogle(strings.TrimSpace(string(token))), nil
	case "apple":
		return newApple(), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (beacondb|google|apple)", cfg.Provider)
	}
}
