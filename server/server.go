package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	proto "ChittyChat/grpc"

	"google.golang.org/grpc"
)

const ( // connection status
	Connect    int = 0
	Disconnect     = 1
	Publish        = 2
)

type Server struct {
	proto.UnimplementedChittyChatServiceServer
	name             string
	port             int
	clientReferences map[string]proto.ChittyChatService_SendMessageServer
	mutex            sync.Mutex // avoid race condition
}

// flags are used to get arguments from the terminal. Flags take a value, a default value and a description of the flag.
var port = flag.Int("port", 1000, "server port") // set with "-port <port>" in terminal

func main() {

	// Get the port from the command line when the server is run
	flag.Parse()

	// Start the server
	go startServer()

	// Keep the server running until it is manually quit
	for {

	}
}

func startServer() {
	// Create a server struct
	server := &Server{
		name: "server",
		port: *port,
	}

	// Create a new grpc server
	grpcServer := grpc.NewServer()

	// Make the server listen at the given port (convert int port to string)
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", strconv.Itoa(server.port)))

	if err != nil {
		log.Fatalf("Could not create the server %v", err)
	}
	log.Printf("Started server at port: %d\n", server.port)

	// Register the grpc service
	proto.RegisterChittyChatServiceServer(grpcServer, server)
	serveError := grpcServer.Serve(listener)

	if serveError != nil {
		log.Fatalf("Could not serve listener")
	}
}

func (s *Server) SendMessage(stream proto.ChittyChatService_SendMessageServer) error {

	if s.clientReferences == nil {
		// initialize map
		s.clientReferences = make(map[string]proto.ChittyChatService_SendMessageServer)
	}

	run := true

	for run {
		msg, err := stream.Recv()

		// client reference as a String
		clientString := msg.ClientReference.ClientAddress + ":" + strconv.Itoa(int(msg.ClientReference.ClientPort))
		log.Printf("Received message from %s", clientString)

		if err != nil {
			log.Printf("Error while receiving message %v", err)
			break
		}

		switch int(msg.Type) {
		case Connect:
			{
				// add to map
				s.clientReferences[clientString] = stream
				break
			}

		case Disconnect:
			{
				// forward the disconnect
				for clientRef, clientStream := range s.clientReferences {
					err = clientStream.Send(msg)
					if err != nil {
						log.Printf("Error during forwarding disconnection message to %s: %v", clientRef, err)
						break
					}
				}
				// remove stream from map
				delete(s.clientReferences, clientString)
				log.Printf("Client %s disconnected", clientString)
				run = false
				break
			}

		case Publish:
			{
				for clientRef, clientStream := range s.clientReferences {
					if clientRef != clientString { // Do not forward the message to the original sender
						err = clientStream.Send(msg)
						if err != nil {
							log.Printf("Error during forwarding to %s: %v", clientRef, err)
							break
						}
					}
				}
				break
			}
		}
	}
	return nil
}

// sets the logger to use a log.txt file instead of the console
func setLog() *os.File {
	// Clears the log.txt file when a new server is started
	if err := os.Truncate("log.txt", 0); err != nil {
		log.Printf("Failed to truncate: %v", err)
	}

	// This connects to the log file/changes the output of the log informaiton to the log.txt file.
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}
