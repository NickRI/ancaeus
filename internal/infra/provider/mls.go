package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"ancaeus/internal/domain"
	"ancaeus/internal/port"
)

const defaultGoogleGeoURL = "https://www.googleapis.com/geolocation/v1/geolocate"

type mlsRequest struct {
	ConsiderIP       bool             `json:"considerIp"`
	WifiAccessPoints []mlsAccessPoint `json:"wifiAccessPoints"`
}

type mlsAccessPoint struct {
	MacAddress     string `json:"macAddress"`
	SignalStrength int    `json:"signalStrength,omitempty"`
}

type mlsResponse struct {
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	Accuracy float64 `json:"accuracy"`
}

func locateMLS(ctx context.Context, url, userAgent, apiKey string, aps []domain.AccessPoint) (*domain.Location, error) {
	if len(aps) == 0 {
		return nil, fmt.Errorf("no wifi access points")
	}

	body := mlsRequest{ConsiderIP: false}
	for _, ap := range aps {
		mac := domain.NormalizeMAC(ap.MacAddress)
		if mac == "" {
			continue
		}
		body.WifiAccessPoints = append(body.WifiAccessPoints, mlsAccessPoint{
			MacAddress:     mac,
			SignalStrength: ap.SignalStrength,
		})
	}
	if len(body.WifiAccessPoints) == 0 {
		return nil, fmt.Errorf("no wifi access points with mac")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	reqURL := url
	if apiKey != "" {
		reqURL = url + "?key=" + apiKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", domain.ErrLocationNotFound, strings.TrimSpace(string(raw)))
		}
		return nil, fmt.Errorf("geolocate %s: %s: %s", url, resp.Status, strings.TrimSpace(string(raw)))
	}

	var out mlsResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	loc := &domain.Location{
		Latitude:  out.Location.Lat,
		Longitude: out.Location.Lng,
		Accuracy:  out.Accuracy,
		SSID:      aps[0].Name,
		BSSID:     domain.NormalizeMAC(aps[0].MacAddress),
	}
	if !loc.Correct() {
		return nil, fmt.Errorf("%w: invalid coordinates", domain.ErrLocationNotFound)
	}
	return loc, nil
}

type beaconDB struct{ url string }

func newBeaconDB(url string) *beaconDB { return &beaconDB{url: url} }

func (p *beaconDB) Name() string { return "beacondb" }

func (p *beaconDB) Locate(ctx context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
	return locateMLS(ctx, p.url, "ancaeus/0.1 (https://github.com/NickRI/ancaeus)", "", aps)
}

var _ port.GeoProvider = (*beaconDB)(nil)

type google struct{ token string }

func newGoogle(token string) *google { return &google{token: token} }

func (p *google) Name() string { return "google" }

func (p *google) Locate(ctx context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
	return locateMLS(ctx, defaultGoogleGeoURL, "ancaeus/0.1", p.token, aps)
}

var _ port.GeoProvider = (*google)(nil)
