package main

import (
	"log"
	"net"
)

func lanDiscovery(addr string) error {

	listen := "239.1.1.1:8888"
	proto := "udp"

	log.Printf("discovery service on %s %s", proto, listen)

	udpAddr, errAddr := net.ResolveUDPAddr(proto, listen)
	if errAddr != nil {
		return errAddr
	}

	conn, errListen := net.ListenMulticastUDP(proto, nil, udpAddr)
	if errAddr != nil {
		return errListen
	}

	buf := make([]byte, 1000)

	go func() {
		for {
			_, src, errRead := conn.ReadFromUDP(buf)
			if errRead != nil {
				log.Printf("read error from %v: %v", src, errRead)
				continue
			}
			_, errWrite := conn.WriteTo([]byte(addr), src)
			if errWrite != nil {
				log.Printf("write error to %v: %v", src, errWrite)
			}
			log.Printf("discovery: replied %s to %v", addr, src)
		}
	}()

	return nil
}
