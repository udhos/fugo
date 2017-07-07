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

	log.Printf("main: entering service loop")
SERVICE:
	for {
		select {
		case p := <-w.playerAdd:
			log.Printf("player add: %v", p)
			w.playerTab = append(w.playerTab, p)

			p.fuelStart = time.Now() // reset fuel
			p.cannonStart = p.fuelStart
			p.cannonSpeed = float32(.1 / 1.0) // 10% every 1 second
			p.cannonCoordX = .8               // 80%
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
			}

		case <-ticker.C:
			//log.Printf("tick: %v", t)

			for i, c := range w.playerTab {
				// calculate fuel for player c
				/*
					rechargeRate := float32(1.0 / 3.0) // 1 unit every 3 seconds
					fuel := rechargeRate * float32(int64(time.Since(c.fuelStart))/1000000000)
					if fuel > 10.0 {
						fuel = 10.0 // clamp max fuel
					}
				*/
				fuel := future.Fuel(0, time.Since(c.fuelStart))

				// calculate position
				/*
					speed := float32(.05 / 1.0) // 1% every 1 second
					delta := speed * float32(int64(time.Since(c.cannonStart))/1000000000)
					c.cannonCoordX += delta * c.cannonDir
					if c.cannonCoordX < 0 {
						c.cannonCoordX = -c.cannonCoordX
						c.cannonDir = 1
					}
					if c.cannonCoordX > 1 {
						c.cannonCoordX = 2 - c.cannonCoordX
						c.cannonDir = -1
					}
				*/
				c.cannonCoordX, c.cannonSpeed = future.CannonX(c.cannonCoordX, c.cannonSpeed, time.Since(c.cannonStart))
				c.cannonStart = time.Now()

				update := msg.Update{Fuel: fuel,
					CannonX:     c.cannonCoordX,
					CannonSpeed: c.cannonSpeed,
					Interval:    w.updateInterval,
				}

				log.Printf("sending update=%v to player %d", update, i)
				c.output <- update // send update to player c
			}
		}
	}
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
	quitWriter := make(chan int)

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
