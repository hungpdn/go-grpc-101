package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor logs the request details and the time taken to execute it.
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// 1. PRE-PROCESSING: Log the incoming request
		logger.Info("gRPC call started", slog.String("method", info.FullMethod))

		// 2. EXECUTE THE HANDLER (or next interceptor in the chain)
		res, err := handler(ctx, req)

		// 3. POST-PROCESSING: Log the result and latency
		latency := time.Since(start)
		if err != nil {
			logger.Error("gRPC call failed", slog.String("method", info.FullMethod), slog.String("error", err.Error()), slog.Duration("latency", latency))
		} else {
			logger.Info("gRPC call succeeded", slog.String("method", info.FullMethod), slog.Duration("latency", latency))
		}

		return res, err
	}
}

// AuthInterceptor checks for a valid Bearer token in the metadata.
func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract Metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.DataLoss, "failed to read metadata")
		}

		// Validate Token
		authTokens := md.Get("authorization")
		if len(authTokens) == 0 || authTokens[0] != "Bearer super-secret-token" {
			return nil, status.Error(codes.Unauthenticated, "invalid or missing token")
		}

		// Token is valid, proceed to the handler
		return handler(ctx, req)
	}
}
