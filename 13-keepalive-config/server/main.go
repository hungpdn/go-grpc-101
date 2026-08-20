package main

import (
	"log/slog"
	"net"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50064")

	// 1. SERVER PARAMETERS: Server auto ping Client
	kaServerParams := keepalive.ServerParameters{
		Time:    30 * time.Second, // If no activity for 30 seconds, ping Client
		Timeout: 5 * time.Second,  // Wait for 5 seconds for the client to respond to the PING, otherwise disconnect
		MaxConnectionIdle: 5 * time.Minute, // Disconnect if idle for more than 5 minutes (free up RAM)
	}

	// 2. ENFORCEMENT POLICY: Policy to protect Server from being spammed by Client (DDoS)
	kaEnforcement := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // Client cannot ping faster than 5s/time
		PermitWithoutStream: true,            // Allow Client to ping even if there are no active RPC calls
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kaServerParams),
		grpc.KeepaliveEnforcementPolicy(kaEnforcement),
	)

	pb.RegisterUserServiceServer(grpcServer, &userServer{})

	logger.Info("keepalive server starting on :50064")
	grpcServer.Serve(listener)
}