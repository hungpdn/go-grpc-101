package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"buf.build/go/protovalidate"
	pb "github.com/hungpdn/go-grpc-101/pb/user/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// ValidationInterceptor automatically validates any incoming protobuf message
func ValidationInterceptor(validator protovalidate.Validator, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		// 1. Cast the request to a proto.Message
		if msg, ok := req.(proto.Message); ok {
			// 2. Validate against the rules defined in the .proto file
			if err := validator.Validate(msg); err != nil {
				logger.Warn("request rejected by validation interceptor", slog.String("method", info.FullMethod), slog.String("error", err.Error()))

				// Return a strict 400 Invalid Argument error to the client
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}

		// 3. If valid, pass it to the business logic
		return handler(ctx, req)
	}
}

type userServer struct {
	pb.UnimplementedUserServiceServer
	logger *slog.Logger
}

func (s *userServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	// Look how clean the business logic is!
	// No 'if req.Email == ""' boilerplate. We guarantee the data is safe to use.
	s.logger.Info("saving valid user to database...", slog.String("email", req.GetEmail()))
	return &pb.CreateUserResponse{Message: "User created successfully!"}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50067")

	// Initialize the high-performance validator
	validator, err := protovalidate.New()
	if err != nil {
		logger.Error("failed to initialize validator", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Register the Interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(ValidationInterceptor(validator, logger)),
	)
	pb.RegisterUserServiceServer(grpcServer, &userServer{logger: logger})

	logger.Info("server with validation interceptor starting on :50067")
	grpcServer.Serve(listener)
}
