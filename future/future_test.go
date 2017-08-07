package future

import (
	"testing"
	"time"
)

func TestMissileY(t *testing.T) {
	missileY(t, .2, .5, 500*time.Millisecond, .45)
	missileY(t, .1, .5, time.Second, .6)
}

func TestFuel(t *testing.T) {
	fuel(t, 1, 3*time.Second, 2.0)
}

func TestCannonX(t *testing.T) {
	cannonX(t, .1, .5, time.Second, .6)
}

func cannonX(t *testing.T, initial, rate float32, elap time.Duration, expected float32) {
	x, _ := CannonX(initial, rate, elap)
	if x != expected {
		t.Errorf("cannonX: initial=%v rate=%v elap=%v expected=%v result=%v", initial, rate, elap, expected, x)
	}
}

func missileY(t *testing.T, initial, rate float32, elap time.Duration, expected float32) {
	y := MissileY(initial, rate, elap)
	if y != expected {
		t.Errorf("missileY: initial=%v rate=%v elap=%v expected=%v result=%v", initial, rate, elap, expected, y)
	}
}

func fuel(t *testing.T, initial float32, elap time.Duration, expected float32) {
	f := Fuel(initial, elap)
	if f != expected {
		t.Errorf("fuel: initial=%v elap=%v expected=%v result=%v", initial, elap, expected, f)
	}
}
