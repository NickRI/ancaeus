package wifi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"ancaeus/internal/domain"
	"ancaeus/internal/port"
)

// Locator resolves location via Radio scan + GeoProvider (no caching).
type Locator struct {
	provider port.GeoProvider
	radio    port.Radio
	hlog     *slog.Logger
}

// NewLocator returns an uncached Wi-Fi locator.
func NewLocator(provider port.GeoProvider, radio port.Radio, hlog *slog.Logger) *Locator {
	return &Locator{provider: provider, radio: radio, hlog: hlog}
}

var _ port.WifiLocator = (*Locator)(nil)

func (l *Locator) Locate(ctx context.Context, aps []domain.AccessPoint) (*domain.Location, error) {
	if len(aps) > 0 {
		loc, err := l.resolve(ctx, aps, true)
		if err == nil {
			return loc, nil
		}
		if !errors.Is(err, domain.ErrLocationNotFound) {
			return nil, err
		}
		l.hlog.Info("provided aps miss, scanning nearby", "err", err)
	} else {
		assoc, err := l.radio.Scan(ctx, false)
		if err != nil {
			return nil, fmt.Errorf("scan associated: %w", err)
		}
		if len(assoc) > 0 {
			loc, err := l.resolve(ctx, assoc, true)
			if err == nil {
				return loc, nil
			}
			if !errors.Is(err, domain.ErrLocationNotFound) {
				return nil, err
			}
			l.hlog.Info("associated miss, scanning nearby", "err", err)
		}
	}

	nearby, err := l.radio.Scan(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("scan nearby: %w", err)
	}

	return l.resolve(ctx, nearby, false)
}

func (l *Locator) resolve(ctx context.Context, aps []domain.AccessPoint, preferInUse bool) (*domain.Location, error) {
	aps = domain.SortedByPriority(aps, preferInUse)
	if len(aps) == 0 {
		return nil, fmt.Errorf("%w: no wifi access points", domain.ErrLocationNotFound)
	}

	primary := aps[0]
	l.hlog.Info("locating",
		"provider", l.provider.Name(),
		"aps", len(aps),
		"ssid", primary.Name,
		"mac", primary.MacAddress,
		"signal", primary.SignalStrength,
		"in_use", primary.InUse,
	)
	loc, err := l.provider.Locate(ctx, aps)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", l.provider.Name(), err)
	}
	annotateLocation(loc, aps, primary)
	l.hlog.Info("located",
		"provider", l.provider.Name(),
		"ssid", loc.SSID,
		"mac", loc.BSSID,
		"lat", loc.Latitude,
		"lng", loc.Longitude,
		"accuracy", loc.Accuracy,
	)
	return loc, nil
}

// annotateLocation fills SSID/BSSID from the AP that produced the hit (or primary).
func annotateLocation(loc *domain.Location, aps []domain.AccessPoint, primary domain.AccessPoint) {
	if ap := findAP(aps, loc.BSSID); ap != nil {
		loc.SSID = ap.Name
		loc.BSSID = ap.MacAddress
		return
	}
	loc.SSID = primary.Name
	if loc.BSSID == "" {
		loc.BSSID = primary.MacAddress
	} else {
		loc.BSSID = domain.NormalizeMAC(loc.BSSID)
	}
}

func findAP(aps []domain.AccessPoint, bssid string) *domain.AccessPoint {
	mac := domain.NormalizeMAC(bssid)
	if mac == "" {
		return nil
	}
	for i := range aps {
		if aps[i].MacAddress == mac {
			return &aps[i]
		}
	}
	return nil
}
