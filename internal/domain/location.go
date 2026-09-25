package domain

// Location is a resolved geographic position.
type Location struct {
	Latitude  float64
	Longitude float64
	SSID      string
	BSSID     string
	Accuracy  float64
}

func (l Location) Correct() bool {
	return l.Latitude != -180 && l.Longitude != -180
}
