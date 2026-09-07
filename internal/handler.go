package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/bradfitz/latlong"
)

type WifiRequest struct {
	WifiAccessPoints []AccessPoint `json:"wifiAccessPoints"`
}

type AccessPoint struct {
	Name           string `json:"name"`
	MacAddress     string `json:"macAddress"`
	InUse          bool   `json:"inUse"`
	SignalStrength int    `json:"signalStrength"`
}

type WifiResponse struct {
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	Accuracy float64 `json:"accuracy"`
}

type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewErrorResponse(code int, message string) *ErrorResponse {
	return &ErrorResponse{
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	}
}

func writeJSONError(hlog *slog.Logger, w http.ResponseWriter, status int, err error) {
	hlog.Error(err.Error())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(NewErrorResponse(status, err.Error()))
}

func sendLocation(w http.ResponseWriter, lookup *LookupResult) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	accuracy := lookup.Accuracy
	if accuracy == 0 {
		accuracy = 30
	}

	_ = json.NewEncoder(w).Encode(&WifiResponse{
		Location: struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		}{
			Lat: lookup.Latitude,
			Lng: lookup.Longitude,
		},
		Accuracy: accuracy,
	})
}

func GoogleLocationHandler(wfl *WifiLocator) func(w http.ResponseWriter, r *http.Request) {
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

		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		if err != nil {
			writeJSONError(hlog, w, http.StatusInternalServerError, fmt.Errorf("error reading body: %w", err))
			return
		}
		defer r.Body.Close()

		hlog.Info("geolocate request", "body_bytes", buf.Len())

		var wreq WifiRequest
		if buf.Len() > 0 {
			if err := json.NewDecoder(&buf).Decode(&wreq); err != nil {
				writeJSONError(hlog, w, http.StatusBadRequest, fmt.Errorf("failed to decode request: %w", err))
				return
			}
		}

		ap, err := wfl.TryProcessWifiAPS(r.Context(), wreq.WifiAccessPoints)
		if err != nil {
			if errors.Is(err, ErrLocationNotFound) {
				writeJSONError(hlog, w, http.StatusNotFound, err)
				return
			}
			writeJSONError(hlog, w, http.StatusInternalServerError, err)
			return
		}

		sendLocation(w, ap)
	}
}

func TimeZoneHandler(wfl *WifiLocator) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		hlog := slog.With(
			slog.String("path", r.URL.Path),
			slog.Int64("ts", time.Now().UnixMilli()),
		)

		aps, err := wfl.TryProcessWifiAPS(r.Context(), nil)
		if err != nil {
			if errors.Is(err, ErrLocationNotFound) {
				writeJSONError(hlog, w, http.StatusNotFound, err)
				return
			}
			writeJSONError(hlog, w, http.StatusInternalServerError, err)
			return
		}

		zone := latlong.LookupZoneName(aps.Latitude, aps.Longitude)
		if zone == "" {
			writeJSONError(hlog, w, http.StatusNotFound, fmt.Errorf("%w: empty timezone for coordinates", ErrLocationNotFound))
			return
		}

		hlog.Info("time zone", slog.String("zone", zone))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, zone)
	}
}
