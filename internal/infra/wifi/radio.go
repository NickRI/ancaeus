package wifi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"ancaeus/internal/domain"
	"ancaeus/internal/port"

	mdwifi "github.com/mdlayher/wifi"
)

// Radio is an nl80211-backed Wi-Fi scanner.
type Radio struct {
	hlog *slog.Logger
}

// NewRadio returns an nl80211 Radio.
func NewRadio(hlog *slog.Logger) *Radio {
	return &Radio{hlog: hlog}
}

var _ port.Radio = (*Radio)(nil)

func (r *Radio) Scan(ctx context.Context, full bool) ([]domain.AccessPoint, error) {
	c, err := mdwifi.New()
	if err != nil {
		return nil, fmt.Errorf("create wifi client: %w", err)
	}
	defer c.Close()

	ifaces, err := c.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list wifi interfaces: %w", err)
	}

	byMAC := map[string]domain.AccessPoint{}
	for _, iface := range ifaces {
		if iface.Name == "" {
			continue
		}

		signals := stationSignals(c, iface)
		if assoc := associatedAPs(c, iface, signals); len(assoc) > 0 {
			r.hlog.Info("associated bss", "iface", iface.Name, "ssid", assoc[0].Name)
			byMAC = mergeAPs(byMAC, assoc...)
		}
		byMAC = mergeAPs(byMAC, stationAPs(signals)...)

		if !full {
			continue
		}

		// Associated BSS alone is already in byMAC — always trigger an active
		// scan for neighbors. Skipping when len(byMAC)>0 left fallback with 1 AP.
		r.hlog.Info("wifi scan", "iface", iface.Name)
		if err := c.Scan(ctx, iface); err != nil {
			r.hlog.Warn("scan failed", "iface", iface.Name, "err", err)
		}
		visible, err := visibleAPs(c, iface, signals)
		if err != nil {
			r.hlog.Warn("access points failed", "iface", iface.Name, "err", err)
			continue
		}
		byMAC = mergeAPs(byMAC, visible...)
	}

	if full && len(byMAC) == 0 {
		return nil, errors.New("no wifi access points found")
	}
	return slices.Collect(maps.Values(byMAC)), nil
}

func stationSignals(c *mdwifi.Client, iface *mdwifi.Interface) map[string]int {
	out := map[string]int{}
	stations, err := c.StationInfo(iface)
	if err != nil {
		return out
	}
	for _, st := range stations {
		if st.HardwareAddr != nil {
			out[domain.NormalizeMAC(st.HardwareAddr.String())] = st.Signal
		}
	}
	return out
}

func associatedAPs(c *mdwifi.Client, iface *mdwifi.Interface, signals map[string]int) []domain.AccessPoint {
	bss, err := c.BSS(iface)
	if err != nil || bss.BSSID == nil {
		return nil
	}
	if bss.Status != mdwifi.BSSStatusAssociated && bss.Status != mdwifi.BSSStatusIBSSJoined {
		return nil
	}
	mac := domain.NormalizeMAC(bss.BSSID.String())
	return []domain.AccessPoint{{
		Name:           bss.SSID,
		MacAddress:     mac,
		InUse:          true,
		SignalStrength: signalStrength(signals, mac, bss.Signal),
	}}
}

func stationAPs(signals map[string]int) []domain.AccessPoint {
	out := make([]domain.AccessPoint, 0, len(signals))
	for mac, signal := range signals {
		out = append(out, domain.AccessPoint{
			MacAddress:     mac,
			InUse:          true,
			SignalStrength: signal,
		})
	}
	return out
}

func visibleAPs(c *mdwifi.Client, iface *mdwifi.Interface, signals map[string]int) ([]domain.AccessPoint, error) {
	infos, err := c.AccessPoints(iface)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AccessPoint, 0, len(infos))
	for _, info := range infos {
		if info.BSSID == nil {
			continue
		}
		mac := domain.NormalizeMAC(info.BSSID.String())
		out = append(out, domain.AccessPoint{
			Name:           info.SSID,
			MacAddress:     mac,
			InUse:          info.Status == mdwifi.BSSStatusAssociated || info.Status == mdwifi.BSSStatusIBSSJoined,
			SignalStrength: signalStrength(signals, mac, info.Signal),
		})
	}
	return out, nil
}

// signalStrength prefers station dBm; otherwise converts BSS mBm (100*dBm) to dBm.
func signalStrength(station map[string]int, mac string, bssMBM int32) int {
	if s, ok := station[mac]; ok {
		return s
	}
	return int(bssMBM / 100)
}

func mergeAPs(byMAC map[string]domain.AccessPoint, aps ...domain.AccessPoint) map[string]domain.AccessPoint {
	out := maps.Clone(byMAC)
	if out == nil {
		out = map[string]domain.AccessPoint{}
	}
	for _, ap := range aps {
		mac := domain.NormalizeMAC(ap.MacAddress)
		if mac == "" || strings.HasSuffix(strings.ToLower(ap.Name), "_nomap") {
			continue
		}
		ap.MacAddress = mac
		if prev, ok := out[mac]; ok {
			out[mac] = mergeAP(prev, ap)
			continue
		}
		out[mac] = ap
	}
	return out
}

func mergeAP(prev, next domain.AccessPoint) domain.AccessPoint {
	if next.InUse {
		prev.InUse = true
	}
	if next.SignalStrength != 0 {
		prev.SignalStrength = next.SignalStrength
	}
	if next.Name != "" {
		prev.Name = next.Name
	}
	return prev
}
