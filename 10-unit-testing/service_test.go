package testing_demo

import (
	"context"
	"net"
	"testing"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024 // 1MB buffer

// setupInMemoryServer initializes the gRPC server and returns a client connected to it.
func setupInMemoryServer(t *testing.T) (pb.UserServiceClient, func()) {
	// 1. Create the in-memory listener
	listener := bufconn.Listen(bufSize)

	// 2. Start the gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &userServer{})

	// Run the server in a goroutine so it doesn't block the test
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			// Serve returns an error when gracefully stopped, which is expected during teardown.
			// We only panic if it's a real startup error.
		}
	}()

	// 3. Create a client that dials the in-memory listener
	// grpc.WithContextDialer overrides the default TCP dialing logic.
	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	client := pb.NewUserServiceClient(conn)

	// 4. Return the client and a teardown function
	teardown := func() {
		err := conn.Close()
		if err != nil {
			t.Errorf("error closing connection: %v", err)
		}
		grpcServer.GracefulStop()
	}

	return client, teardown
}

func TestUserService_GetUser(t *testing.T) {
	// Setup the test environment
	client, teardown := setupInMemoryServer(t)
	defer teardown() // Ensure cleanup runs after tests finish

	// Define Table-Driven Tests
	tests := []struct {
		name         string
		req          *pb.GetUserRequest
		expectedCode codes.Code
		expectedName string
	}{
		{
			name:         "Success - User Found",
			req:          &pb.GetUserRequest{UserId: "usr_123"},
			expectedCode: codes.OK,
			expectedName: "Test User",
		},
		{
			name:         "Failure - User Not Found",
			req:          &pb.GetUserRequest{UserId: "usr_999"},
			expectedCode: codes.NotFound,
		},
		{
			name:         "Failure - Empty ID",
			req:          &pb.GetUserRequest{UserId: ""},
			expectedCode: codes.InvalidArgument,
		},
	}

	// Execute tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel() // You can safely uncomment this because of bufconn!

			res, err := client.GetUser(context.Background(), tc.req)

			// Check Error Codes
			if err != nil {
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("Expected gRPC status error, got: %v", err)
				}
				if st.Code() != tc.expectedCode {
					t.Errorf("Expected code %v, got %v", tc.expectedCode, st.Code())
				}
				return // Test passes if the expected error matches
			}

			// If we expected an error but got none
			if tc.expectedCode != codes.OK {
				t.Fatalf("Expected error code %v, but got success", tc.expectedCode)
			}

			// Check Success Payloads
			if res.GetName() != tc.expectedName {
				t.Errorf("Expected name %q, got %q", tc.expectedName, res.GetName())
			}
		})
	}
}
