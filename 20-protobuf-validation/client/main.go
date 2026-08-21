package main

import (
	"context"
	"log/slog"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	conn, _ := grpc.NewClient("localhost:50067", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	// Deliberately sending an INVALID request
	invalidReq := &pb.CreateUserRequest{
		Name:  "Bo",   // Rule: min_len = 3
		Email: "bob@", // Rule: must be email
		Age:   12,     // Rule: gte = 18
	}

	logger.Info("sending invalid request...")
	_, err := client.CreateUser(context.Background(), invalidReq)

	if err != nil {
		logger.Error("server rejected the request", slog.String("error", err.Error()))
	}
}
