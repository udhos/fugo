package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/udhos/fugo/msg"
)

type world struct {
	playerTab []*player
	playerAdd chan *player
	playerDel chan *player
	input     chan inputMsg
}

type inputMsg struct {
	player *player
	msg    interface{}
}

type player struct {
	conn      net.Conn
	output    chan msg.Update
	fuelStart time.Time
}

func main() {
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	w := world{
		playerTab: []*player{},
		playerAdd: make(chan *player),
		playerDel: make(chan *player),
	}
	if errListen := listenAndServe(&w, addr); errListen != nil {
		log.Printf("main: %v", errListen)
		return
	}

	ticker := time.NewTicker(3000 * time.Millisecond)

	log.Printf("main: entering service loop")
SERVICE:
	for {
		select {
		case p := <-w.playerAdd:
			log.Printf("player add: %v", p)
			w.playerTab = append(w.playerTab, p)
			p.fuelStart = time.Now() // reset fuel
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
		case t := <-ticker.C:
			log.Printf("tick: %v", t)
			for i, c := range w.playerTab {
				// calculate fuel for player c
				rechargeRate := float32(1.0 / 3.0) // 1 unit every 3 seconds
				fuel := rechargeRate * float32(int64(time.Since(c.fuelStart))/1000000000)
				if fuel > 10.0 {
					fuel = 10.0 // clamp max fuel
				}
				update := msg.Update{Fuel: fuel}
				log.Printf("sending update=%v to player %d=%v", update, i, c)
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
			var m msg.Update
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
