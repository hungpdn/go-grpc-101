package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"sync"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type unstableServer struct {
	pb.UnimplementedUserServiceServer
	logger      *slog.Logger
	requestCout int
	mu          sync.Mutex
}

func (s *unstableServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.mu.Lock()
	s.requestCout++
	count := s.requestCout
	s.mu.Unlock()

	// Simulate a struggling server: Fails the first 4 requests
	if count <= 4 {
		s.logger.Error("server overloaded, failing request", slog.Int("req_count", count))
		return nil, status.Error(codes.Internal, "database connection lost")
	}

	// Recovers from the 5th request onwards
	s.logger.Info("server recovered, successfully processing", slog.Int("req_count", count))
	return &pb.GetUserResponse{
		UserId: req.GetUserId(),
		Name:   "Resilient User",
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50065")

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &unstableServer{logger: logger})

	logger.Info("unstable server starting on :50065")
	grpcServer.Serve(listener)
}
