package main

import (
	"crypto/tls"
	"flag"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

var address = flag.String("address", "grpc.example.com:8443", "address")
var tlsFlag = flag.Bool("tls", true, "tls")

const defaultName = "world"

func main() {
	flag.Parse()
	// Set up creds but skip verification of cert/key
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	// Set up a connection to the server.
	var err error
	var conn *grpc.ClientConn
	if *tlsFlag {
		conn, err = grpc.Dial(*address, grpc.WithTransportCredentials(creds))
	} else {
		conn, err = grpc.Dial(*address, grpc.WithInsecure())
	}
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
