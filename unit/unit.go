package unit

const (
	// MissileWidth dimension
	MissileWidth = .03
	// MissileHeight dimension
	MissileHeight = .07
	// CannonWidth dimension
	CannonWidth = .1
	// CannonHeight dimension
	CannonHeight = .1
)

// Rect is bounding box.
type Rect struct {
	X1, Y1, X2, Y2 float64
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
