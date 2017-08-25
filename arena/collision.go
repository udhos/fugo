package main

import (
	//"log"
	"time"

	"github.com/udhos/fugo/future"
	"github.com/udhos/fugo/unit"
)

type box interface {
	Bounding() (float64, float64, float64, float64)
}

func intersect(b1, b2 box) bool {
	b1x1, b1y1, b1x2, b1y2 := b1.Bounding()
	b2x1, b2y1, b2x2, b2y2 := b2.Bounding()

	noOverlap := b1x1 > b2x2 ||
		b2x1 > b1x2 ||
		b1y1 > b2y2 ||
		b2y1 > b1y2

	return !noOverlap
}

func detectCollision(w *world, now time.Time) bool {

	left := -1.0
	right := 1.0
	fieldTop := 1.0
	cannonBottom := -1.0
	hit := false

NEXT_MISSILE:
	for i := 0; i < len(w.missileList); i++ {
		m := w.missileList[i]
		mY := float64(future.MissileY(m.CoordY, m.Speed, now.Sub(m.Start)))
		mUp := m.Team == 0
		mr := unit.MissileBox(left, right, float64(m.CoordX), mY, fieldTop, cannonBottom, mUp)

		for _, p := range w.playerTab {
			if p.cannonLife <= 0 {
				continue
			}
			if m.Team == p.team {
				continue
			}
			cX, _ := future.CannonX(p.cannonCoordX, p.cannonSpeed, now.Sub(p.cannonStart))
			cUp := p.team == 0
			cr := unit.CannonBox(left, right, float64(cX), fieldTop, cannonBottom, cUp)
			if intersect(mr, cr) {
				//log.Printf("collision: %v %v", m, p)
				removeMissile(w, i)
				i--
				hit = true
				p.cannonLife -= .25
				if p.cannonLife <= 0 {
					w.teams[m.Team].score++
					p.cannonLife = 0    // cosmetic
					p.cannonCoordX = cX // cannon freeze
					p.cannonSpeed = 0   // cannon freeze
				}
				continue NEXT_MISSILE
			}
		}
	}

	return hit
}
