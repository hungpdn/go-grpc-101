package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"

	// Import this line at server to register the Gzip decompressor with the grpc.Server
	_ "google.golang.org/grpc/encoding/gzip"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
	logger *slog.Logger
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// Log the payload length to prove that the Server has successfully decompressed the 50,000 'A' string
	s.logger.Info("received request", slog.Int("payload_length", len(req.GetUserId())))

	return &pb.GetUserResponse{
		UserId: req.GetUserId()[:10] + "... (truncated)",
		Name:   "Compressed User",
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50051")

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &userServer{logger: logger})

	logger.Info("server with GZIP decompression enabled starting on :50051")
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
