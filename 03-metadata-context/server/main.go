package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type paymentServer struct {
	pb.UnimplementedPaymentServiceServer
	logger *slog.Logger
}

func (s *paymentServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	// 1. EXTRACT METADATA (Headers)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.DataLoss, "failed to read metadata")
	}

	// Read the Authorization token
	authTokens := md.Get("authorization")
	if len(authTokens) == 0 || authTokens[0] != "Bearer super-secret-token" {
		s.logger.Warn("unauthorized request blocked")
		return nil, status.Error(codes.Unauthenticated, "invalid or missing token")
	}

	s.logger.Info("processing payment", slog.String("tx_id", req.GetTransactionId()))

	// 2. SIMULATE SLOW PROCESSING & CONTEXT CANCELLATION
	// We use a select statement to listen for either completion or context cancellation.
	processChan := make(chan bool)
	go func() {
		// Simulating a slow bank API (takes 3 seconds)
		time.Sleep(3 * time.Second)
		processChan <- true
	}()

	select {
	case <-ctx.Done():
		// The client timed out or canceled the request.
		// We must halt our work and return the context error.
		s.logger.Warn("client canceled or timed out, halting process", slog.String("err", ctx.Err().Error()))
		return nil, status.Error(codes.Canceled, "request canceled by client")

	case <-processChan:
		// Processing finished successfully before the deadline
		s.logger.Info("payment successful")
	}

	// 3. SEND TRAILERS (Metadata sent AFTER the response)
	// Useful for sending metrics like processing time back to the client
	trailer := metadata.Pairs("x-processing-time", "3000ms")
	grpc.SetTrailer(ctx, trailer)

	return &pb.ProcessPaymentResponse{
		TransactionId: req.GetTransactionId(),
		Status:        "SUCCESS",
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50053") // Using port 50053

	grpcServer := grpc.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, &paymentServer{logger: logger})

	logger.Info("metadata/context server starting on :50053")
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
