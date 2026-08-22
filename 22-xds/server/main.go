package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	return &pb.GetUserResponse{UserId: req.GetUserId(), Name: "xDS Routed User"}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	lis, _ := net.Listen("tcp", ":50055")
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &userServer{})

	logger.Info("Backend server starting on :50055")
	s.Serve(lis)
}
