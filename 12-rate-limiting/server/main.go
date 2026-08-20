package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimiterInterceptor restricts incoming requests using a Token Bucket algorithm
func RateLimiterInterceptor(limiter *rate.Limiter, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Try to consume 1 token. If the bucket is empty, reject immediately.
		if !limiter.Allow() {
			logger.Warn("rate limit exceeded, rejecting request", slog.String("method", info.FullMethod))
			// HTTP equivalent of 429 Too Many Requests
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded: please try again later")
		}

		return handler(ctx, req)
	}
}

type userServer struct {
	pb.UnimplementedUserServiceServer
	logger *slog.Logger
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.logger.Info("processing user request", slog.String("user_id", req.GetUserId()))
	return &pb.GetUserResponse{UserId: req.GetUserId(), Name: "Rate Limited User"}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50063")

	// Create a limiter: 2 requests per second, maximum burst size of 2
	limiter := rate.NewLimiter(2, 2)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(RateLimiterInterceptor(limiter, logger)),
	)
	pb.RegisterUserServiceServer(grpcServer, &userServer{logger: logger})

	logger.Info("rate-limited server starting on :50063")
	grpcServer.Serve(listener)
}
