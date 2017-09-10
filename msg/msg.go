package msg

import (
	"time"
)

// Update message is sent from server do client.
type Update struct {
	Fuel          float32
	Interval      time.Duration // notify client about update interval
	WorldMissiles []*Missile
	Cannons       []*Cannon
	Bricks        []*Brick
	Team          int // notify player about his team
	Scores        [2]int
	FireSound     bool
}

const (
	// ButtonFire ID
	ButtonFire = 0
	// ButtonTurn ID
	ButtonTurn = 1
	// ButtonBrick ID
	ButtonBrick = 2
)

// Button message is sent from client to server.
type Button struct {
	ID int
}

// Missile is issued by cannons.
type Missile struct {
	ID     int
	CoordX float32
	CoordY float32
	Speed  float32
	Team   int
	Start  time.Time
}

// Brick is issued by cannons.
type Brick struct {
	ID     int
	CoordX float32
	CoordY float32
	Team   int
}

// Cannon belongs to player.
type Cannon struct {
	ID     int
	Start  time.Time
	CoordX float32
	Speed  float32
	Team   int
	Player bool // belongs to player
	Life   float32
}
