package main

import (
	"context"
	"fmt"
	"hash/crc32"
	"log/slog"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
)

const balancerName = "consistent_hash"

// 1. REGISTER THE LOAD BALANCER
func init() {
	// Register the custom balancer so gRPC knows about it when parsing the ServiceConfig
	balancer.Register(base.NewBalancerBuilder(
		balancerName,
		&hashPickerBuilder{},
		base.Config{HealthCheck: true},
	))
}

// 2. THE PICKER BUILDER
// This is called whenever the list of servers changes (e.g., a server dies or scales up)
type hashPickerBuilder struct{}

func (b *hashPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	// Extract all active connections into a slice
	var conns []balancer.SubConn
	for sc := range info.ReadySCs {
		conns = append(conns, sc)
	}

	return &hashPicker{
		subConns: conns,
	}
}

// 3. THE PICKER (The Algorithm)
// This is called FOR EVERY SINGLE REQUEST to decide which server gets the traffic.
type hashPicker struct {
	subConns []balancer.SubConn
}

func (p *hashPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	// Extract custom metadata from the context
	md, ok := metadata.FromOutgoingContext(info.Ctx)
	if !ok || len(md.Get("user_id")) == 0 {
		// Fallback: If no user_id is provided, just pick the first server
		return balancer.PickResult{SubConn: p.subConns[0]}, nil
	}

	userID := md.Get("user_id")[0]

	// Hash the user_id (CRC32 is fast and sufficient for basic routing)
	hash := crc32.ChecksumIEEE([]byte(userID))

	// Modulo operation to select a server index
	index := hash % uint32(len(p.subConns))

	return balancer.PickResult{SubConn: p.subConns[index]}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. SETUP MANUAL RESOLVER (Giả lập DNS trả về 3 địa chỉ IP)
	r := manual.NewBuilderWithScheme("demo-cluster")
	r.InitialState(resolver.State{
		Addresses: []resolver.Address{
			{Addr: "localhost:50071"},
			{Addr: "localhost:50072"},
			{Addr: "localhost:50073"},
		},
	})

	// Define our Service Config to use the custom balancer
	serviceConfig := fmt.Sprintf(`{"loadBalancingPolicy": "%s"}`, balancerName)

	target := "demo-cluster:///my-service"

	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(serviceConfig),
		grpc.WithResolvers(r),
	)
	if err != nil {
		logger.Error("failed to create client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	// Test 1: User A
	logger.Info("--- Routing User A ---")
	for i := 0; i < 3; i++ {
		ctx := metadata.AppendToOutgoingContext(context.Background(), "user_id", "usr_A")
		res, _ := client.GetUser(ctx, &pb.GetUserRequest{UserId: "usr_A"})
		logger.Info("User A result", slog.String("handled_by", res.GetName()))
	}

	// Test 2: User B
	logger.Info("--- Routing User B ---")
	for i := 0; i < 3; i++ {
		ctx := metadata.AppendToOutgoingContext(context.Background(), "user_id", "usr_B")
		res, _ := client.GetUser(ctx, &pb.GetUserRequest{UserId: "usr_B"})
		logger.Info("User B result", slog.String("handled_by", res.GetName()))
	}
}
