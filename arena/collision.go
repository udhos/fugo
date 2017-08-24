package main

import (
	"time"

	"github.com/udhos/fugo/future"
	"github.com/udhos/fugo/unit"
)

func detectCollision(w *world, now time.Time) {

	//field := unit.Rect{X1: -1, Y1: -1, X2: 1, Y2: 1}
	fieldTop := 1.0
	cannonBottom := -1.0

	for _, m := range w.missileList {
		mY := float64(future.MissileY(m.CoordY, m.Speed, now.Sub(m.Start)))
		up := m.Team == 0
		//mr := unit.MissileBox(field, float64(m.CoordX), mY, fieldTop, cannonBottom, up)
		unit.MissileBox(-1, 1, float64(m.CoordX), mY, fieldTop, cannonBottom, up)

		for _, p := range w.playerTab {
			//cX, _ := future.CannonX(p.cannonCoordX, p.cannonSpeed, now.Sub(p.cannonStart))
			future.CannonX(p.cannonCoordX, p.cannonSpeed, now.Sub(p.cannonStart))
		}
	}
}
