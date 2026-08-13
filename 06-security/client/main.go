package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/secure/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// tokenAuth implements credentials.PerRPCCredentials
type tokenAuth struct {
	token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	// This automatically injects the Bearer token into EVERY request sent by this client
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (t tokenAuth) RequireTransportSecurity() bool {
	// PRINCIPAL ENGINEER INSIGHT:
	// If this returns true, gRPC will PANIC if you try to send this token over a non-TLS connection.
	// This prevents accidental token leakage over plaintext networks.
	return true
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. Load Client's Certificate and Key (Identity)
	clientCert, err := tls.LoadX509KeyPair("06-security/certs/client-cert.pem", "06-security/certs/client-key.pem")
	if err != nil {
		logger.Error("failed to load client key pair", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. Load the CA Certificate (To verify the Server's certificate)
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
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
		ServerName:   "localhost", // Must match the SAN in the server's certificate
	}
	tlsCreds := credentials.NewTLS(tlsConfig)

	// 4. Configure Per-RPC Credentials (Token)
	authCreds := tokenAuth{
		token: "admin-token-123",
	}

	// 5. Connect with both mTLS and Token Auth
	conn, err := grpc.NewClient("localhost:50056",
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithPerRPCCredentials(authCreds), // Automatically injects tokens
	)
	if err != nil {
		logger.Error("failed to connect", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewSecureServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Notice we pass a CLEAN context here! No metadata appending needed in the business layer.
	logger.Info("requesting sensitive data...")
	res, err := client.GetSensitiveData(ctx, &pb.GetSensitiveDataRequest{Query: "vault-door-code"})
	if err != nil {
		logger.Error("rpc failed", slog.String("error", err.Error()))
	} else {
		logger.Info("success", slog.String("response", res.GetData()))
	}
}
