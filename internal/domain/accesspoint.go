package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// AccessPoint is a visible Wi-Fi BSS.
type AccessPoint struct {
	Name           string
	MacAddress     string
	InUse          bool
	SignalStrength int
}

// NormalizeMAC lowercases, unifies separators to ':', and zero-pads octets.
// Apple WPS often returns BSSIDs like "12:9:d:54:1f:bb".
func NormalizeMAC(mac string) string {
	mac = strings.ToLower(strings.ReplaceAll(mac, "-", ":"))
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return mac
	}
	for i, p := range parts {
		if p == "" {
			return mac
		}
		n, err := strconv.ParseUint(p, 16, 8)
		if err != nil {
			return mac
		}
		parts[i] = fmt.Sprintf("%02x", n)
	}
	return strings.Join(parts, ":")
}

// FilterAccessPoints drops _nomap SSIDs, empty MACs, and duplicates.
func FilterAccessPoints(aps []AccessPoint) []AccessPoint {
	out := make([]AccessPoint, 0, len(aps))
	seen := make(map[string]struct{}, len(aps))
	for _, ap := range aps {
		if strings.HasSuffix(strings.ToLower(ap.Name), "_nomap") {
			continue
		}
		mac := NormalizeMAC(ap.MacAddress)
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
