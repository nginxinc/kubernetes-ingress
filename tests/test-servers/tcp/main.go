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

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading:", err.Error())
		conn.Close()
		return
	}
	instruction := string(buf[:n])
	log.Printf("instruction:%q\n", instruction)
	if instruction != "hold" {
		defer conn.Close()
		defer log.Println("closed connection")
	}

	response := conn.LocalAddr().String()
	if instruction == "health" {
		response = "healthy"
	}

	log.Printf("write data to connection: %v\n", response)

	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Printf("error writing to connection: %v", err)
		return
	}
}
