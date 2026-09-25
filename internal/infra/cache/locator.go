package cache

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"ancaeus/internal/domain"
	"ancaeus/internal/port"
)

const autoCacheKey = "auto"

// Locator caches successful Locate results.
// Explicit aps are keyed by sorted MAC list; empty aps use key "auto".
type Locator struct {
	next  port.WifiLocator
	store *Store[*domain.Location]
	hlog  *slog.Logger
}

// NewLocator wraps next with a TTL cache. Errors are never cached.
func NewLocator(next port.WifiLocator, store *Store[*domain.Location], hlog *slog.Logger) *Locator {
	return &Locator{next: next, store: store, hlog: hlog}
}

var _ port.WifiLocator = (*Locator)(nil)

func (c *Locator) Locate(ctx context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
	key := cacheKey(aps)
	if key == "" {
		return c.next.Locate(ctx, aps)
	}

	cached, ok, err := c.store.Get(key)
	if err != nil {
		return nil, fmt.Errorf("cache get %q: %w", key, err)
	}
	if ok {
		c.hlog.Info("cache hit",
			"key", key,
			"ssid", cached.SSID,
			"mac", cached.BSSID,
			"lat", cached.Latitude,
			"lng", cached.Longitude,
			"accuracy", cached.Accuracy,
		)
		return cached, nil
	}

	loc, err := c.next.Locate(ctx, aps)
	if err != nil {
		return nil, err
	}

	if err := c.store.Set(key, loc); err != nil {
		return nil, fmt.Errorf("cache set %q: %w", key, err)
	}
	c.hlog.Info("cache store",
		"key", key,
		"ssid", loc.SSID,
		"mac", loc.BSSID,
		"lat", loc.Latitude,
		"lng", loc.Longitude,
		"accuracy", loc.Accuracy,
	)
	return loc, nil
}

func cacheKey(aps []domain.AccessPoint) string {
	if len(aps) == 0 {
		return autoCacheKey
	}
	macs := make([]string, 0, len(aps))
	for _, ap := range domain.FilterAccessPoints(aps) {
		if mac := domain.NormalizeMAC(ap.MacAddress); mac != "" {
			macs = append(macs, mac)
		}
	}
	slices.Sort(macs)
	return strings.Join(macs, ",")
}
