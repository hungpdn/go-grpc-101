package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/hungpdn/go-grpc-101/pb/secure/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// tokenInterceptor validates the Bearer token.
// (In a real app, extract this to your middleware folder like in Topic 5)
func tokenInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 || tokens[0] != "Bearer admin-token-123" {
			logger.Warn("invalid token attempt")
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(ctx, req)
	}
}

type secureServer struct {
	pb.UnimplementedSecureServiceServer
	logger *slog.Logger
}

func (s *secureServer) GetSensitiveData(ctx context.Context, req *pb.GetSensitiveDataRequest) (*pb.GetSensitiveDataResponse, error) {
	s.logger.Info("secure request authorized", slog.String("query", req.GetQuery()))
	return &pb.GetSensitiveDataResponse{Data: "TOP SECRET: Project X"}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. Load Server's Certificate and Key (Identity)
	serverCert, err := tls.LoadX509KeyPair("06-security/certs/server-cert.pem", "06-security/certs/server-key.pem")
	if err != nil {
		logger.Error("failed to load server key pair", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. Load the CA Certificate (To verify the Client's certificate)
	caCert, err := os.ReadFile("06-security/certs/ca-cert.pem")
	if err != nil {
		logger.Error("failed to read CA cert", slog.String("error", err.Error()))
		os.Exit(1)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		logger.Error("failed to append CA cert")
		os.Exit(1)
	}

	// 3. Configure mTLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // <--- THIS ENFORCES mTLS
	}

	// 4. Create the gRPC Server with TLS and Token Interceptor
	creds := credentials.NewTLS(tlsConfig)
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(tokenInterceptor(logger)),
	)

	pb.RegisterSecureServiceServer(grpcServer, &secureServer{logger: logger})

	listener, _ := net.Listen("tcp", ":50056")

	go func() {
		logger.Info("secure mTLS server starting on :50056")
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error("server failed", slog.String("error", err.Error()))
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	grpcServer.GracefulStop()
}
