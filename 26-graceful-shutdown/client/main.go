package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	conn, err := grpc.NewClient("localhost:50075", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to dial server", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	logger.Info("📡 Sending request to server (Server will take 3s to process)...")
	logger.Info("👉 EXPERIMENT: Go to the SERVER terminal and press [Ctrl + C] RIGHT NOW!")

	start := time.Now()
	// Set 10s context deadline to allow plenty of time for processing + graceful drain
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetUser(ctx, &pb.GetUserRequest{UserId: "usr_graceful_test"})
	if err != nil {
		logger.Error("❌ RPC FAILED! Request was dropped during shutdown.", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("🎉 SUCCESS! Server completed in-flight request before shutting down.",
		slog.String("user_name", resp.GetName()),
		slog.Duration("elapsed_time", time.Since(start)),
	)
}
