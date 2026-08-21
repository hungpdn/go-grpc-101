package order

import (
	"context"
	"fmt"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
)

// OrderManager handles business logic for orders and depends on the User Service
type OrderManager struct {
	userClient pb.UserServiceClient
}

func (m *OrderManager) CreateOrder(ctx context.Context, userID string, amount float64) (string, error) {
	// 1. Call gRPC to another microservice to get user information
	resp, err := m.userClient.GetUser(ctx, &pb.GetUserRequest{UserId: userID})
	if err != nil {
		return "", fmt.Errorf("failed to verify user: %w", err)
	}

	// 2. Business logic (Create order...)
	if amount <= 0 {
		return "", fmt.Errorf("invalid amount")
	}

	return fmt.Sprintf("Order successfully created for %s (Email: %s)", resp.GetName(), resp.GetEmail()), nil
}
