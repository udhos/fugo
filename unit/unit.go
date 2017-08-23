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
func MissileBox(screen Rect, x, y, fieldTop, cannonBottom float64, up bool) Rect {
	gameMinX := screen.X1
	gameMinY := screen.Y1
	gameMaxX := screen.X2
	gameMaxY := screen.Y2
	screenWidth := gameMaxX - gameMinX
	screenHeight := gameMaxY - gameMinY
	minX := gameMinX + .5*CannonWidth*screenWidth - .5*MissileWidth*screenWidth
	maxX := gameMaxX - .5*CannonWidth*screenWidth - .5*MissileWidth*screenWidth
	fx := x*(maxX-minX) + minX
	var fy float64
	if up {
		// upward
		minY := cannonBottom + CannonHeight*screenHeight
		maxY := fieldTop - MissileHeight*screenHeight
		fy = y*(maxY-minY) + minY
	} else {
		// downward
		minY := cannonBottom
		maxY := fieldTop - CannonHeight*screenHeight
		fy = y*(minY-maxY) + maxY

	}
	return Rect{
		X1: fx,
		Y1: fy,
		X2: fx + MissileWidth*screenWidth,
		Y2: fy + MissileHeight*screenHeight,
	}
}
