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

	"ancaeus/internal"

	bolt "go.etcd.io/bbolt"
)

var (
	cachePath          = flag.String("cache-path", "/var/cache/ancaeus/ancaeus_cache.db", "BSSID/wifi cache DB path")
	wifiCacheTTL       = flag.Duration("wifi-cache-ttl", 5*time.Minute, "WiFi get cache TTL")
	lookupCacheTTL     = flag.Duration("lookup-cache-ttl", 6*time.Hour, "Lookup cache TTL")
	listenAddress      = flag.String("listen", "127.0.0.1:1223", "Listen address")
	provider           = flag.String("provider", "beacondb", "Geolocation provider: beacondb|google|apple")
	beaconDBURL        = flag.String("beacondb-url", internal.DefaultBeaconDBURL, "BeaconDB geolocate URL")
	googleGeoTokenFile = flag.String("google-geo-token-file", "", "File with Google Geolocation API key (required for -provider=google)")
)

func main() {
	flag.Parse()

	geo, err := internal.NewGeoProvider(internal.ProviderConfig{
		Provider:           *provider,
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

	lookupCache, err := internal.NewCache[*internal.LookupResult](db, "lookup_cache", *lookupCacheTTL)
	if err != nil {
		hlog.Error("can't create lookup cache", "err", err)
		os.Exit(1)
	}

	wifiCache, err := internal.NewCache[[]internal.AccessPoint](db, "wifi_get_cache", *wifiCacheTTL)
	if err != nil {
		hlog.Error("can't create wifi cache", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	wifiLocator := internal.NewWifiLocator(geo, lookupCache, wifiCache, hlog.WithGroup("wifi_location"))
	mux.HandleFunc("/geolocate", internal.GoogleLocationHandler(wifiLocator))
	mux.HandleFunc("/time-zone", internal.TimeZoneHandler(wifiLocator))

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
