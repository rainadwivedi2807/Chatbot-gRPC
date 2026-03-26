package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	pb "github.com/rainadwivedi/grpc-chatbot/protocol"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main(){

	connection, err := grpc.NewClient("localhost:9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to the server: %v", err)
	}
	defer connection.Close()

	client := pb.NewChatbotServiceClient(connection)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		username = "Anonymous"
	}

	streamExample(client, username)
}

const (
	prompt = "[You] --> "
	// ANSI escape codes to clear the line more reliably
	clearLine = "\r\x1b[K"
)

func streamExample(client pb.ChatbotServiceClient, username string) {
	fmt.Printf("\nStreaming Started (User: %s) \n(Type 'exit' to quit)\n", username)
	stream, err := client.Chat(context.Background())
	if err != nil {
		log.Fatalf("Failed to start the server: %v", err)
	}
	go func() {
		for {
			response, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive message: %v", err)
			}

			msgTime := response.SentAt.AsTime().Format("15:04:05")

			if response.Type == pb.MessageType_SYSTEM {
				fmt.Printf("%s[%s] [SYSTEM] > %s\n%s", clearLine, msgTime, response.Content, prompt)
			} else {
				fmt.Printf("%s[%s] [%s] > %s\n%s", clearLine, msgTime, response.SenderName, response.Content, prompt)
			}
		}
	}()
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if text == "exit" || text == "quit" {
			fmt.Println("Exiting chat...")
			return
		}

		if err := stream.Send(&pb.Message{
			SenderName: username,
			Content:    text,
			SentAt:     timestamppb.Now(),
			Type:       pb.MessageType_TEXT,
		}); err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
		fmt.Print(prompt)
	}
}