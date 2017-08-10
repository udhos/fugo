package main

import (
	"encoding/gob"
	"log"
	"net"
	"strings"
	"time"
	//"fmt"

	"golang.org/x/mobile/app"
	"golang.org/x/net/ipv4"

	"github.com/udhos/fugo/msg"
)

func serverHandler(a app.App, serverAddr string, output <-chan msg.Button) {
	log.Printf("serverHandler: starting %s", serverAddr)

	// the reconnect loop switches between trying to connect to:
	// 1. address returned as response for UDP request to 239.1.1.1:8888 -- enables easy arena server on LAN
	// 2. serverAddr (loaded from file assets/server.txt) -- enables connection to public server
	discovery := false

	// reconnect loop
	for {
		server := serverAddr
		discovery = !discovery
		if discovery {
			addr, errReq := request() // request to LAN discovery at 239.1.1.1:8888
			if errReq == nil {
				server = addr
				log.Printf("serverHandler: discovery: %s", server)
			} else {
				log.Printf("serverHandler: discovery error: %v", errReq)
			}
		}

		proto := "tcp"

		log.Printf("serverHandler: opening %s %s", proto, server)
		conn, errDial := net.DialTimeout(proto, server, 2*time.Second)
		if errDial != nil {
			log.Printf("serverHandler: error %s: %v", server, errDial)
		} else {
			log.Printf("serverHandler: connected %s", server)
			quitWriter := make(chan struct{})
			go writeLoop(conn, quitWriter, output) // spawn writer
			readLoop(a, conn)                      // loop reader
			conn.Close()
			close(quitWriter)
		}

		time.Sleep(2 * time.Second) // reconnect delay - do not hammer the server
	}
}

func request() (string, error) {
	timeout := 2 * time.Second

	destAddr, errDest := net.ResolveUDPAddr("udp", "239.1.1.1:8888")
	if errDest != nil {
		return "", errDest
	}

	/*
		c, errDial := net.DialTimeout("udp", "239.1.1.1:8888", timeout)
		if errDial != nil {
		   return "", errDial
		}
	*/

	conn, errListen := net.ListenUDP("udp", nil)
	if errListen != nil {
		return "", errListen
	}

	/*
		conn, ok := c.(*net.UDPConn)
		if !ok {
		   return "", fmt.Errorf("not a UDP conn: %v", conn)
		}
	*/

	pc := ipv4.NewPacketConn(conn)

	if errLoop := pc.SetMulticastLoopback(true); errLoop != nil {
		log.Printf("SetMulticastLoopback error: %v", errLoop)
	}

	if errTTL := pc.SetTTL(5); errTTL != nil {
		log.Printf("SetTTL error: %v", errTTL)
	}

	if errSet := conn.SetDeadline(time.Now().Add(timeout)); errSet != nil {
		return "", errSet
	}

	_, errWrite := conn.WriteTo([]byte("request\n"), destAddr)
	if errWrite != nil {
		return "", errWrite
	}

	buf := make([]byte, 1000)

	if errSet := conn.SetDeadline(time.Now().Add(timeout)); errSet != nil {
		return "", errSet
	}

	log.Printf("discovery request sent")

	n, src, errRead := conn.ReadFrom(buf)
	if errRead != nil {
		return "", errRead
	}

	srcHost := strings.Split(src.String(), ":")[0]

	listen := strings.TrimSpace(string(buf[:n]))
	hp := strings.Split(listen, ":")
	var port string
	if len(hp) > 1 {
		port = hp[1]
	}

	log.Printf("discovery response received: src=%s listen=%s", src.String(), listen)

	return srcHost + ":" + port, nil
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
		//log.Printf("readLoop: received: %v", m)
		a.Send(m)
	}
	log.Printf("readLoop: exiting")
}

func writeLoop(conn net.Conn, quit <-chan struct{}, output <-chan msg.Button) {
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
