package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
	logger *slog.Logger
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.logger.Info("request received", slog.String("user_id", req.GetUserId()))
	return &pb.GetUserResponse{UserId: req.GetUserId(), Name: "Multiplexed User"}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. CREATE A SINGLE MAIN TCP LISTENER
	mainListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		logger.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. CREATE THE MULTIPLEXER
	m := cmux.New(mainListener)

	// 3. MATCHING RULES (Order matters!)
	// First, match connections that look like HTTP/2 (gRPC natively uses HTTP/2)
	grpcListener := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))

	// Then, match connections that look like HTTP/1.1 (Standard REST/JSON)
	httpListener := m.Match(cmux.HTTP1Fast())

	// 4. SETUP gRPC SERVER
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &userServer{logger: logger})

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	// 5. SETUP HTTP SERVER (gRPC-Gateway)
	// We dial our OWN in-memory grpcListener instead of a physical network port!
	conn, err := grpc.NewClient("passthrough://localhost",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return net.Dial("tcp", "localhost:8080")
		}),
	)
	if err != nil {
		logger.Error("failed to dial gateway", slog.String("error", err.Error()))
		os.Exit(1)
	}

	gwmux := runtime.NewServeMux()
	pb.RegisterUserServiceHandler(context.Background(), gwmux, conn)
	httpServer := &http.Server{Handler: gwmux}

	// 6. START SERVERS IN GOROUTINES
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("grpc server crashed", slog.String("error", err.Error()))
		}
	}()

	go func() {
		if err := httpServer.Serve(httpListener); err != nil {
			logger.Error("http server crashed", slog.String("error", err.Error()))
		}
	}()

	// 7. START THE MULTIPLEXER (Blocks the main thread)
	logger.Info("multiplexing server starting on a SINGLE PORT :8080 (gRPC + REST)")
	if err := m.Serve(); err != nil {
		logger.Error("cmux server failed", slog.String("error", err.Error()))
	}
}
