package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	conn, _ := grpc.NewClient("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	client := pb.NewPaymentServiceClient(conn)

	// --- SCENARIO 1: WRONG TOKEN ---
	logger.Info("--- SCENARIO 1: TESTING AUTHENTICATION ---")

	// Create context with wrong metadata
	ctx1 := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer WRONG-TOKEN")

	_, err := client.ProcessPayment(ctx1, &pb.ProcessPaymentRequest{TransactionId: "tx_1", Amount: 100.0})
	if err != nil {
		logger.Error("expected auth error", slog.String("error", err.Error())) // Expected to fail with Unauthenticated
	}

	time.Sleep(1 * time.Second)

	// --- SCENARIO 2: CORRECT TOKEN BUT SHORT TIMEOUT ---
	logger.Info("--- SCENARIO 2: TESTING TIMEOUT & CANCELLATION ---")

	// 1. Create a timeout of 1 second (Server takes 3 seconds, so this WILL timeout)
	ctx2, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel() // Always defer cancel to prevent memory leaks

	// 2. Append the correct token to the context
	ctx2 = metadata.AppendToOutgoingContext(ctx2, "authorization", "Bearer super-secret-token")

	// 3. Variables to capture Headers and Trailers from the Server
	var header, trailer metadata.MD

	logger.Info("sending payment request (will timeout in 1s)...")

	_, err = client.ProcessPayment(ctx2, &pb.ProcessPaymentRequest{TransactionId: "tx_2", Amount: 500.0},
		grpc.Header(&header),   // Capture headers
		grpc.Trailer(&trailer), // Capture trailers
	)

	if err != nil {
		// Expected to fail with "DeadlineExceeded" or "Canceled"
		logger.Error("payment failed (expected timeout)", slog.String("error", err.Error()))
	} else {
		// If it succeeded, print the trailers
		logger.Info("payment success", slog.Any("server_processing_time", trailer.Get("x-processing-time")))
	}
}
