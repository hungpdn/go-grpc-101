package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/echo/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

// 1. IMPLEMENT THE CUSTOM RESOLVER
// In production, this talks to Consul/etcd/K8s to dynamically fetch IP addresses.
const demoScheme = "demo"

type demoResolverBuilder struct{}

func (*demoResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	// We hardcode the 3 backend addresses our server script is running.
	// We pass these addresses to the gRPC client connection state.
	addresses := []resolver.Address{
		{Addr: "localhost:50059"},
		{Addr: "localhost:50060"},
		{Addr: "localhost:50061"},
	}
	cc.UpdateState(resolver.State{Addresses: addresses})
	return &demoResolver{}, nil
}

func (*demoResolverBuilder) Scheme() string { return demoScheme }

type demoResolver struct{}

func (*demoResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*demoResolver) Close()                                  {}

func init() {
	// Register the custom resolver scheme globally before the app starts
	resolver.Register(&demoResolverBuilder{})
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 2. CONFIGURE SERVICE CONFIG (Load Balancing Policy)
	// We explicitly tell the client to use Round Robin.
	// If omitted, gRPC defaults to "pick_first" (which only sends traffic to Server-A).
	serviceConfig := fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, roundrobin.Name)

	// 3. DIAL WITH THE CUSTOM SCHEME
	// Notice the target URI: "demo:///backend-cluster"
	// This triggers our demoResolverBuilder instead of standard DNS.
	conn, err := grpc.NewClient(
		"demo:///backend-cluster",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(serviceConfig),
	)
	if err != nil {
		logger.Error("failed to create client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewEchoServiceClient(conn)

	// 4. FIRE 6 REQUESTS TO PROVE LOAD BALANCING
	logger.Info("firing 6 consecutive requests...")
	for i := 1; i <= 6; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		res, err := client.Ping(ctx, &pb.PingRequest{Message: fmt.Sprintf("Request %d", i)})
		if err != nil {
			logger.Error("rpc failed", slog.String("error", err.Error()))
		} else {
			logger.Info("received response",
				slog.Int("req_num", i),
				slog.String("handled_by", res.GetServerId()))
		}

		cancel()
		time.Sleep(200 * time.Millisecond) // Slight pause for readability
	}
}
