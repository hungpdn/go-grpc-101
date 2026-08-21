package order

import (
	"context"
	"errors"
	"testing"

	"github.com/hungpdn/go-grpc-101/18-client-mocking/mocks"
	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"go.uber.org/mock/gomock"
)

func TestCreateOrder_Success(t *testing.T) {
	// 1. Initialize Gomock Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // Ensure all "Expectations" are called

	// 2. Create Mock Client
	mockUserClient := mocks.NewMockUserServiceClient(ctrl)

	// 3. Define Mocking Scenario
	// 3.1. Any time GetUser is called with userID "usr_999",
	// 3.2. Return this data immediately, do not throw an error (nil).
	expectedReq := &pb.GetUserRequest{UserId: "usr_999"}
	mockUserClient.EXPECT().
		GetUser(gomock.Any(), expectedReq).
		Return(&pb.GetUserResponse{Name: "Mocked Principal", Email: "mock@local"}, nil).
		Times(1) //  This function must be called exactly once

	// 4. Inject Mock Client into OrderManager
	manager := &OrderManager{userClient: mockUserClient}

	// 5. Run test
	res, err := manager.CreateOrder(context.Background(), "usr_999", 500.0)

	// 6. Check result
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedMsg := "Order successfully created for Mocked Principal (Email: mock@local)"
	if res != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, res)
	}
}

func TestCreateOrder_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserClient := mocks.NewMockUserServiceClient(ctrl)

	// 7. Define mock scenarios (MOCKING)
	// 7.1. Any time GetUser is called with userID "usr_999",
	// 7.2. Return this data immediately, do not throw an error (nil).
	mockUserClient.EXPECT().
		GetUser(gomock.Any(), gomock.Any()). // Any time GetUser is called with userID "usr_999",
		Return(nil, errors.New("rpc error: code = NotFound desc = user not found")).
		Times(1)

	manager := &OrderManager{userClient: mockUserClient}

	_, err := manager.CreateOrder(context.Background(), "usr_ghost", 500.0)

	if err == nil {
		t.Fatal("expected error, but got success")
	}
	if err.Error() != "failed to verify user: rpc error: code = NotFound desc = user not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}
