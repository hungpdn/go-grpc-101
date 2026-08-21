package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// CircuitBreakerInterceptor wraps the gRPC call inside a Sony gobreaker
func CircuitBreakerInterceptor(cb *gobreaker.CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
	) error {
		// Execute the RPC call through the breaker
		_, err := cb.Execute(func() (interface{}, error) {
			err := invoker(ctx, method, req, reply, cc, opts...)
			return nil, err
		})
		return err
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. CONFIGURE THE CIRCUIT BREAKER
	cbSettings := gobreaker.Settings{
		Name:        "UserServiceClient",
		MaxRequests: 1,               // How many requests are allowed to pass through when Half-Open
		Timeout:     3 * time.Second, // Keep the breaker Open for 3 seconds before attempting recovery (Half-Open)
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// TRIP CONDITION: Open the breaker after 3 consecutive failures
			return counts.ConsecutiveFailures >= 3
		},
		IsSuccessful: func(err error) bool {
			// PRINCIPAL INSIGHT: Do not trip the breaker on business errors!
			if err == nil {
				return true
			}
			st, _ := status.FromError(err)
			// Only consider these infrastructure codes as actual network failures
			if st.Code() == codes.Internal || st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
				return false
			}
			return true
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("CIRCUIT BREAKER STATE CHANGED", slog.String("from", from.String()), slog.String("to", to.String()))
		},
	}

	cb := gobreaker.NewCircuitBreaker(cbSettings)

	// 2. CONNECT WITH THE INTERCEPTOR
	conn, _ := grpc.NewClient("localhost:50065",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(CircuitBreakerInterceptor(cb)),
	)
	defer conn.Close()
	client := pb.NewUserServiceClient(conn)

	// 3. FIRE 10 REQUESTS (1 request every second)
	for i := 1; i <= 10; i++ {
		_, err := client.GetUser(context.Background(), &pb.GetUserRequest{UserId: "usr_1"})

		if err != nil {
			// If the error is gobreaker.ErrOpenState, it means the request NEVER hit the network!
			if err == gobreaker.ErrOpenState {
				logger.Error("Client side rejected", slog.Int("req", i), slog.String("reason", "CIRCUIT BREAKER IS OPEN"))
			} else {
				logger.Error("Server side failed", slog.Int("req", i), slog.String("error", err.Error()))
			}
		} else {
			logger.Info("Success", slog.Int("req", i))
		}
		time.Sleep(1 * time.Second)
	}
}
