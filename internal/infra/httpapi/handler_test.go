package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ancaeus/internal/domain"
	"ancaeus/internal/infra/httpapi"
	"ancaeus/internal/mocks"

	"github.com/gojuno/minimock/v3"
)

func TestGeolocate(t *testing.T) {
	okLoc := &domain.Location{Latitude: 55.75, Longitude: 37.62, Accuracy: 40}

	t.Run("returns location for provided aps", func(t *testing.T) {
		mc := minimock.NewController(t)
		wfl := mocks.NewWifiLocatorMock(mc)
		wfl.LocateMock.Set(func(_ context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
			if len(aps) != 1 || aps[0].MacAddress != "aa:aa:aa:aa:aa:01" {
				t.Fatalf("aps=%+v", aps)
			}
			return okLoc, nil
		})

		body := `{"wifiAccessPoints":[{"macAddress":"aa:aa:aa:aa:aa:01","inUse":true}]}`
		req := httptest.NewRequest(http.MethodPost, "/geolocate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		httpapi.Geolocate(wfl).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		var got struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			Accuracy float64 `json:"accuracy"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Location.Lat != okLoc.Latitude || got.Accuracy != okLoc.Accuracy {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mc := minimock.NewController(t)
		wfl := mocks.NewWifiLocatorMock(mc)
		wfl.LocateMock.Set(func(_ context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
			if len(aps) != 0 {
				t.Fatalf("aps=%+v", aps)
			}
			return nil, domain.ErrLocationNotFound
		})

		req := httptest.NewRequest(http.MethodPost, "/geolocate", nil)
		rec := httptest.NewRecorder()
		httpapi.Geolocate(wfl).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("code=%d", rec.Code)
		}
	})

	t.Run("options", func(t *testing.T) {
		mc := minimock.NewController(t)
		wfl := mocks.NewWifiLocatorMock(mc)

		req := httptest.NewRequest(http.MethodOptions, "/geolocate", nil)
		rec := httptest.NewRecorder()
		httpapi.Geolocate(wfl).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("code=%d", rec.Code)
		}
	})
}

func TestTimeZone(t *testing.T) {
	t.Run("returns iana zone", func(t *testing.T) {
		mc := minimock.NewController(t)
		wfl := mocks.NewWifiLocatorMock(mc)
		wfl.LocateMock.Expect(minimock.AnyContext, nil).Return(
			&domain.Location{Latitude: 55.75, Longitude: 37.62},
			nil,
		)

		req := httptest.NewRequest(http.MethodGet, "/time-zone", nil)
		rec := httptest.NewRecorder()
		httpapi.TimeZone(wfl).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := strings.TrimSpace(rec.Body.String()); got == "" {
			t.Fatal("empty zone")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mc := minimock.NewController(t)
		wfl := mocks.NewWifiLocatorMock(mc)
		wfl.LocateMock.Expect(minimock.AnyContext, nil).Return(nil, domain.ErrLocationNotFound)

		req := httptest.NewRequest(http.MethodGet, "/time-zone", nil)
		rec := httptest.NewRecorder()
		httpapi.TimeZone(wfl).ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("code=%d", rec.Code)
		}
	})
}
