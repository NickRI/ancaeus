package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"ancaeus/internal/domain"
	"ancaeus/internal/port"

	"github.com/bradfitz/latlong"
)

type accessPointDTO struct {
	Name           string `json:"name"`
	MacAddress     string `json:"macAddress"`
	InUse          bool   `json:"inUse"`
	SignalStrength int    `json:"signalStrength"`
}

func (d accessPointDTO) toDomain() domain.AccessPoint {
	return domain.AccessPoint{
		Name:           d.Name,
		MacAddress:     d.MacAddress,
		InUse:          d.InUse,
		SignalStrength: d.SignalStrength,
	}
}

type wifiRequest struct {
	WifiAccessPoints []accessPointDTO `json:"wifiAccessPoints"`
}

func (r wifiRequest) accessPoints() []domain.AccessPoint {
	out := make([]domain.AccessPoint, len(r.WifiAccessPoints))
	for i, ap := range r.WifiAccessPoints {
		out[i] = ap.toDomain()
	}
	return out
}

type wifiResponse struct {
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	Accuracy float64 `json:"accuracy"`
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSONError(hlog *slog.Logger, w http.ResponseWriter, status int, err error) {
	hlog.Error(err.Error())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{Code: status, Message: err.Error()},
	})
}

func sendLocation(w http.ResponseWriter, loc *domain.Location) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	accuracy := loc.Accuracy
	if accuracy == 0 {
		accuracy = 30
	}

	_ = json.NewEncoder(w).Encode(&wifiResponse{
		Location: struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		}{Lat: loc.Latitude, Lng: loc.Longitude},
		Accuracy: accuracy,
	})
}

// Geolocate handles Google/GeoClue-compatible location requests.
func Geolocate(wfl port.WifiLocator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hlog := slog.With(
			slog.String("path", r.URL.Path),
			slog.Int64("ts", time.Now().UnixMilli()),
			slog.String("method", r.Method),
		)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		defer r.Body.Close()

		hlog.Info("geolocate request")

		var wreq wifiRequest
		if r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&wreq); err != nil {
				writeJSONError(hlog, w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))
				return
			}
		}

		loc, err := wfl.Locate(r.Context(), wreq.accessPoints())
		if err != nil {
			if errors.Is(err, domain.ErrLocationNotFound) {
				writeJSONError(hlog, w, http.StatusNotFound, err)
				return
			}
			writeJSONError(hlog, w, http.StatusInternalServerError, err)
			return
		}
		sendLocation(w, loc)
	}
}

// TimeZone returns an IANA timezone for the current Wi-Fi location.
func TimeZone(wfl port.WifiLocator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hlog := slog.With(
			slog.String("path", r.URL.Path),
			slog.Int64("ts", time.Now().UnixMilli()),
		)

		loc, err := wfl.Locate(r.Context(), nil)
		if err != nil {
			if errors.Is(err, domain.ErrLocationNotFound) {
				writeJSONError(hlog, w, http.StatusNotFound, err)
				return
			}
			writeJSONError(hlog, w, http.StatusInternalServerError, err)
			return
		}

		zone := latlong.LookupZoneName(loc.Latitude, loc.Longitude)
		if zone == "" {
			writeJSONError(hlog, w, http.StatusNotFound, fmt.Errorf("%w: empty timezone for coordinates", domain.ErrLocationNotFound))
			return
		}

		hlog.Info("time zone", slog.String("zone", zone))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, zone)
	}
}
