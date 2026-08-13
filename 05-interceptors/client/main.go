package main

import (
	"context"
	"log/slog"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	conn, _ := grpc.NewClient("localhost:50054", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	client := pb.NewPaymentServiceClient(conn)

	// Test 1: Unauthorized
	logger.Info("--- TEST 1: NO TOKEN ---")
	_, err := client.ProcessPayment(context.Background(), &pb.ProcessPaymentRequest{TransactionId: "tx_1"})
	if err != nil {
		logger.Error("payment failed", slog.String("error", err.Error()))
	}

	// Test 2: Authorized
	logger.Info("--- TEST 2: VALID TOKEN ---")
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer super-secret-token")
	res, err := client.ProcessPayment(ctx, &pb.ProcessPaymentRequest{TransactionId: "tx_2"})
	if err != nil {
		logger.Error("payment failed", slog.String("error", err.Error()))
	} else {
		logger.Info("payment success", slog.String("status", res.GetStatus()))
	}
}
