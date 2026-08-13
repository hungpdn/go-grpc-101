package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// server implementation tightly binds to the generated protobuf interface.
// We embed UnimplementedUserServiceServer for forward compatibility.
type userServer struct {
	pb.UnimplementedUserServiceServer
	logger *slog.Logger
	// In a real app, inject your business logic layer or repository here
	// e.g., userRepo UserRepository
}

// NewUserServer is a constructor ensuring proper dependency injection.
func NewUserServer(logger *slog.Logger) *userServer {
	return &userServer{
		logger: logger,
	}
}

// GetUser handles the unary RPC request.
func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// 1. Validate Input
	if req.GetUserId() == "" {
		s.logger.Warn("rejected request with empty user_id")
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 2. Respect Context Cancellation
	// Always check if the client has already canceled the request or timed out
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			s.logger.Info("client canceled request")
			return nil, status.Error(codes.Canceled, "request canceled by client")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			s.logger.Warn("request deadline exceeded")
			return nil, status.Error(codes.DeadlineExceeded, "deadline exceeded")
		}
	}

	// 3. Business Logic (Mocked)
	s.logger.Info("fetching user", slog.String("user_id", req.GetUserId()))

	// Simulate DB lookup
	if req.GetUserId() != "usr_123" {
		return nil, status.Errorf(codes.NotFound, "user %s not found", req.GetUserId())
	}

	// 4. Return standard response
	return &pb.GetUserResponse{
		UserId: req.GetUserId(),
		Name:   "Alice Engineer",
		Email:  "alice@production.local",
	}, nil
}

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := ":50051"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		logger.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize gRPC server (this is where you'd inject interceptors later)
	grpcServer := grpc.NewServer()

	// Register the service implementation
	pb.RegisterUserServiceServer(grpcServer, NewUserServer(logger))

	// Graceful Shutdown Setup
	// Channel to listen for OS signals (SIGINT, SIGTERM)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine so it doesn't block OS signal handling
	go func() {
		logger.Info("starting gRPC server", slog.String("port", port))
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error("grpc server failed", slog.String("error", err.Error()))
		}
	}()

	// Block until we receive a termination signal
	<-stopChan
	logger.Info("shutting down gracefully...")

	// GracefulStop waits for active connections to finish (with an optional hard timeout in real apps)
	grpcServer.GracefulStop()
	logger.Info("server stopped")
}
