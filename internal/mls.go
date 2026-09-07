package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const DefaultBeaconDBURL = "https://api.beacondb.net/v1/geolocate"
const DefaultGoogleGeoURL = "https://www.googleapis.com/geolocation/v1/geolocate"

type mlsRequest struct {
	ConsiderIP       bool              `json:"considerIp"`
	WifiAccessPoints []mlsAccessPoint  `json:"wifiAccessPoints"`
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

func locateMLS(ctx context.Context, url, userAgent, apiKey string, aps []AccessPoint) (*LookupResult, error) {
	if len(aps) == 0 {
		return nil, fmt.Errorf("no wifi access points")
	}

	body := mlsRequest{ConsiderIP: false}
	for _, ap := range aps {
		mac := strings.ReplaceAll(ap.MacAddress, "-", ":")
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
			return nil, fmt.Errorf("%w: %s", ErrLocationNotFound, strings.TrimSpace(string(raw)))
		}
		return nil, fmt.Errorf("geolocate %s: %s: %s", url, resp.Status, strings.TrimSpace(string(raw)))
	}

	var out mlsResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	result := &LookupResult{
		Latitude:  out.Location.Lat,
		Longitude: out.Location.Lng,
		Accuracy:  out.Accuracy,
	}
	if !result.Correct() {
		return nil, fmt.Errorf("%w: invalid coordinates", ErrLocationNotFound)
	}
	return result, nil
}

type BeaconDBProvider struct {
	url string
}

func NewBeaconDBProvider(url string) *BeaconDBProvider {
	return &BeaconDBProvider{url: url}
}

func (p *BeaconDBProvider) Name() string { return "beacondb" }

func (p *BeaconDBProvider) Locate(ctx context.Context, aps []AccessPoint) (*LookupResult, error) {
	return locateMLS(ctx, p.url, "ancaeus/0.1 (https://github.com/NickRI/ancaeus)", "", aps)
}

type GoogleProvider struct {
	token string
}

func NewGoogleProvider(token string) *GoogleProvider {
	return &GoogleProvider{token: token}
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) Locate(ctx context.Context, aps []AccessPoint) (*LookupResult, error) {
	return locateMLS(ctx, DefaultGoogleGeoURL, "ancaeus/0.1", p.token, aps)
}
