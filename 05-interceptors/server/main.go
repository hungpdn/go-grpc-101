package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/hungpdn/go-grpc-101/05-interceptors/middleware"
	pb "github.com/hungpdn/go-grpc-101/pb/payment/v1"
	"google.golang.org/grpc"
)

type paymentServer struct {
	pb.UnimplementedPaymentServiceServer
	logger *slog.Logger
}

func (s *paymentServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	// BEAUTIFUL: No auth logic here! Just core business rules.
	s.logger.Info("processing payment...", slog.String("tx_id", req.GetTransactionId()))

	return &pb.ProcessPaymentResponse{
		TransactionId: req.GetTransactionId(),
		Status:        "SUCCESS",
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50055")

	// Inject the Interceptor Chain
	// Order matters: Logging runs first (to catch auth rejections), Auth runs second.
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor(logger),
			middleware.AuthInterceptor(),
		),
	)

	pb.RegisterPaymentServiceServer(grpcServer, &paymentServer{logger: logger})

	logger.Info("server with interceptors starting on :50055")
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
