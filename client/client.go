package main

import (
	"context"
	"flag"
	"fmt"
	"log"
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
)

var (
	clientPort = flag.Int("cPort", 5500, "client port")
	serverPort = flag.Int("sPort", 1000, "server port")
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
		address:    "127.0.0.1",
		portNumber: *clientPort,
	}

	// Connect and publish message
	connectAndPublish(client)
}

func connectAndPublish(client *Client) {

	// Connect to the server
	serverConnection := connectToServer()

	// Get a stream to the server
	stream, err := serverConnection.SendMessage(context.Background())
	if err != nil {
		log.Println(err)
		return
	}

	// Connect message from client to server
	connectMessage := &proto.Message{
		Text: "Connect message from client",
		Type: int32(Connect),
		ClientReference: &proto.ClientReference{
			ClientAddress: client.address, // modify in case of global test
			ClientPort:    int32(client.portNumber),
		},
	}

	if err := stream.Send(connectMessage); err != nil {
		log.Fatalf("Error while sending connection message: %v", err)
	}

	// wait for go routine
	var wg sync.WaitGroup
	wg.Add(1)

	// Create go-routine for reading messages from server
	go func() {
		me := client.address + ":" + strconv.Itoa(client.portNumber)

		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Fatalf("Error while receiving message: %v", err)
			}
			clientReference := msg.ClientReference.ClientAddress + ":" + strconv.Itoa(int(msg.ClientReference.ClientPort))
			if me == clientReference { // i receive the disconnected message that i sent
				defer wg.Done() // inform the main thread to terminate
				break
			} else {
				log.Printf("[%s] sent %s\n", clientReference, msg.Text)
			}

		}
	}()

	for {
		var text string
		log.Printf("Enter the content of the message ('exit' to quit): ")
		fmt.Scanln(&text)
		if text == "exit" {
			break
		}

		// Send message to server
		err := stream.Send(&proto.Message{
			Text: text,
			Type: int32(Publish),
			ClientReference: &proto.ClientReference{
				ClientAddress: "127.0.0.1", // modify in case of global test
				ClientPort:    int32(client.portNumber),
			},
		})
		if err != nil {
			log.Fatalf("Errore durante l'invio di un messaggio: %v", err)
		}
	}

	// Disconnect message from client
	diconnectMessage := &proto.Message{
		Text: "disconnect message",
		Type: int32(Disconnect),
		ClientReference: &proto.ClientReference{
			ClientAddress: "127.0.0.1", // modify in case of global test
			ClientPort:    int32(client.portNumber),
		},
	}
	if err := stream.Send(diconnectMessage); err != nil {
		log.Fatalf("Error while sending disconnection message: %v", err)
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

func setLog() *os.File {
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}
