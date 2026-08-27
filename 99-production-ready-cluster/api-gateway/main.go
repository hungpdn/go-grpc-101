package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pbOrder "github.com/hungpdn/go-grpc-101/pb/order/v99"
	"github.com/soheilhy/cmux"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/xds"
)

func main() {
	ctx := context.Background()

	// 1. Setup Circuit Breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "OrderServiceCB",
		MaxRequests: 100,
		Timeout:     5, // seconds
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})

	// 2. Connect to Order Service via xDS
	conn, err := grpc.NewClient(
		"xds:///order-service",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Intercept calls with Circuit Breaker
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			_, cbErr := cb.Execute(func() (interface{}, error) {
				return nil, invoker(ctx, method, req, reply, cc, opts...)
			})
			return cbErr
		}),
	)
	if err != nil {
		log.Fatalf("Failed to dial order service: %v", err)
	}
	defer conn.Close()

	// 3. Setup cmux on :8080
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen on 8080: %v", err)
	}
	m := cmux.New(lis)
	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any())

	// 4. Start HTTP Server (gRPC-Gateway)
	go func() {
		mux := runtime.NewServeMux()
		if err := pbOrder.RegisterOrderServiceHandler(ctx, mux, conn); err != nil {
			log.Fatalf("Failed to register gateway: %v", err)
		}
		log.Println("HTTP Gateway starting...")
		http.Serve(httpL, mux)
	}()

	// 5. Start gRPC Server (Fallback for native gRPC clients)
	go func() {
		grpcServer := grpc.NewServer()
		log.Println("gRPC Server starting...")
		grpcServer.Serve(grpcL)
	}()

	// 6. Serve cmux
	log.Println("API Gateway cmux listening on :8080")
	if err := m.Serve(); err != nil {
		log.Fatalf("cmux server error: %v", err)
	}
}
