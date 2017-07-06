package future

import (
	"time"
)

// Fuel calculates new value after elap delta time interval.
func Fuel(initial float32, elap time.Duration) float32 {
	rate := float32(1.0 / 3.0) // 1 unit every 3 seconds
	fuel := initial + rate*float32(int64(elap))/1000000000
	if fuel > 10 {
		fuel = 10
	}
	return fuel
}

// CannonX calculates new value after elap delta time interval.
func CannonX(initial float32, rate float32, elap time.Duration) (float32, float32) {
	x := initial + rate*float32(int64(elap))/1000000000
	switch {
	case x < 0:
		x = -x
		rate = -rate
	case x > 1:
		x = 2 - x
		rate = -rate
	}
	return x, rate
}
