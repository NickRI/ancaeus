package domain

import (
	"cmp"
	"slices"
)

// SortedByPriority returns a new slice: InUse first (if preferInUse), then by signal desc.
func SortedByPriority(aps []AccessPoint, preferInUse bool) []AccessPoint {
	out := FilterAccessPoints(aps)
	slices.SortFunc(out, func(a, b AccessPoint) int {
		if preferInUse && a.InUse != b.InUse {
			if a.InUse {
				return -1
			}
			return 1
		}
		return cmp.Compare(b.SignalStrength, a.SignalStrength)
	})
	return out
}
