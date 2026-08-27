package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v99"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type userServer struct {
	pb.UnimplementedUserServiceServer
	hostname string
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	return &pb.GetUserResponse{
		UserId: req.GetUserId(),
		Name:   "Production User",
		Email:  "prod@example.com - Handled by: " + s.hostname,
	}, nil
}

func main() {
	hostname, _ := os.Hostname()

	// 1. Start Prometheus Metrics Server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("User Service metrics listening on :8082")
		http.ListenAndServe(":8082", nil)
	}()

	// 2. Configure gRPC Server with OTel and Keepalive
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              2 * time.Hour,
		Timeout:           20 * time.Second,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // OTel Tracing
	)

	pb.RegisterUserServiceServer(grpcServer, &userServer{hostname: hostname})

	lis, err := net.Listen("tcp", ":50069")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("User Service gRPC listening on :50069 (Pod: %s)", hostname)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
