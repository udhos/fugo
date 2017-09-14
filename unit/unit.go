package unit

import (
	"image"
)

const (
	// ScaleCannon cannon scale
	ScaleCannon = .2
	// ScaleMissile missile scale
	ScaleMissile = .15
	// ScaleBrick brick scale
	ScaleBrick = .08
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
func MissileBox(gameMinX, gameMaxX, x, y, fieldTop, cannonBottom, cannonWidth, cannonHeight, missileWidth, missileHeight float64, up bool) Rect {
	minX := gameMinX + .5*cannonWidth - .5*missileWidth
	maxX := gameMaxX - .5*cannonWidth - .5*missileWidth
	fx := x*(maxX-minX) + minX
	var fy float64
	if up {
		// upward
		minY := cannonBottom + cannonHeight
		maxY := fieldTop - missileHeight
		fy = y*(maxY-minY) + minY
	} else {
		// downward
		minY := cannonBottom
		maxY := fieldTop - cannonHeight - missileHeight
		fy = y*(minY-maxY) + maxY
	}
	return Rect{
		X1: fx,
		Y1: fy,
		X2: fx + missileWidth,
		Y2: fy + missileHeight,
	}
}

// BrickBox returns bounding box.
func BrickBox(gameMinX, gameMaxX, x, y, fieldTop, cannonBottom, cannonHeight, brickWidth, brickHeight float64, up bool) Rect {
	minX := gameMinX
	maxX := gameMaxX - brickWidth
	fx := x*(maxX-minX) + minX
	var fy float64
	if up {
		// upward
		minY := cannonBottom + cannonHeight
		maxY := fieldTop - brickHeight
		fy = y*(maxY-minY) + minY
	} else {
		// downward
		minY := cannonBottom
		maxY := fieldTop - cannonHeight - brickHeight
		fy = y*(minY-maxY) + maxY
	}
	return Rect{
		X1: fx,
		Y1: fy,
		X2: fx + brickWidth,
		Y2: fy + brickHeight,
	}
}

// Box has a bounding image.Rectangle.
type Box interface {
	Bounds() image.Rectangle
}

// BoxSize returns the width,height of bounding rectangle.
// Bounding rectangle in pixel. Resulting width,height in NDC (-1.0 to 1.0).
func BoxSize(b Box, scale float64) (float64, float64) {
	sb := b.Bounds()
	sw := sb.Max.X - sb.Min.X
	sh := sb.Max.Y - sb.Min.Y
	var sdmax int
	if sw < sh {
		sdmax = sh
	} else {
		sdmax = sw
	}
	width := scale * float64(sw) / float64(sdmax)
	height := scale * float64(sh) / float64(sdmax)
	return width, height
}
