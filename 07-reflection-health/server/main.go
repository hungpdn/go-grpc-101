package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	return &pb.GetUserResponse{
		UserId: req.GetUserId(),
		Name:   "Reflective Engineer",
		Email:  "reflection@production.local",
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50057")

	grpcServer := grpc.NewServer()

	// 1. REGISTER BUSINESS LOGIC
	pb.RegisterUserServiceServer(grpcServer, &userServer{})

	// 2. REGISTER HEALTH CHECK SERVICE
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Mark the overall server as SERVING
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	// Mark the specific UserService as SERVING
	healthServer.SetServingStatus("user.v1.UserService", grpc_health_v1.HealthCheckResponse_SERVING)

	// 3. REGISTER REFLECTION
	// This exposes the Protobuf schema to clients at runtime
	reflection.Register(grpcServer)

	logger.Info("server with Reflection and Health Check starting on :50057")
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
