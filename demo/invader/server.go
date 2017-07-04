package main

import (
	"encoding/gob"
	"log"
	"net"
	"time"

	"golang.org/x/mobile/app"

	"github.com/udhos/fugo/msg"
)

func serverHandler(a app.App, serverAddr string) {
	log.Printf("serverHandler: starting %s", serverAddr)

	// reconnect loop
	for {
		log.Printf("serverHandler: opening %s", serverAddr)
		conn, errDial := net.Dial("tcp", serverAddr)
		if errDial != nil {
			log.Printf("serverHandler: error %s: %v", serverAddr, errDial)
		} else {
			log.Printf("serverHandler: connected %s", serverAddr)
			readLoop(a, conn)
			conn.Close()
		}
		time.Sleep(2 * time.Second) // reconnect delay - do not hammer the server
	}
}

func readLoop(a app.App, conn net.Conn) {
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
}
