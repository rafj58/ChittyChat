package main

import (
	proto "ChittyChat/grpc"
	"flag"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedProtoServer
	name string
	port int
}

var port = flag.Int("port", 0, "server port number")

func main() {
	flag.Parse()

	server := &Server{
		name: "ChittyChat",
		port: *port,
	}

	go startServer(server)

	for {

	}
}

func startServer(server *Server) {
	grpcServer := grpc.NewServer()

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))

	if err != nil {
		log.Fatalf("Could not create the server %v", err)
	}
	log.Printf("Started %s server at port: %d\n", server.name, server.port)

	proto.RegisterProtoServer(grpcServer, server)
	serveError := grpcServer.Serve(listener)

	if serveError != nil {
		log.Fatalf("Could not serve listener")
	}
}
