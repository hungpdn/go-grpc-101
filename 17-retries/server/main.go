package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type unifiedServer struct {
	pb.UnimplementedUserServiceServer
	logger        *slog.Logger
	mu            sync.Mutex
	getAttempt    int
	updateAttempt int
}

// GetUser simulates LATENCY (Slow Network) to trigger HEDGING
func (s *unifiedServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.mu.Lock()
	s.getAttempt++
	attempt := s.getAttempt
	s.mu.Unlock()

	s.logger.Info("[GET] request received", slog.Int("attempt", attempt))

	// First attempt hangs for 200ms
	if attempt == 1 {
		s.logger.Warn("[GET] simulating slow response (200ms delay)...")
		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
			s.logger.Error("[GET] request CANCELED by client (Hedging won!)")
			return nil, ctx.Err()
		}
	}

	return &pb.GetUserResponse{UserId: req.GetUserId(), Name: "Hedged User"}, nil
}

// UpdateUser simulates FAILURE (Dropped Packets) to trigger RETRIES
func (s *unifiedServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	s.mu.Lock()
	s.updateAttempt++
	attempt := s.updateAttempt
	s.mu.Unlock()

	s.logger.Info("[UPDATE] request received", slog.Int("attempt", attempt))

	// First 2 attempts return network failure
	if attempt <= 2 {
		s.logger.Error("[UPDATE] simulating network failure", slog.Int("attempt", attempt))
		return nil, status.Error(codes.Unavailable, "server temporarily unavailable")
	}

	s.logger.Info("[UPDATE] request succeeded on attempt", slog.Int("attempt", attempt))
	return &pb.UpdateUserResponse{Status: "SUCCESS"}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50066")

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &unifiedServer{logger: logger})

	logger.Info("unified server (Hedging + Retries) starting on :50066")
	grpcServer.Serve(listener)
}
