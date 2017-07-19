package future

import (
	"time"
)

// FuelRechargeRate is number of units recharged per second.
const FuelRechargeRate = float32(1.0 / 3.0) // 1 unit every 3 seconds

// Fuel calculates new value after elap delta time interval.
func Fuel(initial float32, elap time.Duration) float32 {
	fuel := initial + FuelRechargeRate*float32(int64(elap))/float32(time.Second)
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

// MissileY calculates new value after elap delta time interval. 0.0 to 1.0
func MissileY(initial float32, rate float32, elap time.Duration) float32 {
	y := initial + rate*float32(int64(elap))/1000000000
	if y > 1 {
		y = 1
	}
	return y
}
