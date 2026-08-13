package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/order/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type orderServer struct {
	pb.UnimplementedOrderServiceServer
	logger *slog.Logger
}

func (s *orderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	s.logger.Info("received CreateOrder request", slog.String("item_id", req.GetItemId()), slog.Int("quantity", int(req.GetQuantity())))

	// 1. VALIDATION ERROR (Bad Request)
	if req.GetQuantity() <= 0 {
		// Create a base status
		st := status.New(codes.InvalidArgument, "invalid order quantity")

		// Attach rich error details
		v := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       "quantity",
					Description: "Quantity must be greater than zero",
				},
			},
		}

		// WithDetails returns a new status with the appended details
		stWithDetails, err := st.WithDetails(v)
		if err != nil {
			// Fallback to base status if attachment fails (rare)
			return nil, st.Err()
		}
		return nil, stWithDetails.Err()
	}

	// 2. BUSINESS LOGIC ERROR (Out of Stock)
	if req.GetItemId() == "item_out_of_stock" {
		st := status.New(codes.FailedPrecondition, "item is currently out of stock")

		v := &errdetails.ErrorInfo{
			Reason: "OUT_OF_STOCK",
			Domain: "inventory.your-org.com",
			Metadata: map[string]string{
				"available_stock": "0",
				"restock_date":    "2026-09-01",
			},
		}

		stWithDetails, _ := st.WithDetails(v)
		return nil, stWithDetails.Err()
	}

	// 3. SUCCESS
	return &pb.CreateOrderResponse{
		OrderId: "ord_999",
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50054")

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, &orderServer{logger: logger})

	logger.Info("error-handling server starting on :50054")
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
