package main

import (
	"log"
	"net"
)

func lanDiscovery(addr string) error {

	listen := "239.1.1.1:8888"
	proto := "udp"

	log.Printf("discovery service reporting %s on %s %s", addr, proto, listen)

	udpAddr, errAddr := net.ResolveUDPAddr(proto, listen)
	if errAddr != nil {
		return errAddr
	}

	conn, errListen := net.ListenMulticastUDP(proto, nil, udpAddr)
	if errListen != nil {
		return errListen
	}

	go func() {
		buf := make([]byte, 1000)
		for {
			_, src, errRead := conn.ReadFromUDP(buf)
			if errRead != nil {
				log.Printf("discovery read error from %v: %v", src, errRead)
				continue
			}
			_, errWrite := conn.WriteTo([]byte(addr), src)
			if errWrite != nil {
				log.Printf("discovery write error to %v: %v", src, errWrite)
				continue
			}
			log.Printf("discovery: replied %s to %v", addr, src)
		}
	}()

	return nil
}
