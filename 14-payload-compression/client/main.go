package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 2. USE COMPRESSOR: Tell the client to compress requests and request compressed responses
	conn, err := grpc.NewClient("localhost:50051", 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)), // <--- ENABLES COMPRESSION
	)
	if err != nil {
		logger.Error("failed to connect", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	// Create a massively bloated request to simulate large payloads
	heavyPayload := strings.Repeat("A", 50000) // 50KB string
	
	logger.Info("sending compressed request...")
	
	// The gRPC library will compress this 50KB string down to a few hundred bytes before sending!
	res, err := client.GetUser(context.Background(), &pb.GetUserRequest{UserId: heavyPayload})
	if err != nil {
		logger.Error("rpc failed", slog.String("error", err.Error()))
	} else {
		logger.Info("received successful response", slog.String("user_id", res.GetUserId()[:10]+"..."))
	}
}