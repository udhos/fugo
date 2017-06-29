package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	addr := ":8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	log.Printf("serving on port TCP %s", addr)
	if err := listenAndServe(addr); err != nil {
		log.Printf("error: %v", err)
	}
}

func listenAndServe(addr string) error {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on TCP %s: %s", addr, err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept on TCP %s: %s", addr, err)
			continue
		}
		go handler(conn)
	}
}

func handler(conn net.Conn) {
	log.Printf("handler for connection %v", conn)
	conn.Close()
}
