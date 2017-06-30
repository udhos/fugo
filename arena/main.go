package main

import (
	//"fmt"
	"encoding/gob"
	"log"
	"net"
	"os"
	"time"
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
	go listenAndServe(&w, addr)

	ticker := time.NewTicker(3000 * time.Millisecond)

	log.Printf("main: entering service loop")
SERVICE:
	for {
		select {
		case p := <-w.playerAdd:
			log.Printf("player add: %v", p)
			w.playerTab = append(w.playerTab, p)
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
		}
	}
}

func listenAndServe(w *world, addr string) {

	log.Printf("serving on port TCP %s", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("listen on TCP %s: %s", addr, err)
		return
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept on TCP %s: %s", addr, err)
			continue
		}
		go handler(w, conn)
	}
}

func handler(w *world, conn net.Conn) {
	log.Printf("handler for connection %v", conn)

	defer conn.Close()

	p := &player{
		conn:   conn,
		output: make(chan interface{}),
	}

	w.playerAdd <- p // register player
	quitWriter := make(chan int)

	go func() {
		// copy from socket into input channel
		dec := gob.NewDecoder(conn)
		for {
			var m struct{}
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

type player struct {
	conn   net.Conn
	output chan interface{}
}
