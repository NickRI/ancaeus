package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/mdlayher/wifi"
)

var ErrLocationNotFound = errors.New("location not found")

type WifiLocator struct {
	provider GeoProvider
	lcache   *Cache[*LookupResult]
	wcache   *Cache[[]AccessPoint]
	hlog     *slog.Logger
}

func NewWifiLocator(provider GeoProvider, lcache *Cache[*LookupResult], wcache *Cache[[]AccessPoint], hlog *slog.Logger) *WifiLocator {
	return &WifiLocator{provider: provider, lcache: lcache, wcache: wcache, hlog: hlog}
}

func cacheKey(aps []AccessPoint) string {
	macs := make([]string, 0, len(aps))
	for _, ap := range aps {
		mac := strings.ReplaceAll(ap.MacAddress, "-", ":")
		if mac != "" {
			macs = append(macs, strings.ToLower(mac))
		}
	}
	sort.Strings(macs)
	return strings.Join(macs, ",")
}

func isNomapSSID(ssid string) bool {
	return strings.HasSuffix(strings.ToLower(ssid), "_nomap")
}

func filterAccessPoints(aps []AccessPoint) []AccessPoint {
	out := make([]AccessPoint, 0, len(aps))
	seen := map[string]struct{}{}
	for _, ap := range aps {
		if isNomapSSID(ap.Name) {
			continue
		}
		mac := strings.ToLower(strings.ReplaceAll(ap.MacAddress, "-", ":"))
		if mac == "" {
			continue
		}
		if _, ok := seen[mac]; ok {
			continue
		}
		seen[mac] = struct{}{}
		ap.MacAddress = mac
		out = append(out, ap)
	}
	return out
}

func (l *WifiLocator) TryProcessWifiAPS(ctx context.Context, accessPoints []AccessPoint) (*LookupResult, error) {
	if len(accessPoints) == 0 {
		scanPoints, exists, err := l.wcache.Get("wifi-scan")
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve wifi scan: %w", err)
		}

		if exists {
			l.hlog.Info("using cached wifi scan", "count", len(scanPoints))
			accessPoints = scanPoints
		} else {
			accessPoints, err = GetWifiInfo(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch wifi access points: %w", err)
			}
			if err := l.wcache.Set("wifi-scan", accessPoints); err != nil {
				return nil, fmt.Errorf("failed to cache wifi access points: %w", err)
			}
		}
	}

	accessPoints = filterAccessPoints(accessPoints)

	sort.Slice(accessPoints, func(i, j int) bool {
		if accessPoints[i].InUse != accessPoints[j].InUse {
			return accessPoints[i].InUse
		}
		return accessPoints[i].SignalStrength > accessPoints[j].SignalStrength
	})

	key := cacheKey(accessPoints)
	if key == "" {
		return nil, fmt.Errorf("%w: no wifi access points", ErrLocationNotFound)
	}

	if cached, exists, err := l.lcache.Get(key); err != nil {
		return nil, fmt.Errorf("failed to get lookup cache: %w", err)
	} else if exists {
		l.hlog.Info("cache hit", "provider", l.provider.Name())
		return cached, nil
	}

	l.hlog.Info("locating", "provider", l.provider.Name(), "aps", len(accessPoints))

	lookup, err := l.provider.Locate(ctx, accessPoints)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", l.provider.Name(), err)
	}

	if err := l.lcache.Set(key, lookup); err != nil {
		return nil, fmt.Errorf("failed to set lookup cache: %w", err)
	}

	l.hlog.Info("located", "provider", l.provider.Name(), "lat", lookup.Latitude, "lng", lookup.Longitude)
	return lookup, nil
}

func wifiInUse(status wifi.BSSStatus) bool {
	return status == wifi.BSSStatusAssociated || status == wifi.BSSStatusIBSSJoined
}

func stationSignals(c *wifi.Client, iface *wifi.Interface) map[string]int {
	out := map[string]int{}
	stations, err := c.StationInfo(iface)
	if err != nil {
		return out
	}
	for _, st := range stations {
		if st.HardwareAddr != nil {
			out[strings.ToLower(st.HardwareAddr.String())] = st.Signal
		}
	}
	return out
}

func mergeAP(dst map[string]AccessPoint, ap AccessPoint) {
	mac := strings.ToLower(strings.ReplaceAll(ap.MacAddress, "-", ":"))
	if mac == "" || isNomapSSID(ap.Name) {
		return
	}
	ap.MacAddress = mac
	if prev, ok := dst[mac]; ok {
		if ap.InUse {
			prev.InUse = true
		}
		if ap.SignalStrength != 0 {
			prev.SignalStrength = ap.SignalStrength
		}
		if ap.Name != "" {
			prev.Name = ap.Name
		}
		dst[mac] = prev
		return
	}
	dst[mac] = ap
}

func dumpAccessPoints(c *wifi.Client, iface *wifi.Interface, byMAC map[string]AccessPoint) error {
	infos, err := c.AccessPoints(iface)
	if err != nil {
		return err
	}
	signals := stationSignals(c, iface)
	for _, info := range infos {
		if info.BSSID == nil {
			continue
		}
		mac := strings.ToLower(info.BSSID.String())
		mergeAP(byMAC, AccessPoint{
			Name:           info.SSID,
			MacAddress:     mac,
			InUse:          wifiInUse(info.Status),
			SignalStrength: signals[mac],
		})
	}
	return nil
}

func GetWifiInfo(ctx context.Context) ([]AccessPoint, error) {
	c, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create wifi: %v", err)
	}
	defer c.Close()

	ifs, err := c.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list wifi interfaces: %w", err)
	}

	byMAC := map[string]AccessPoint{}

	for _, iface := range ifs {
		if iface.Name == "" {
			continue
		}
		hlog := slog.With("iface", iface.Name)
		before := len(byMAC)

		if bss, err := c.BSS(iface); err == nil && bss.BSSID != nil && wifiInUse(bss.Status) {
			hlog.Info("associated bss", "ssid", bss.SSID)
			mergeAP(byMAC, AccessPoint{
				Name:       bss.SSID,
				MacAddress: bss.BSSID.String(),
				InUse:      true,
			})
		}

		for mac, signal := range stationSignals(c, iface) {
			mergeAP(byMAC, AccessPoint{
				MacAddress:     mac,
				InUse:          true,
				SignalStrength: signal,
			})
		}

		if err := dumpAccessPoints(c, iface, byMAC); err != nil {
			hlog.Warn("access points failed", "err", err)
		}

		if len(byMAC) > before {
			continue
		}

		hlog.Info("wifi scan")
		if err := c.Scan(ctx, iface); err != nil {
			hlog.Warn("scan failed", "err", err)
			continue
		}
		if err := dumpAccessPoints(c, iface, byMAC); err != nil {
			hlog.Warn("access points failed", "err", err)
		}
	}

	if len(byMAC) == 0 {
		return nil, errors.New("no wifi access points found")
	}

	out := make([]AccessPoint, 0, len(byMAC))
	for _, ap := range byMAC {
		out = append(out, ap)
	}
	return out, nil
}
