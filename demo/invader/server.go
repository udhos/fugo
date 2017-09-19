package main

import (
	"encoding/gob"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/net/ipv4"

	"github.com/udhos/fugo/msg"
)

func serverHandler(a app.App, serverAddr string, output <-chan msg.Msg) {
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

	discAddr := "239.1.1.1:8888"

	destAddr, errDest := net.ResolveUDPAddr("udp", discAddr)
	if errDest != nil {
		return "", errDest
	}

	conn, errListen := net.ListenUDP("udp", nil)
	if errListen != nil {
		return "", errListen
	}

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

	log.Printf("discovery request sent to %s", discAddr)

	buf := make([]byte, 1000)

	if errSet := conn.SetDeadline(time.Now().Add(timeout)); errSet != nil {
		return "", errSet
	}

	n, src, errRead := conn.ReadFrom(buf)
	if errRead != nil {
		return "", errRead
	}

	listen := strings.TrimSpace(string(buf[:n]))

	log.Printf("discovery response received: src=%s listen=%s", src.String(), listen)

	srcAddr, errSrc := net.ResolveUDPAddr("udp", src.String())
	if errSrc != nil {
		return "", errSrc
	}
	srcHost := srcAddr.IP.String()

	listenAddr, errAddr := net.ResolveUDPAddr("udp", listen)
	if errAddr != nil {
		return "", errAddr
	}
	listenHost := listenAddr.IP.String()
	if listenHost != "<nil>" {
		if listenAddr.IP.To4() == nil {
			// IPv6
			srcHost = "[" + listenHost + "]"
		} else {
			// IPv4
			srcHost = listenHost
		}
	}

	endpoint := srcHost + ":" + strconv.Itoa(listenAddr.Port)

	return endpoint, nil
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
		a.Send(m)
	}
	log.Printf("readLoop: exiting")
}

func writeLoop(conn net.Conn, quit <-chan struct{}, output <-chan msg.Msg) {
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
		}
	}
	log.Printf("writeLoop: goroutine exiting")
}
