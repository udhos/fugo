package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
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
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	w := world{
		playerTab:      []*player{},
		playerAdd:      make(chan *player),
		playerDel:      make(chan *player),
		updateInterval: 1000 * time.Millisecond,
		input:          make(chan inputMsg),
	}
	if errListen := listenAndServe(&w, addr); errListen != nil {
		log.Printf("main: %v", errListen)
		return
	}

	ticker := time.NewTicker(w.updateInterval)

	team := 0

	log.Printf("main: entering service loop")
SERVICE:
	for {
		select {
		case p := <-w.playerAdd:
			log.Printf("player add: %v team=%d", p, team)
			w.playerTab = append(w.playerTab, p)

			//p.fuelStart = time.Now() // reset fuel
			playerFuelSet(p, time.Now(), 5) // reset fuel to 50%
			p.cannonStart = p.fuelStart
			p.cannonSpeed = float32(.1 / 1.0) // 10% every 1 second
			p.cannonCoordX = .8               // 80%
			p.team = team
			team = (team + 1) % 2 // switch next team
		case p := <-w.playerDel:
			log.Printf("player del: %v", p)
			for i, pl := range w.playerTab {
				if pl == p {
					w.playerTab = append(w.playerTab[:i], w.playerTab[i+1:]...)
					log.Printf("player removed: %v", p)
					continue SERVICE
				}
			}
			log.Printf("player not found: %v", p)
		case i := <-w.input:
			log.Printf("input: %v", i)

			switch m := i.msg.(type) {
			case msg.Fire:
				log.Printf("input fire: %v", m)

				fuel := playerFuel(i.player)
				if fuel < 1 {
					continue SERVICE // not enough fuel
				}

				if fuel >= 10 {
					//i.player.fuelStart = time.Now().Add(-time.Duration(float32(time.Second) * 10 / future.FuelRechargeRate))
					playerFuelSet(i.player, time.Now(), 9)
				} else {
					playerFuelSet(i.player, time.Now(), fuel-1)
				}
				//i.player.fuelStart = i.player.fuelStart.Add(time.Duration(float32(time.Second) / future.FuelRechargeRate))

				missileSpeed := float32(.5 / 1.0) // 50% every 1 second
				now := time.Now()
				miss1 := &msg.Missile{
					CoordX: i.player.cannonCoordX,
					Speed:  missileSpeed,
					Team:   i.player.team,
					Start:  now,
				}
				w.missileList = append(w.missileList, miss1)

				log.Printf("fuel was=%v is=%v missiles=%d", fuel, playerFuel(i.player), len(w.missileList))

				updateWorld(&w)
			}

		case <-ticker.C:
			//log.Printf("tick: %v", t)

			updateWorld(&w)
		}
	}
}

func updateWorld(w *world) {
	now := time.Now()
	for _, p := range w.playerTab {
		p.cannonCoordX, p.cannonSpeed = future.CannonX(p.cannonCoordX, p.cannonSpeed, time.Since(p.cannonStart))
		p.cannonStart = now
	}

	size := len(w.missileList)
	for i := 0; i < size; i++ {
		m := w.missileList[i]
		m.CoordY = future.MissileY(0, m.Speed, time.Since(m.Start))
		if m.CoordY >= 1 {
			size--
			if i >= size {
				// last element
				break
			}
			w.missileList[i] = w.missileList[i+1]
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
		Fuel: playerFuel(p),
		//CannonX:       p.cannonCoordX,
		//CannonSpeed:   p.cannonSpeed,
		Interval:      w.updateInterval,
		WorldMissiles: w.missileList,
		Team:          p.team,
	}

	for _, p := range w.playerTab {
		cannon := msg.Cannon{
			Start:  p.cannonStart,
			CoordX: p.cannonCoordX,
			Speed:  p.cannonSpeed,
			Team:   p.team,
		}
		update.Cannons = append(update.Cannons, &cannon)
	}

	//log.Printf("sending updates to player %v", p)

	p.output <- update
}

func listenAndServe(w *world, addr string) error {

	log.Printf("serving on TCP %s", addr)

	listener, errListen := net.Listen("tcp", addr)
	if errListen != nil {
		return fmt.Errorf("listenAndServe: %s: %v", addr, errListen)
	}

	gob.Register(msg.Update{})
	gob.Register(msg.Fire{})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("accept on TCP %s: %s", addr, err)
				continue
			}
			go connHandler(w, conn)
		}
	}()

	return nil
}

func connHandler(w *world, conn net.Conn) {
	log.Printf("handler for connection %v", conn)

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
			var m msg.Fire
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
