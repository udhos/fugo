package unit

const (
	// MissileWidth dimension
	MissileWidth = .03
	// MissileHeight dimension
	MissileHeight = .07
	// CannonWidth dimension
	CannonWidth = .15
	// CannonHeight dimension
	CannonHeight = .15
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
func CannonBox(gameMinX, gameMaxX, x, fieldTop, cannonBottom float64, up bool) Rect {
	cx := x*(gameMaxX-CannonWidth-gameMinX) + gameMinX
	var cy1, cy2 float64
	if up {
		// upward
		cy1 = cannonBottom
		cy2 = cy1 + CannonHeight
	} else {
		// downward
		cy2 = fieldTop
		cy1 = cy2 - CannonHeight
	}
	return Rect{
		X1: cx,
		Y1: cy1,
		X2: cx + CannonWidth,
		Y2: cy2,
	}
}

// MissileBox returns bounding box.
func MissileBox(gameMinX, gameMaxX, x, y, fieldTop, cannonBottom float64, up bool) Rect {
	minX := gameMinX + .5*CannonWidth - .5*MissileWidth
	maxX := gameMaxX - .5*CannonWidth - .5*MissileWidth
	fx := x*(maxX-minX) + minX
	var fy float64
	if up {
		// upward
		minY := cannonBottom + CannonHeight
		maxY := fieldTop - MissileHeight
		fy = y*(maxY-minY) + minY
	} else {
		// downward
		minY := cannonBottom
		maxY := fieldTop - CannonHeight
		fy = y*(minY-maxY) + maxY
	}
	return Rect{
		X1: fx,
		Y1: fy,
		X2: fx + MissileWidth,
		Y2: fy + MissileHeight,
	}
}
