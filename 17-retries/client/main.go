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

	// 1. THE UNIFIED SERVICE CONFIG
	// Notice how we use the "method" field to route different policies to different RPCs.
	unifiedPolicy := `{
		"methodConfig": [
			{
				"name": [{"service": "user.v1.UserService", "method": "GetUser"}],
				"hedgingPolicy": {
					"MaxAttempts": 3,
					"HedgingDelay": "0.05s",
					"NonFatalStatusCodes": ["UNAVAILABLE", "INTERNAL", "DEADLINE_EXCEEDED"]
				}
			},
			{
				"name": [{"service": "user.v1.UserService", "method": "UpdateUser"}],
				"retryPolicy": {
					"MaxAttempts": 4,
					"InitialBackoff": "0.1s",
					"MaxBackoff": "1s",
					"BackoffMultiplier": 2,
					"RetryableStatusCodes": ["UNAVAILABLE"]
				}
			}
		]
	}`

	// 2. INJECT THE UNIFIED CONFIG
	conn, err := grpc.NewClient("localhost:50066",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(unifiedPolicy),
	)
	if err != nil {
		logger.Error("failed to create client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	// 3. EXECUTE GET USER (Will trigger Hedging if slow)
	logger.Info("--- EXECUTING GET USER (HEDGING) ---")
	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()

	start := time.Now()
	res, err := client.GetUser(ctx1, &pb.GetUserRequest{UserId: "usr_hedge_and_retry"})
	if err != nil {
		logger.Error("rpc failed", slog.String("error", err.Error()))
	} else {
		logger.Info("rpc succeeded!", slog.String("name", res.GetName()), slog.Duration("total_time", time.Since(start)))
	}

	// 4. EXECUTE UPDATE USER (Will trigger Retries if it fails)
	logger.Info("--- EXECUTING UPDATE USER (RETRIES) ---")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second) // Timeout lớn hơn để kịp Retry
	defer cancel2()

	resUpdate, err := client.UpdateUser(ctx2, &pb.UpdateUserRequest{UserId: "usr_1", NewName: "New Name"})
	if err != nil {
		logger.Error("update rpc failed", slog.String("error", err.Error()))
	} else {
		logger.Info("update rpc succeeded!", slog.String("status", resUpdate.GetStatus()))
	}
}
