package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/udhos/fugo/future"
	"github.com/udhos/fugo/msg"
)

type world struct {
	playerTab      []*player
	playerAdd      chan *player
	playerDel      chan *player
	input          chan inputMsg
	updateInterval time.Duration
	missileList    []*msg.Missile
	teamCount      [2]int
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
	team         int
}

func main() {

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
	if errListen := listenAndServe(&w, addr); errListen != nil {
		log.Printf("main: listen: %v", errListen)
		return
	}

	if errDisc := lanDiscovery(addr); errDisc != nil {
		log.Printf("main: discovery: %v", errDisc)
		return
	}

	ticker := time.NewTicker(w.updateInterval)

	log.Printf("main: entering service loop")
SERVICE:
	for {
		select {
		case p := <-w.playerAdd:
			p.team = 0
			if w.teamCount[0] > w.teamCount[1] {
				p.team = 1
			}
			log.Printf("player add: %v team=%d team0=%d team1=%d", p, p.team, w.teamCount[0], w.teamCount[1])
			w.playerTab = append(w.playerTab, p)

			playerFuelSet(p, time.Now(), 5) // reset fuel to 50%
			p.cannonStart = p.fuelStart
			p.cannonSpeed = float32(.1 / 1.0) // 10% every 1 second
			p.cannonCoordX = .5               // 50%
			w.teamCount[p.team]++
		case p := <-w.playerDel:
			log.Printf("player del: %v team=%d team0=%d team1=%d", p, p.team, w.teamCount[0], w.teamCount[1])
			for i, pl := range w.playerTab {
				if pl == p {
					w.playerTab = append(w.playerTab[:i], w.playerTab[i+1:]...)
					w.teamCount[p.team]--
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

				if m.ID == msg.ButtonTurn {
					p := i.player
					//p.cannonCoordX, p.cannonSpeed = future.CannonX(p.cannonCoordX, p.cannonSpeed, time.Since(p.cannonStart))
					updateCannon(p, time.Now())
					p.cannonSpeed = -p.cannonSpeed
					updateWorld(&w)
					continue SERVICE
				}

				if m.ID != msg.ButtonFire {
					continue SERVICE // non-fire button
				}

				fuel := playerFuel(i.player)
				if fuel < 1 {
					continue SERVICE // not enough fuel
				}

				if fuel >= 10 {
					playerFuelSet(i.player, time.Now(), 9)
				} else {
					playerFuelSet(i.player, time.Now(), fuel-1)
				}

				now := time.Now()
				updateCannon(i.player, now)
				miss1 := &msg.Missile{
					CoordX: i.player.cannonCoordX,
					Speed:  .5, // 50% every 1 second
					Team:   i.player.team,
					Start:  now,
				}
				w.missileList = append(w.missileList, miss1)

				log.Printf("input fire - fuel was=%v is=%v missiles=%d", fuel, playerFuel(i.player), len(w.missileList))

				updateWorld(&w)
			}

		case <-ticker.C:
			//log.Printf("tick: %v", t)

			updateWorld(&w)
		}
	}
}

func updateCannon(p *player, now time.Time) {
	p.cannonCoordX, p.cannonSpeed = future.CannonX(p.cannonCoordX, p.cannonSpeed, time.Since(p.cannonStart))
	p.cannonStart = now
}

func updateWorld(w *world) {
	now := time.Now()
	for _, p := range w.playerTab {
		updateCannon(p, now)
	}

	size := len(w.missileList)
	for i := 0; i < size; i++ {
		m := w.missileList[i]
		m.CoordY = future.MissileY(m.CoordY, m.Speed, time.Since(m.Start))
		m.Start = now
		if m.CoordY >= 1 {
			size--
			if i >= size {
				// last element
				break
			}
			w.missileList[i] = w.missileList[size]
			i--
		}
	}
	w.missileList = w.missileList[:size]

	for _, p := range w.playerTab {
		sendUpdatesToPlayer(w, p)
	}
}

func playerFuel(p *player) float32 {
	return future.Fuel(0, time.Since(p.fuelStart))
}

func playerFuelSet(p *player, now time.Time, fuel float32) {
	p.fuelStart = now.Add(-time.Duration(float32(time.Second) * fuel / future.FuelRechargeRate))
}

func sendUpdatesToPlayer(w *world, p *player) {
	update := msg.Update{
		Fuel:          playerFuel(p),
		Interval:      w.updateInterval,
		WorldMissiles: w.missileList,
		Team:          p.team,
	}

	for _, p1 := range w.playerTab {
		cannon := msg.Cannon{
			Start:  p1.cannonStart,
			CoordX: p1.cannonCoordX,
			Speed:  p1.cannonSpeed,
			Team:   p1.team,
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
