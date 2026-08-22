package main

import (
	"context"
	"log/slog"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	// MAGIC IMPORT to activate xDS
	_ "google.golang.org/grpc/xds"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// instead of using "localhost:50055" or "xds:///"
	// we use "xds:///user-service" to let gRPC automatically query the Control Plane.
	target := "xds:///user-service"

	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to create xDS client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	res, err := client.GetUser(context.Background(), &pb.GetUserRequest{UserId: "usr_xds"})
	if err != nil {
		logger.Error("rpc failed", slog.String("error", err.Error()))
	} else {
		logger.Info("Success", slog.String("name", res.GetName()))
	}
}
