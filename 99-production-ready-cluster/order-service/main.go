package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"buf.build/go/protovalidate"
	pbOrder "github.com/hungpdn/go-grpc-101/pb/order/v99"
	pbUser "github.com/hungpdn/go-grpc-101/pb/user/v99"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	_ "google.golang.org/grpc/xds" // Enable xDS
)

type orderServer struct {
	pbOrder.UnimplementedOrderServiceServer
	userClient pbUser.UserServiceClient
	hostname   string
	validator  protovalidate.Validator
}

func (s *orderServer) CreateOrder(ctx context.Context, req *pbOrder.CreateOrderRequest) (*pbOrder.CreateOrderResponse, error) {
	// 1. Validation (Topic 20)
	if err := s.validator.Validate(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Validation failed: %v", err)
	}

	// 2. Call User Service via xDS
	userResp, err := s.userClient.GetUser(ctx, &pbUser.GetUserRequest{UserId: req.GetUserId()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch user: %v", err)
	}

	return &pbOrder.CreateOrderResponse{
		OrderId:   "ORD-" + req.GetItemId(),
		UserName:  userResp.GetName(),
		HandledBy: s.hostname,
	}, nil
}

func main() {
	hostname, _ := os.Hostname()
	v, _ := protovalidate.New()

	// 1. Start Prometheus Metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Order Service metrics listening on :8083")
		http.ListenAndServe(":8083", nil)
	}()

	// 2. Connect to User Service via xDS
	userConn, err := grpc.NewClient(
		"xds:///user-service",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userConn.Close()
	userClient := pbUser.NewUserServiceClient(userConn)

	// 3. Start gRPC Server
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{MaxConnectionIdle: 5 * time.Minute}),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		// In a real app, add your rate limit unary interceptor here
	)

	pbOrder.RegisterOrderServiceServer(grpcServer, &orderServer{
		userClient: userClient,
		hostname:   hostname,
		validator:  v,
	})

	lis, _ := net.Listen("tcp", ":50068")
	log.Printf("Order Service gRPC listening on :50068 (Pod: %s)", hostname)
	grpcServer.Serve(lis)
}
