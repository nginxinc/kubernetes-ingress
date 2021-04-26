package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	ip := os.Getenv("POD_IP")
	log.Printf("ip: %v\n", ip)
	if ip == "" {
		log.Fatalf("missing required env var: POD_IP")
	}
	port := flag.String("port", "3334", "The port the server listens to")
	flag.Parse()
	listener, err := net.ListenPacket("udp", fmt.Sprintf(":%v", *port))
	if err != nil {
		log.Panicln(err)
	}
	defer listener.Close()
	log.Printf("listening to udp connections at: :%v\n", *port)
	buffer := make([]byte, 1024)
	for {
		n, addr, err := listener.ReadFrom(buffer)
		if err != nil {
			log.Panicln(err)
		}

		fmt.Printf("packet-received: bytes=%d from=%s\n", n, addr.String())
		address := fmt.Sprintf("%v:%v", ip, *port)
		log.Printf("write data to connection: %v\n", address)
		n, err = listener.WriteTo([]byte(address), addr)
		if err != nil {
			log.Panicln(err)
		}
		fmt.Printf("packet-written: bytes=%d to=%s\n", n, addr.String())
	}
}
