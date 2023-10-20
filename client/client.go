package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"sync"

	proto "ChittyChat/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	name       string
	address    string
	portNumber int
}

const ( // connection status
	Connect    int = 0
	Disconnect     = 1
	Publish        = 2
	Ack            = 3
)

var (
	clientPort = flag.Int("cPort", 5500, "client port")
	serverPort = flag.Int("sPort", 1000, "server port")
	time       = 0 // Lamport variable
)

func main() {
	// Parse the flags to get the port for the client
	flag.Parse()

	// sets the logger to use a log.txt file instead of the console
	//f := setLog()
	//defer f.Close()

	// Create a client
	client := &Client{
		name:       "client",
		address:    "127.0.0.1", // modify in case of global test
		portNumber: *clientPort,
	}

	// Connect and publish message
	connectAndPublish(client)
}

func connectAndPublish(client *Client) {

	// Connect to the server
	serverConnection := connectToServer()

	client_reference := &proto.ClientReference{
		ClientAddress: client.address,
		ClientPort:    int32(client.portNumber),
	}

	// Get a stream to the server
	stream, err := serverConnection.SendMessage(context.Background())
	increaseTime() // an event occurred
	if err != nil {
		log.Println(err)
		return
	}

	// Connect message from client to server
	connectMessage := &proto.Message{
		Text:            "connect message",
		Type:            int32(Connect),
		ClientReference: client_reference,
	}

	if err := stream.Send(connectMessage); err != nil {
		log.Fatalf("Error while sending connection message: %v", err)
	}
	increaseTime() // an event occurred

	// wait for go routine
	var wg sync.WaitGroup
	wg.Add(1)

	// Create go-routine for reading messages from server
	go func() {
		for {
			msg, err := stream.Recv()
			// update local time
			if msg.Time != 0 {
				setTime(int(msg.Time))
			}
			if err != nil {
				log.Fatalf("Error while receiving message: %v", err)
			}

			if msg.Type == Ack { // I receive the ack for the disconnected message that i sent
				defer wg.Done() // inform the main thread to terminate
				break
			} else {
				// R4: When a client receives a broadcasted message, it has to write the message and the current logical timestamp
				senderReference := msg.ClientReference.ClientAddress + ":" + strconv.Itoa(int(msg.ClientReference.ClientPort))
				log.Printf("[%s] sent %s serverTime %d  localTime %d \n", senderReference, msg.Text, msg.Time, time)
				log.Printf("Enter the content of the message ('exit' to quit): ") // just for better user experience
			}

		}
	}()

	for {
		var text string
		log.Printf("Enter the content of the message ('exit' to quit): ")
		fmt.Scanln(&text)
		// R7: Chat clients can drop out at any time
		if text == "exit" {
			// Disconnect message from client to server
			diconnectMessage := &proto.Message{
				Text:            "disconnect message",
				Type:            int32(Disconnect),
				ClientReference: client_reference,
			}
			if err := stream.Send(diconnectMessage); err != nil {
				log.Fatalf("Error while sending disconnection message: %v", err)
			}
			increaseTime() // an event occurred
			break
		}

		// R2: Clients in Chitty-Chat can Publish a valid chat message at any time they wish
		err := stream.Send(&proto.Message{
			Text:            text,
			Type:            int32(Publish),
			ClientReference: client_reference,
		})
		if err != nil {
			log.Fatalf("Error while sending message: %v", err)
		}
		increaseTime() // an event occurred
	}

	// wait for response to disconnect message
	wg.Wait()
}

func connectToServer() proto.ChittyChatServiceClient {
	// Dial the server at the specified port.
	conn, err := grpc.Dial("127.0.0.1:"+strconv.Itoa(*serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to port %d", *serverPort)
	} else {
		log.Printf("Connected to the server at port %d\n", *serverPort)
	}
	return proto.NewChittyChatServiceClient(conn)
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
