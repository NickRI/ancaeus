package domain

import "errors"

// ErrLocationNotFound is returned when no usable location can be derived from Wi-Fi.
var ErrLocationNotFound = errors.New("location not found")
