package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// For production, replace insecure.NewCredentials() with TLS credentials.
	target := "localhost:50051"
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("failed to create client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.Error("error closing connection", slog.String("error", err.Error()))
		}
	}()

	client := pb.NewUserServiceClient(conn)

	// PRODUCTION RULE: Never make a remote call without a timeout/deadline.
	// We allocate 2 seconds for this RPC to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // Always defer cancel to prevent context leaks

	req := &pb.GetUserRequest{
		UserId: "usr_123",
	}

	logger.Info("sending GetUser request", slog.String("user_id", req.UserId))

	// Execute the RPC
	res, err := client.GetUser(ctx, req)
	if err != nil {
		// Proper gRPC error parsing
		st, ok := status.FromError(err)
		if ok {
			// It's a gRPC error, we can check the exact code
			if st.Code() == codes.NotFound {
				logger.Warn("user not found", slog.String("details", st.Message()))
			} else {
				logger.Error("gRPC error occurred",
					slog.String("code", st.Code().String()),
					slog.String("message", st.Message()))
			}
		} else {
			// Non-gRPC error (e.g., connection dial failure, context timeout before dialing)
			logger.Error("system error", slog.String("error", err.Error()))
		}
		return
	}

	logger.Info("received response",
		slog.String("name", res.GetName()),
		slog.String("email", res.GetEmail()))
}
