package port

import (
	"context"

	"ancaeus/internal/domain"
)

// WifiLocator resolves a geographic location from Wi-Fi access points.
type WifiLocator interface {
	Locate(ctx context.Context, aps []domain.AccessPoint) (*domain.Location, error)
}

// GeoProvider talks to an external geolocation backend.
type GeoProvider interface {
	Name() string
	Locate(ctx context.Context, aps []domain.AccessPoint) (*domain.Location, error)
}

// Radio scans nearby Wi-Fi access points via the local radio.
type Radio interface {
	// Scan returns associated/station APs when full is false,
	// or a full dump/scan when full is true.
	Scan(ctx context.Context, full bool) ([]domain.AccessPoint, error)
}

//go:generate go run github.com/gojuno/minimock/v3/cmd/minimock@v3.4.7 -i ancaeus/internal/port.WifiLocator -o ../mocks/wifi_locator_mock.go -n WifiLocatorMock -p mocks
//go:generate go run github.com/gojuno/minimock/v3/cmd/minimock@v3.4.7 -i ancaeus/internal/port.GeoProvider -o ../mocks/geo_provider_mock.go -n GeoProviderMock -p mocks
//go:generate go run github.com/gojuno/minimock/v3/cmd/minimock@v3.4.7 -i ancaeus/internal/port.Radio -o ../mocks/radio_mock.go -n RadioMock -p mocks
