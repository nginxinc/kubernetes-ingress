package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

func main() {
	port := flag.String("port", "3333", "Port")
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf(":%v", *port))
	if err != nil {
		log.Panicln(err)
	}
	defer l.Close()
	log.Printf("listening to tcp connections at: :%v\n", *port)
	log.Printf("responding with: %v\n", *port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Panicln(err)
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	log.Println("accepted new connection")
	defer conn.Close()
	defer log.Println("closed connection")
	address := conn.LocalAddr().String()
	log.Printf("write data to connection: %v\n", address)
	_, err := conn.Write([]byte(address))
	if err != nil {
		log.Printf("error writing to connection: %v", err)
		return
	}
}
