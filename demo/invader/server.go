package main

import (
	"encoding/gob"
	"log"
	"net"
	"time"

	"golang.org/x/mobile/app"

	"github.com/udhos/fugo/msg"
)

func serverHandler(a app.App, serverAddr string, output <-chan msg.Fire) {
	log.Printf("serverHandler: starting %s", serverAddr)

	// reconnect loop
	for {
		log.Printf("serverHandler: opening %s", serverAddr)
		conn, errDial := net.Dial("tcp", serverAddr)
		if errDial != nil {
			log.Printf("serverHandler: error %s: %v", serverAddr, errDial)
		} else {
			log.Printf("serverHandler: connected %s", serverAddr)
			quitWriter := make(chan int)
			go writeLoop(conn, quitWriter, output) // spawn writer
			readLoop(a, conn)                      // loop reader
			conn.Close()
			close(quitWriter)
		}
		time.Sleep(2 * time.Second) // reconnect delay - do not hammer the server
	}
}

func readLoop(a app.App, conn net.Conn) {
	log.Printf("readLoop: entering")
	// copy from socket into event channel
	dec := gob.NewDecoder(conn)
	for {
		var m msg.Update
		if err := dec.Decode(&m); err != nil {
			log.Printf("readLoop: Decode: %v", err)
			break
		}
		log.Printf("readLoop: received: %v", m)
		a.Send(m)
	}
	log.Printf("readLoop: exiting")
}

func writeLoop(conn net.Conn, quit <-chan int, output <-chan msg.Fire) {
	log.Printf("writeLoop: goroutine starting")
	// copy from output channel into socket
	enc := gob.NewEncoder(conn)
LOOP:
	for {
		select {
		case <-quit:
			log.Printf("writeLoop: quit request")
			break LOOP
		case m := <-output:
			if err := enc.Encode(&m); err != nil {
				log.Printf("writeLoop: Encode: %v", err)
				break LOOP
			}
			//log.Printf("writeLoop: sent %v", m)
		}
	}
	log.Printf("writeLoop: goroutine exiting")
}
