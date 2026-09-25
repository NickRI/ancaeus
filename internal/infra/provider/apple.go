package provider

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"ancaeus/internal/bssidapple"
	"ancaeus/internal/domain"
	"ancaeus/internal/port"

	"google.golang.org/protobuf/proto"
)

type apple struct{}

func newApple() *apple { return &apple{} }

func (p *apple) Name() string { return "apple" }

func (p *apple) Locate(ctx context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
	if len(aps) == 0 {
		return nil, fmt.Errorf("no wifi access points")
	}

	var lastErr error
	for _, ap := range aps {
		mac := domain.NormalizeMAC(ap.MacAddress)
		if mac == "" {
			continue
		}
		lookups, err := lookupAppleBSSID(ctx, mac)
		if err != nil {
			lastErr = err
			continue
		}
		if loc := pickAppleLocation(ap, lookups); loc != nil {
			return loc, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("%w: apple wps no usable bssid", domain.ErrLocationNotFound)
}

// pickAppleLocation prefers an exact BSSID match; otherwise attributes the first
// correct neighbor hit to the queried AP (SSID/BSSID).
func pickAppleLocation(ap domain.AccessPoint, lookups []domain.Location) *domain.Location {
	mac := domain.NormalizeMAC(ap.MacAddress)
	var fallback *domain.Location
	for i := range lookups {
		loc := &lookups[i]
		if !loc.Correct() {
			continue
		}
		loc.BSSID = domain.NormalizeMAC(loc.BSSID)
		if loc.BSSID == mac {
			loc.SSID = ap.Name
			return loc
		}
		if fallback == nil {
			fallback = loc
		}
	}
	if fallback == nil {
		return nil
	}
	fallback.BSSID = mac
	fallback.SSID = ap.Name
	return fallback
}

var _ port.GeoProvider = (*apple)(nil)

func lookupAppleBSSID(ctx context.Context, bssid string) ([]domain.Location, error) {
	reqBody := buildAppleRequest(bssid)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://gs-loc.apple.com/clls/wloc", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "en-us")
	req.Header.Set("User-Agent", "locationd/1753.17 CFNetwork/711.1.12 Darwin/14.0.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	var body []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		body, err = decompressGzip(resp.Body)
	} else {
		body, err = io.ReadAll(resp.Body)
	}
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if len(body) <= 10 {
		return nil, errors.New("response too short")
	}
	body = body[10:]

	var response bssidapple.BSSIDResp
	if err := proto.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode protobuf: %w", err)
	}

	result := make([]domain.Location, 0, len(response.GetWifi()))
	for _, w := range response.GetWifi() {
		result = append(result, domain.Location{
			Latitude:  float64(w.Location.GetLat()) / 1e8,
			Longitude: float64(w.Location.GetLon()) / 1e8,
			BSSID:     w.GetBssid(),
		})
	}
	return result, nil
}

func buildAppleRequest(bssid string) []byte {
	bssidBytes := fmt.Sprintf("\x12\x13\n\x11%s\x18\x00\x20\x01", bssid)
	payload := []byte("\x00\x01\x00\x05en_US\x00\x13com.apple.locationd\x00\x0a8.1.12B411\x00\x00\x00\x01\x00\x00\x00")
	lengthByte := []byte{byte(len(bssidBytes))}
	return append(payload, append(lengthByte, []byte(bssidBytes)...)...)
}

func decompressGzip(data io.Reader) ([]byte, error) {
	reader, err := gzip.NewReader(data)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
