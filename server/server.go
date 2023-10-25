package main

import (
	proto "ChittyChat/grpc"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
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
				log.Printf("Client %s connected", clientString)
				/* R6: A "Participant X  joined Chitty-Chat at Lamport time L" message is broadcast
				to all Participants when client X joins, including the new Participant. */
				for clientRef, clientStream := range s.clientReferences {
					msg.Time = int32(time)
					err = clientStream.Send(msg)
					increaseTime() // an event occurred
					if err != nil {
						log.Printf("Error during forwarding connection message to %s: %v", clientRef, err)
						break
					}
				}
				break
			}
		case Disconnect:
			{
				// reply with ack
				stream.Send(&proto.Message{Type: Ack})
				// remove stream from map
				delete(s.clientReferences, clientString)
				log.Printf("Client %s disconnected", clientString)
				/* R8: A "Participant X left Chitty-Chat at Lamport time L" message is broadcast
				to all remaining Participants when Participant X leaves. */
				for clientRef, clientStream := range s.clientReferences {
					msg.Time = int32(time)
					err = clientStream.Send(msg)
					increaseTime() // an event occurred
					if err != nil {
						log.Printf("Error during forwarding disconnection message to %s: %v", clientRef, err)
						break
					}
				}
				run = false
				break
			}
		case Publish:
			{
				/* R3: Chitty-Chat service broadcast every published message, together with the current logical time */
				for clientRef, clientStream := range s.clientReferences {
					if clientRef != clientString { // Do not forward the message to the original sender
						msg.Time = int32(time) // send current logial timestamp
						err = clientStream.Send(msg)
						increaseTime() // an event occurred
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

func increaseTime() {
	time++
}

func setTime(received int) {
	max := math.Max(float64(received), float64(time))
	time = int(max + 1)
}

// Get preferred outbound ip of this machine
// Usefull if you have to know which ip you should dial, in a client running on an other computer
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
