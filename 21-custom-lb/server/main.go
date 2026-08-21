package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedUserServiceServer
	serverID string
}

func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// Return the serverID in the Name field so we can see which server handled it
	return &pb.GetUserResponse{
		UserId: req.GetUserId(),
		Name:   fmt.Sprintf("Handled by Server [%s]", s.serverID),
	}, nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50071"
	}
	serverID := ":" + port

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, err := net.Listen("tcp", serverID)
	if err != nil {
		logger.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &server{serverID: serverID})

	logger.Info("server instance starting", slog.String("port", port))
	grpcServer.Serve(listener)
}
