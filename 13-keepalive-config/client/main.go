package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// CLIENT PARAMETERS: Client actively keeps the connection alive through the Load Balancers
	kaClientParams := keepalive.ClientParameters{
		Time:                10 * time.Second, // Ping server every 10s if there is no activity
		Timeout:             3 * time.Second,  // Wait for 3s for the server to respond to the PING
		PermitWithoutStream: true,             // Ping even if the app is idle (no RPC calls)
	}

	conn, _ := grpc.NewClient("localhost:50064",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kaClientParams),
	)
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	// Send the first request to establish the network connection
	client.GetUser(context.Background(), &pb.GetUserRequest{UserId: "usr_1"})
	logger.Info("first request sent, connection established")

	// Simulate Client idle for 1 minute.
	// Thanks to keepalive.ClientParameters, gRPC will continue to send PINGs
	// every 10 seconds to keep the network connection alive!
	logger.Info("sleeping for 1 minute... keepalive PINGs are working in the background!")
	time.Sleep(1 * time.Minute)

	// Call again after 1 minute, the connection is still smooth with zero cold-start
	_, err := client.GetUser(context.Background(), &pb.GetUserRequest{UserId: "usr_1"})
	if err != nil {
		logger.Error("request failed", slog.String("error", err.Error()))
	} else {
		logger.Info("second request succeeded immediately!")
	}
}