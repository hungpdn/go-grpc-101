package testing_demo

import (
	"context"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if req.GetUserId() == "usr_123" {
		return &pb.GetUserResponse{
			UserId: "usr_123",
			Name:   "Test User",
			Email:  "test@local",
		}, nil
	}

	return nil, status.Errorf(codes.NotFound, "user %s not found", req.GetUserId())
}
