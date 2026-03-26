package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"github.com/rainadwivedi/grpc-chatbot/protocol"
)

type server struct {
	protocol.UnimplementedChatbotServiceServer
	mu      sync.Mutex
	clients map[protocol.ChatbotService_ChatServer]struct{}
}

func (s *server) SendMessage(ctx context.Context, in *protocol.Message) (*protocol.Response, error) {
	log.Printf("Received message from %s: %s", in.SenderName, in.Content)
	return &protocol.Response{
		Success:    true,
		ReceivedAt: timestamppb.Now(),
	}, nil
}

func (s *server) Chat(stream protocol.ChatbotService_ChatServer) error {
	// Register the new client
	s.mu.Lock()
	s.clients[stream] = struct{}{}
	s.mu.Unlock()

	var name string
	// Ensure the client is removed when the connection closes
	defer func() {
		s.mu.Lock()
		delete(s.clients, stream)
		s.mu.Unlock()
		if name != "" {
			s.broadcast(&protocol.Message{
				SenderName: "System",
				Content:    fmt.Sprintf("%s has left the room", name),
				SentAt:     timestamppb.Now(),
				Type:       protocol.MessageType_SYSTEM,
			}, nil)
		}
	}()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if name == "" {
			name = in.SenderName
			s.broadcast(&protocol.Message{
				SenderName: "System",
				Content:    fmt.Sprintf("%s has joined the room", name),
				SentAt:     timestamppb.Now(),
				Type:       protocol.MessageType_SYSTEM,
			}, nil)
		}

		log.Printf("Received message from %s: %s", in.SenderName, in.Content)
		s.broadcast(in, nil)
	}
}

func (s *server) broadcast(msg *protocol.Message, exclude protocol.ChatbotService_ChatServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for clientStream := range s.clients {
		if clientStream == exclude {
			continue
		}
		err := clientStream.Send(msg)
		if err != nil {
			log.Printf("Error broadcasting to a client: %v", err)
		}
	}
}

func main() {
	var port = ":9000"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to start the server: %v", err)
	}
	grpcServer := grpc.NewServer()
	s := &server{
		clients: make(map[protocol.ChatbotService_ChatServer]struct{}),
	}
	protocol.RegisterChatbotServiceServer(grpcServer, s)
	log.Printf("Server is listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start the server: %v", err)
	}
}
