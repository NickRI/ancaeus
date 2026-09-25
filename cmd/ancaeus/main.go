package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ancaeus/internal/domain"
	"ancaeus/internal/infra/cache"
	"ancaeus/internal/infra/httpapi"
	"ancaeus/internal/infra/provider"
	"ancaeus/internal/infra/wifi"

	bolt "go.etcd.io/bbolt"
)

var (
	cachePath          = flag.String("cache-path", "/var/cache/ancaeus/ancaeus_cache.db", "Lookup cache DB path")
	cacheTTL           = flag.Duration("cache-ttl", 30*time.Minute, "Lookup cache TTL (explicit APs and auto/empty)")
	listenAddress      = flag.String("listen", "127.0.0.1:7609", "Listen address")
	providerName       = flag.String("provider", "beacondb", "Geolocation provider: beacondb|google|apple")
	beaconDBURL        = flag.String("beacondb-url", provider.DefaultBeaconDBURL, "BeaconDB geolocate URL")
	googleGeoTokenFile = flag.String("google-geo-token-file", "", "File with Google Geolocation API key (required for -provider=google)")
)

func main() {
	flag.Parse()

	geo, err := provider.New(provider.Config{
		Provider:           *providerName,
		BeaconDBURL:        *beaconDBURL,
		GoogleGeoTokenFile: *googleGeoTokenFile,
	})
	if err != nil {
		slog.Error("provider", "err", err)
		os.Exit(1)
	}

	hlog := slog.With(
		"listen-address", *listenAddress,
		"provider", geo.Name(),
		"cache-path", *cachePath,
	)

	db, err := bolt.Open(*cachePath, 0600, nil)
	if err != nil {
		hlog.Error("failed to open bolt database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	store, err := cache.NewStore[*domain.Location](db, "lookup_cache", *cacheTTL)
	if err != nil {
		hlog.Error("can't create lookup cache", "err", err)
		os.Exit(1)
	}

	locator := cache.NewLocator(
		wifi.NewLocator(geo, wifi.NewRadio(hlog.WithGroup("wifi_radio")), hlog.WithGroup("wifi_location")),
		store,
		hlog.WithGroup("wifi_cache"),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/geolocate", httpapi.Geolocate(locator))
	mux.HandleFunc("/time-zone", httpapi.TimeZone(locator))

	server := http.Server{Addr: *listenAddress, Handler: mux}

	go func() {
		hlog.Info("ancaeus started")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			hlog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	sig := <-signalChan
	hlog.Warn("stopping", "signal", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		hlog.Error("shutdown error", "err", err)
	}
	hlog.Info("ancaeus stopped")
}
