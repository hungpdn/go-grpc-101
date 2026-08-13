package main

import (
	"context"
	"log/slog"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/order/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	conn, _ := grpc.NewClient("localhost:50054", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()

	client := pb.NewOrderServiceClient(conn)

	// We will run two tests to catch both error types
	testCases := []struct {
		name string
		req  *pb.CreateOrderRequest
	}{
		{"Validation Test", &pb.CreateOrderRequest{ItemId: "item_1", Quantity: -5}},
		{"Business Logic Test", &pb.CreateOrderRequest{ItemId: "item_out_of_stock", Quantity: 10}},
	}

	for _, tc := range testCases {
		logger.Info("--- RUNNING TEST ---", slog.String("test", tc.name))

		_, err := client.CreateOrder(context.Background(), tc.req)
		if err != nil {
			// 1. Convert standard Go error to gRPC Status
			st, ok := status.FromError(err)
			if !ok {
				logger.Error("non-gRPC error occurred", slog.String("error", err.Error()))
				continue
			}

			// 2. Log the standard Code and Message
			logger.Error("RPC failed",
				slog.String("code", st.Code().String()),
				slog.String("message", st.Message()))

			// 3. Unpack and type-switch the Rich Error Details
			for _, detail := range st.Details() {
				switch t := detail.(type) {

				case *errdetails.BadRequest:
					logger.Warn("caught BadRequest details")
					for _, violation := range t.GetFieldViolations() {
						logger.Warn("field violation",
							slog.String("field", violation.GetField()),
							slog.String("description", violation.GetDescription()))
					}

				case *errdetails.ErrorInfo:
					logger.Warn("caught ErrorInfo details",
						slog.String("reason", t.GetReason()),
						slog.String("domain", t.GetDomain()))

					// Read dynamic metadata (e.g., restock date)
					for k, v := range t.GetMetadata() {
						logger.Warn("error metadata", slog.String(k, v))
					}
				}
			}
		}
	}
}
