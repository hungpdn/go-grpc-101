package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type slowUserServer struct {
	pb.UnimplementedUserServiceServer
	logger *slog.Logger
}

// GetUser deliberately takes 3 seconds to simulate a long-running database or I/O operation
func (s *slowUserServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.logger.Info("⏳ Received request, processing heavy work (will take 3 seconds)...", slog.String("user_id", req.GetUserId()))

	select {
	case <-time.After(3 * time.Second):
		s.logger.Info("✅ Heavy work completed successfully!", slog.String("user_id", req.GetUserId()))
		return &pb.GetUserResponse{
			UserId: req.GetUserId(),
			Name:   "Graceful User",
			Email:  "graceful@production.local",
		}, nil
	case <-ctx.Done():
		s.logger.Warn("❌ Client cancelled context before work finished!", slog.String("reason", ctx.Err().Error()))
		return nil, ctx.Err()
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	lis, err := net.Listen("tcp", ":50075")
	if err != nil {
		logger.Error("Failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	// 1. Register Health Check API
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// 2. Register Business Service
	pb.RegisterUserServiceServer(grpcServer, &slowUserServer{logger: logger})

	// 3. Start gRPC Server in a background goroutine so main thread can listen for signals
	go func() {
		logger.Info("🚀 Server started on :50075. Waiting for requests...")
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			logger.Error("Server error", slog.String("error", err.Error()))
		}
	}()

	// 4. Trap OS termination signals (SIGINT - Ctrl+C, SIGTERM - Kubernetes Kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Warn("🛑 Termination signal received! Initiating graceful shutdown...", slog.String("signal", sig.String()))

	// Step A: Immediately flip Health Check to NOT_SERVING.
	// In Kubernetes, this causes the Readiness Probe to fail, removing this pod from traffic routing.
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	logger.Info("Step 1/2: Health check marked as NOT_SERVING. No new traffic will route here.")

	// Step B: Start GracefulStop in background with a Timeout Guard
	stopped := make(chan struct{})
	go func() {
		logger.Info("Step 2/2: GracefulStop initiated. Draining all in-flight requests...")
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Timeout Guard: Wait at most 8 seconds for active RPCs to finish.
	// If an RPC hangs or streams indefinitely, force stop before OS sends SIGKILL.
	const drainTimeout = 8 * time.Second
	select {
	case <-stopped:
		logger.Info("🎉 All active RPCs completed cleanly. Server terminated with ZERO dropped requests.")
	case <-time.After(drainTimeout):
		logger.Error("⚠️ Drain timeout exceeded! Forcing ungraceful termination...", slog.Duration("timeout", drainTimeout))
		grpcServer.Stop() // Force close all pending connections
	}
}
