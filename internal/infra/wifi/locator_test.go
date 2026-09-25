package wifi_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"ancaeus/internal/domain"
	"ancaeus/internal/infra/wifi"
	"ancaeus/internal/mocks"

	"github.com/gojuno/minimock/v3"
)

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestLocator_UseCases(t *testing.T) {
	assoc := []domain.AccessPoint{{MacAddress: "aa:aa:aa:aa:aa:01", InUse: true}}
	nearby := []domain.AccessPoint{
		{MacAddress: "aa:aa:aa:aa:aa:01", SignalStrength: -30},
		{MacAddress: "bb:bb:bb:bb:bb:02", SignalStrength: -40},
	}
	provided := []domain.AccessPoint{{MacAddress: "cc:cc:cc:cc:cc:03", InUse: true}}
	okLoc := &domain.Location{Latitude: 55.75, Longitude: 37.62}
	boom := errors.New("provider down")

	t.Run("associated hit does not scan nearby", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		radio.ScanMock.Expect(minimock.AnyContext, false).Return(assoc, nil)
		geo.NameMock.Return("stub")
		geo.LocateMock.Set(func(_ context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
			if len(aps) != 1 || aps[0].MacAddress != "aa:aa:aa:aa:aa:01" {
				t.Fatalf("unexpected aps: %+v", aps)
			}
			return okLoc, nil
		})

		loc, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if loc.Latitude != okLoc.Latitude {
			t.Fatalf("lat=%v", loc.Latitude)
		}
	})

	t.Run("associated miss falls back to nearby", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		radio.ScanMock.When(minimock.AnyContext, false).Then(assoc, nil)
		radio.ScanMock.When(minimock.AnyContext, true).Then(nearby, nil)
		geo.NameMock.Return("stub")
		geo.LocateMock.Set(func(_ context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
			if len(aps) == 1 && aps[0].MacAddress == "aa:aa:aa:aa:aa:01" {
				return nil, domain.ErrLocationNotFound
			}
			return okLoc, nil
		})

		loc, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if loc.Latitude != okLoc.Latitude {
			t.Fatalf("lat=%v", loc.Latitude)
		}
	})

	t.Run("associated non-not-found does not fall back", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		radio.ScanMock.Expect(minimock.AnyContext, false).Return(assoc, nil)
		geo.NameMock.Return("stub")
		geo.LocateMock.Return(nil, boom)

		_, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), nil)
		if !errors.Is(err, boom) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("provided aps hit", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		geo.NameMock.Return("stub")
		geo.LocateMock.Set(func(_ context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
			if len(aps) != 1 || aps[0].MacAddress != "cc:cc:cc:cc:cc:03" {
				t.Fatalf("unexpected aps: %+v", aps)
			}
			return okLoc, nil
		})

		loc, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), provided)
		if err != nil {
			t.Fatal(err)
		}
		if loc.Latitude != okLoc.Latitude {
			t.Fatalf("lat=%v", loc.Latitude)
		}
	})

	t.Run("provided aps miss falls back to nearby scan", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		radio.ScanMock.Expect(minimock.AnyContext, true).Return(nearby, nil)
		geo.NameMock.Return("stub")
		geo.LocateMock.Set(func(_ context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
			if len(aps) == 1 && aps[0].MacAddress == "cc:cc:cc:cc:cc:03" {
				return nil, domain.ErrLocationNotFound
			}
			return okLoc, nil
		})

		loc, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), provided)
		if err != nil {
			t.Fatal(err)
		}
		if loc.Latitude != okLoc.Latitude {
			t.Fatalf("lat=%v", loc.Latitude)
		}
	})

	t.Run("no associated goes straight to nearby", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		radio.ScanMock.When(minimock.AnyContext, false).Then(nil, nil)
		radio.ScanMock.When(minimock.AnyContext, true).Then(nearby, nil)
		geo.NameMock.Return("stub")
		geo.LocateMock.Return(okLoc, nil)

		loc, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if loc.Latitude != okLoc.Latitude {
			t.Fatalf("lat=%v", loc.Latitude)
		}
	})

	t.Run("associated scan error", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		radio.ScanMock.Expect(minimock.AnyContext, false).Return(nil, boom)

		_, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), nil)
		if !errors.Is(err, boom) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("nearby scan error after assoc miss", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		radio.ScanMock.When(minimock.AnyContext, false).Then(assoc, nil)
		radio.ScanMock.When(minimock.AnyContext, true).Then(nil, boom)
		geo.NameMock.Return("stub")
		geo.LocateMock.Return(nil, domain.ErrLocationNotFound)

		_, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), nil)
		if !errors.Is(err, boom) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("annotates ssid and mac on success", func(t *testing.T) {
		mc := minimock.NewController(t)
		radio := mocks.NewRadioMock(mc)
		geo := mocks.NewGeoProviderMock(mc)

		aps := []domain.AccessPoint{{Name: "Cafe", MacAddress: "cc:cc:cc:cc:cc:03", InUse: true}}
		geo.NameMock.Return("stub")
		geo.LocateMock.Return(&domain.Location{Latitude: 1, Longitude: 2}, nil)

		loc, err := wifi.NewLocator(geo, radio, discardLog()).Locate(t.Context(), aps)
		if err != nil {
			t.Fatal(err)
		}
		if loc.SSID != "Cafe" || loc.BSSID != "cc:cc:cc:cc:cc:03" {
			t.Fatalf("loc=%+v", loc)
		}
	})
}
