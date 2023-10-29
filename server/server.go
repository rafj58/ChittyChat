package main

import (
	proto "ChittyChat/grpc"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"sync"

	"google.golang.org/grpc"
)

const ( // connection status
	Connect    int = 0
	Disconnect     = 1
	Publish        = 2
	Ack            = 3
)

type Server struct {
	proto.UnimplementedChittyChatServiceServer
	name             string
	address          string
	port             int
	clientReferences map[string]proto.ChittyChatService_SendMessageServer
	mutex            sync.Mutex // avoid race condition
}

// flags are used to get arguments from the terminal. Flags take a value, a default value and a description of the flag.
var port = flag.Int("port", 1000, "server port") // set with "-port <port>" in terminal

var time = 0 // Lamport variable

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
		name:    "server",
		port:    *port,
		address: GetOutboundIP().String(),
	}

	// Create a new grpc server
	grpcServer := grpc.NewServer()

	// Make the server listen at the given port (convert int port to string)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.address, strconv.Itoa(server.port)))

	if err != nil {
		log.Fatalf("Could not create the server %v", err)
	}
	log.Printf("Started server at address: %s and at port: %d\n", server.address, server.port)

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

	// when a client disconnect I terminate the relative gRPC instance
	run := true
	for run {
		msg, err := stream.Recv()
		// update local time
		if msg.Time != 0 {
			setTime(int(msg.Time))
		}
		// client reference as a String
		clientString := msg.ClientReference.ClientName + " " + msg.ClientReference.ClientAddress + ":" + strconv.Itoa(int(msg.ClientReference.ClientPort))

		if err != nil {
			log.Printf("Error while receiving message %v", err)
			break
		}

		switch int(msg.Type) {
		case Connect:
			{
				// add to map
				s.clientReferences[clientString] = stream
				log.Printf("[Lamport time: %d] Client %s has connected", time, clientString)
				/* R6: A "Participant X  joined Chitty-Chat at Lamport time L" message is broadcast
				to all Participants when client X joins, including the new Participant. */
				joinedtime := time
				for clientRef, clientStream := range s.clientReferences {
					increaseTime() // an event occurred
					msg.Time = int32(time)
					msg.Text = clientString + " has joined the chat at Lamport time " + strconv.Itoa(joinedtime)
					err = clientStream.Send(msg)
					if err != nil {
						log.Printf("Error during forwarding connection message to %s: %v", clientRef, err)
						break
					}
					log.Printf("[Lamport time: %d] Sent message to client: %s", time, clientRef)
				}
				break
			}
		case Disconnect:
			{
				log.Printf("[Lamport Time: %d] Received disconnect message from client: %s", time, clientString)
				// reply with ack
				increaseTime()
				stream.Send(&proto.Message{Type: Ack, Time: int32(time)})
				// remove stream from map
				delete(s.clientReferences, clientString)
				log.Printf("[Lamport time: %d] Client %s disconnected", time, clientString)
				/* R8: A "Participant X left Chitty-Chat at Lamport time L" message is broadcast
				to all remaining Participants when Participant X leaves. */
				disconnectedTime := time
				for clientRef, clientStream := range s.clientReferences {
					increaseTime()
					msg.Time = int32(time)
					msg.Text = clientString + " has left the chat at Lamport time " + strconv.Itoa(disconnectedTime)
					err = clientStream.Send(msg)
					if err != nil {
						log.Printf("Error during forwarding disconnection message to %s: %v", clientRef, err)
						break
					}
					log.Printf("[Lamport time: %d] Sent message to client: %s", time, clientRef)
				}
				run = false
				break
			}
		case Publish:
			{
				log.Printf("[Lamport Time: %d] Received messgae from client: %s", time, clientString)
				/* R3: Chitty-Chat service broadcast every published message, together with the current logical time */
				for clientRef, clientStream := range s.clientReferences {
					if clientRef != clientString { // Do not forward the message to the original sender
						increaseTime()         // an event occurred
						msg.Time = int32(time) // send current logial timestamp
						err = clientStream.Send(msg)
						if err != nil {
							log.Printf("Error during forwarding to %s: %v", clientRef, err)
							break
						}
						log.Printf("[Lamport time: %d] Sent message to client: %s", time, clientRef)
					}
				}
				break
			}
		}
	}
	return nil
}

func increaseTime() {
	time++
}

func setTime(received int) {
	max := math.Max(float64(received), float64(time))
	time = int(max + 1)
}

// Get preferred outbound ip of this machine
// Taken from provided tutorial: https://github.com/PatrickMatthiesen/DSYS-gRPC-template/blob/main/server/server.go
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
