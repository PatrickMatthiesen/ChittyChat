package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	gRPC "github.com/PatrickMatthiesen/ChittyChat/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Same principle as in client. Flags allows for user specific arguments/values
var clientsName = flag.String("name", "default", "Senders name")
var serverPort = flag.String("server", "5400", "Tcp server")
var lamportTime = flag.Int64("lamport", 0, "Lamport time")

var server gRPC.ChittyChatClient //the server

func main() {
	//parse flag/arguments
	flag.Parse()

	//log to file instead of console
	f := setLog()
	defer f.Close()

	//connect to server and close the connection when program closes
	connectToServer()
	go joinChat()

	// start allowing user input
	parseInputAndPublish()
}

// connect to server (you should know this by now)
func connectToServer() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	fmt.Printf("client %s: Attempts to dial on port %s\n", *clientsName, *serverPort)
	conn, err := grpc.Dial(fmt.Sprintf(":%s", *serverPort), opts...)
	if err != nil {
		fmt.Printf("Failed to Dial : %v", err)
		return
	}

	server = gRPC.NewChittyChatClient(conn)
}

func joinChat() {
	log.Println(*clientsName, ": Tries to join chat")
	stream, _ := server.Join(context.Background(), &gRPC.JoinRequest{
		Name:        *clientsName,
		LamportTime: 0,
	})

	for {
		incomeing, err := stream.Recv()
		if err != nil {
			fmt.Printf("\rThe server has shut down, hope to see you again another time!\n")
			log.Fatalf("%s: Failed to receive message from stream. \n\t%v", *clientsName, err)
		}

		if incomeing.LamportTime > *lamportTime {
			*lamportTime = incomeing.LamportTime + 1
		} else {
			*lamportTime++
		}

		log.Printf("%s | Time %d | From %s: %s", *clientsName, *lamportTime, incomeing.Sender, incomeing.Message)
		fmt.Printf("\rLamport: %v | %v: %v \n", *lamportTime, incomeing.Sender, incomeing.Message)
		fmt.Print("-> ")
	}
}

func parseInputAndPublish() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--------------------")
	fmt.Println("Type a message you want to share with the chat.")
	fmt.Println("--------------------")

	//Infinite loop to listen for client input.
	fmt.Print("-> ")
	for {
		//Read user input to the next newline
		input, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			log.Fatalf("%s had error: %v", *clientsName, err)
		}
		if err == io.EOF {
			fmt.Println("\rBye bye. Hope to see you again soon!")
			return
		}

		//Trim whitespace from input
		input = strings.TrimSpace(input)

		// we are sending a message so we increment the lamport time
		*lamportTime++

		// publish the message in the chat
		log.Printf("%s | Time %d | Attempts to publish: %s", *clientsName, *lamportTime, input)
		response, err := server.Publish(context.Background(), &gRPC.Message{
			Sender:      *clientsName,
			Message:     input,
			LamportTime: *lamportTime,
		})

		if err != nil || response == nil {
			log.Printf("Client %s: something went wrong trying to publish the message :(", *clientsName)
			continue
		}
	}
}

// sets the logger to use a log.txt file instead of the console
func setLog() *os.File {
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}
