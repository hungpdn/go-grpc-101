package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"sync"

	pb "github.com/hungpdn/go-grpc-101/pb/echo/v1"
	"google.golang.org/grpc"
)

type echoServer struct {
	pb.UnimplementedEchoServiceServer
	serverID string
}

func (s *echoServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	// Return the server ID so the client knows who processed this request
	return &pb.PingResponse{
		ServerId: s.serverID,
	}, nil
}

// startServer is a helper to spin up a gRPC server on a specific port
func startServer(port string, serverID string, wg *sync.WaitGroup, logger *slog.Logger) {
	defer wg.Done()

	listener, err := net.Listen("tcp", port)
	if err != nil {
		logger.Error("failed to listen", slog.String("port", port))
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterEchoServiceServer(grpcServer, &echoServer{serverID: serverID})

	logger.Info("backend server started", slog.String("server_id", serverID), slog.String("port", port))
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	var wg sync.WaitGroup

	// Simulate a cluster of 3 backend instances
	instances := map[string]string{
		":50059": "Server-A",
		":50060": "Server-B",
		":50061": "Server-C",
	}

	for port, id := range instances {
		wg.Add(1)
		go startServer(port, id, &wg, logger)
	}

	wg.Wait() // Block forever
}
