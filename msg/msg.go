package msg

import (
	"time"
)

// Update message is sent from server do client.
type Update struct {
	Fuel        float32
	CannonX     float32
	CannonSpeed float32
	Interval    time.Duration
}
