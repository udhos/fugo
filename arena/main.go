package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"image"
	_ "image/png" // The _ means to import a package purely for its initialization side effects.
	"log"
	"net"
	"os"
	"time"

	"github.com/udhos/fugo/future"
	"github.com/udhos/fugo/msg"
	"github.com/udhos/fugo/unit"
	"github.com/udhos/fugo/version"
)

type world struct {
	playerTab      []*player
	playerAdd      chan *player
	playerDel      chan *player
	input          chan inputMsg
	updateInterval time.Duration
	missileList    []*msg.Missile
	brickList      []*msg.Brick
	teams          [2]team
	cannonWidth    float64
	cannonHeight   float64
	missileWidth   float64
	missileHeight  float64
	brickWidth     float64
	brickHeight    float64
	brickID        int
}

type team struct {
	count int // player count
	score int // team score
}

type inputMsg struct {
	player *player
	msg    interface{}
}

type player struct {
	conn         net.Conn
	output       chan msg.Update
	fuelStart    time.Time
	cannonStart  time.Time
	cannonSpeed  float32
	cannonCoordX float32
	cannonLife   float32
	cannonID     int
	team         int
}

func main() {

	log.Printf("arena version " + version.Version)

	var addr string

	flag.StringVar(&addr, "addr", ":8080", "listen address")

	flag.Parse()

	w := world{
		playerTab:      []*player{},
		playerAdd:      make(chan *player),
		playerDel:      make(chan *player),
		updateInterval: 1000 * time.Millisecond,
		input:          make(chan inputMsg),
	}

	cannon := "assets/ship.png"
	var errCanSz error
	w.cannonWidth, w.cannonHeight, errCanSz = loadSize(cannon, unit.ScaleCannon)
	if errCanSz != nil {
		log.Printf("collision will NOT work: %v", errCanSz)
	}
	log.Printf("cannon: %s: %vx%v", cannon, w.cannonWidth, w.cannonHeight)

	missile := "assets/rocket.png"
	var errMisSz error
	w.missileWidth, w.missileHeight, errMisSz = loadSize(missile, unit.ScaleMissile)
	if errMisSz != nil {
		log.Printf("collision will NOT work: %v", errMisSz)
	}
	log.Printf("missile: %s: %vx%v", missile, w.missileWidth, w.missileHeight)

	brick := "assets/brick.png"
	var errBrickSz error
	w.brickWidth, w.brickHeight, errBrickSz = loadSize(brick, unit.ScaleBrick)
	if errBrickSz != nil {
		log.Printf("collision will NOT work: %v", errBrickSz)
	}
	log.Printf("brick: %s: %vx%v", brick, w.brickWidth, w.brickHeight)

	if errListen := listenAndServe(&w, addr); errListen != nil {
		log.Printf("main: listen: %v", errListen)
		return
	}

	if errDisc := lanDiscovery(addr); errDisc != nil {
		log.Printf("main: discovery: %v", errDisc)
		return
	}

	missileID := 0
	cannonID := 0

	tickerUpdate := time.NewTicker(w.updateInterval)
	tickerCollision := time.NewTicker(250 * time.Millisecond)

	log.Printf("main: entering service loop")
SERVICE:
	for {
		select {
		case p := <-w.playerAdd:
			p.team = 0
			if w.teams[0].count > w.teams[1].count {
				p.team = 1
			}
			log.Printf("player add: %v team=%d team0=%d team1=%d", p, p.team, w.teams[0].count, w.teams[1].count)
			w.playerTab = append(w.playerTab, p)

			playerFuelSet(p, time.Now(), 5) // reset fuel to 50%
			p.cannonStart = p.fuelStart
			p.cannonSpeed = float32(.15) // 15%
			p.cannonCoordX = .5          // 50%
			p.cannonID = cannonID
			p.cannonLife = 1 // 100%
			cannonID++
			w.teams[p.team].count++
		case p := <-w.playerDel:
			log.Printf("player del: %v team=%d team0=%d team1=%d", p, p.team, w.teams[0].count, w.teams[1].count)
			for i, pl := range w.playerTab {
				if pl == p {
					//w.playerTab = append(w.playerTab[:i], w.playerTab[i+1:]...)
					if i < len(w.playerTab)-1 {
						w.playerTab[i] = w.playerTab[len(w.playerTab)-1]
					}
					w.playerTab = w.playerTab[:len(w.playerTab)-1]
					w.teams[p.team].count--
					log.Printf("player removed: %v", p)
					continue SERVICE
				}
			}
			log.Printf("player not found: %v", p)
		case i := <-w.input:
			//log.Printf("input: %v", i)

			switch m := i.msg.(type) {
			case msg.Button:
				log.Printf("input button: %v", m)

				if i.player.cannonLife <= 0 {
					continue // cannon destroyed
				}

				if m.ID == msg.ButtonTurn {
					p := i.player
					updateCannon(p, time.Now())
					p.cannonSpeed = -p.cannonSpeed
					updateWorld(&w, false)
					continue SERVICE
				}

				now := time.Now()
				fuel := playerFuel(i.player, now)

				if m.ID == msg.ButtonBrick {
					if fuel < 2 {
						continue SERVICE // not enough fuel
					}
					playerFuelConsume(i.player, now, 2)
					spawnBricks(&w, i.player, now)
					continue SERVICE
				}

				if m.ID != msg.ButtonFire {
					continue SERVICE // non-fire button
				}

				if fuel < 1 {
					continue SERVICE // not enough fuel
				}

				playerFuelConsume(i.player, now, 1)

				updateCannon(i.player, now)
				miss1 := &msg.Missile{
					ID:     missileID,
					CoordX: i.player.cannonCoordX,
					Speed:  .5, // 50% every 1 second
					Team:   i.player.team,
					Start:  now,
				}
				missileID++
				w.missileList = append(w.missileList, miss1)

				log.Printf("input fire - fuel was=%v is=%v missiles=%d", fuel, playerFuel(i.player, now), len(w.missileList))

				updateWorld(&w, true)
			}

		case <-tickerUpdate.C:
			//log.Printf("tick: %v", t)

			updateWorld(&w, false)
		case <-tickerCollision.C:
			if detectCollision(&w, time.Now()) {
				updateWorld(&w, false)
			}
		}
	}
}

func spawnBricks(w *world, p *player, now time.Time) {
	updateCannon(p, now)
	rows := 1
	cols := 1
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			br := &msg.Brick{
				ID:     w.brickID,
				CoordX: p.cannonCoordX, // FIXME
				CoordY: 0,              // FIXME
				Team:   p.team,
			}
			w.brickID++
			w.brickList = append(w.brickList, br)
		}
	}
}

func loadSize(name string, scale float64) (float64, float64, error) {
	bogus := image.Rect(0, 0, 10, 10)
	w, h := unit.BoxSize(bogus, scale)

	f, errOpen := os.Open(name)
	if errOpen != nil {
		return w, h, fmt.Errorf("loadSize: open: %s: %v", name, errOpen)
	}
	defer f.Close()
	img, _, errDec := image.Decode(f)
	if errDec != nil {
		return w, h, fmt.Errorf("loadSize: decode: %s: %v", name, errDec)
	}
	i, ok := img.(*image.NRGBA)
	if !ok {
		return w, h, fmt.Errorf("loadSize: %s: not NRGBA", name)
	}

	w, h = unit.BoxSize(i, scale)
	b := i.Bounds()

	log.Printf("loadSize: %s: %vx%v => %vx%v", name, b.Max.X, b.Max.Y, w, h)

	return w, h, nil
}

func updateCannon(p *player, now time.Time) {
	p.cannonCoordX, p.cannonSpeed = future.CannonX(p.cannonCoordX, p.cannonSpeed, time.Since(p.cannonStart))
	p.cannonStart = now
}

func removeMissile(w *world, i int) {
	last := len(w.missileList) - 1
	if i < last {
		w.missileList[i] = w.missileList[last]
	}
	w.missileList = w.missileList[:last]
}

func updateWorld(w *world, fire bool) {
	now := time.Now()

	for _, p := range w.playerTab {
		updateCannon(p, now)
	}

	for i := 0; i < len(w.missileList); i++ {
		m := w.missileList[i]
		m.CoordY = future.MissileY(m.CoordY, m.Speed, time.Since(m.Start))
		m.Start = now
		if m.CoordY >= 1 {
			removeMissile(w, i)
			i--
		}
	}

	for _, p := range w.playerTab {
		sendUpdatesToPlayer(w, p, now, fire)
	}
}

func playerFuel(p *player, now time.Time) float32 {
	return future.Fuel(0, now.Sub(p.fuelStart))
}

func playerFuelSet(p *player, now time.Time, fuel float32) {
	p.fuelStart = now.Add(-time.Duration(float32(time.Second) * fuel / future.FuelRechargeRate))
}

func playerFuelConsume(p *player, now time.Time, amount float32) {
	fuel := playerFuel(p, now)
	playerFuelSet(p, now, fuel-amount)
}

func sendUpdatesToPlayer(w *world, p *player, now time.Time, fire bool) {
	update := msg.Update{
		Fuel:          playerFuel(p, now),
		Interval:      w.updateInterval,
		WorldMissiles: w.missileList,
		Bricks:        w.brickList,
		Team:          p.team,
		Scores:        [2]int{w.teams[0].score, w.teams[1].score},
		FireSound:     fire,
	}

	for _, p1 := range w.playerTab {
		cannon := msg.Cannon{
			ID:     p1.cannonID,
			Start:  p1.cannonStart,
			CoordX: p1.cannonCoordX,
			Speed:  p1.cannonSpeed,
			Team:   p1.team,
			Life:   p1.cannonLife,
			Player: p1 == p,
		}
		update.Cannons = append(update.Cannons, &cannon)
	}

	//log.Printf("sending updates to player %v", p)

	p.output <- update
}

func listenAndServe(w *world, addr string) error {

	proto := "tcp"

	log.Printf("serving on %s %s", proto, addr)

	listener, errListen := net.Listen(proto, addr)
	if errListen != nil {
		return fmt.Errorf("listenAndServe: %s: %v", addr, errListen)
	}

	gob.Register(msg.Update{})
	gob.Register(msg.Button{})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("accept on TCP %s: %s", addr, err)
				continue
			}
			c, _ := conn.(*net.TCPConn)
			go connHandler(w, c)
		}
	}()

	return nil
}

func connHandler(w *world, conn *net.TCPConn) {
	log.Printf("handler for connection %v", conn.RemoteAddr())

	defer conn.Close()

	p := &player{
		conn:   conn,
		output: make(chan msg.Update),
	}

	w.playerAdd <- p // register player
	quitWriter := make(chan struct{})

	go func() {
		// copy from socket into input channel
		dec := gob.NewDecoder(conn)
		for {
			var m msg.Button
			if err := dec.Decode(&m); err != nil {
				log.Printf("handler: Decode: %v", err)
				break
			}
			w.input <- inputMsg{player: p, msg: m}
		}
		close(quitWriter) // send quit request to output goroutine
		log.Printf("handler: reader goroutine exiting")
	}()

	// copy from output channel into socket
	enc := gob.NewEncoder(conn)
LOOP:
	for {
		select {
		case <-quitWriter:
			log.Printf("handler: quit request")
			break LOOP
		case m := <-p.output:
			if err := enc.Encode(&m); err != nil {
				log.Printf("handler: Encode: %v", err)
				break LOOP
			}
		}
	}
	w.playerDel <- p // deregister player
	log.Printf("handler: writer goroutine exiting")
}
