package unit

import (
	"image"
)

const (
	// MissileWidth dimension
	MissileWidth = .03
	// MissileHeight dimension
	MissileHeight = .07
	// CannonWidth dimension
	//CannonWidth = .15
	// CannonHeight dimension
	//CannonHeight = .15
)

// Rect is bounding box.
type Rect struct {
	X1, Y1, X2, Y2 float64
}

// Bounding returns rectangle vertices.
func (r Rect) Bounding() (float64, float64, float64, float64) {
	return r.X1, r.Y1, r.X2, r.Y2
}

// CannonBox returns bounding box.
func CannonBox(gameMinX, gameMaxX, x, fieldTop, cannonBottom, cannonWidth, cannonHeight float64, up bool) Rect {
	cx := x*(gameMaxX-cannonWidth-gameMinX) + gameMinX
	var cy1, cy2 float64
	if up {
		// upward
		cy1 = cannonBottom
		cy2 = cy1 + cannonHeight
	} else {
		// downward
		cy2 = fieldTop
		cy1 = cy2 - cannonHeight
	}
	return Rect{
		X1: cx,
		Y1: cy1,
		X2: cx + cannonWidth,
		Y2: cy2,
	}
}

// MissileBox returns bounding box.
func MissileBox(gameMinX, gameMaxX, x, y, fieldTop, cannonBottom, cannonWidth, cannonHeight float64, up bool) Rect {
	minX := gameMinX + .5*cannonWidth - .5*MissileWidth
	maxX := gameMaxX - .5*cannonWidth - .5*MissileWidth
	fx := x*(maxX-minX) + minX
	var fy float64
	if up {
		// upward
		minY := cannonBottom + cannonHeight
		maxY := fieldTop - MissileHeight
		fy = y*(maxY-minY) + minY
	} else {
		// downward
		minY := cannonBottom
		maxY := fieldTop - cannonHeight
		fy = y*(minY-maxY) + maxY
	}
	return Rect{
		X1: fx,
		Y1: fy,
		X2: fx + MissileWidth,
		Y2: fy + MissileHeight,
	}
}

type Box interface {
	Bounds() image.Rectangle
}

func CannonSize(b Box) (float64, float64) {
	sb := b.Bounds()
	sw := sb.Max.X - sb.Min.X
	sh := sb.Max.Y - sb.Min.Y
	var sdmax int
	if sw < sh {
		sdmax = sh
	} else {
		sdmax = sw
	}
	cannonSize := .4
	cannonWidth := cannonSize * float64(sw) / float64(sdmax)
	cannonHeight := cannonSize * float64(sh) / float64(sdmax)
	return cannonWidth, cannonHeight
}
