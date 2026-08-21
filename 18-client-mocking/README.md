# 18 - Client Mocking with `gomock`

## 🎯 Purpose
In Topic 10, we used `bufconn` to test a gRPC **Server**. But what about testing a service that *calls* another gRPC service?

For example: `OrderService` depends on `UserServiceClient` to look up user data before creating an order. In a unit test, you cannot (and should not) open a real network connection to a live `UserService`. You need to simulate it.

This is where **Client Mocking** with `go.uber.org/mock` (`gomock`) comes in. `mockgen` reads the gRPC client interface (generated from your `.proto`) and auto-generates a "Fake Client". You can then program exact scenarios with `EXPECT()`:
- *"Pretend UserService returns a valid user"* → test the happy path.
- *"Pretend UserService returns `NotFound`"* → test that `OrderService` handles it correctly.

This means your tests are blazing fast, fully isolated, and perfectly deterministic — no network, no real server needed.

## 🚀 How to Run & Test

1. **Install `mockgen`**
   ```bash
   go install go.uber.org/mock/mockgen@latest
   ```

2. **Generate the Mock Client**  
   Run this from the repository root to generate the mock for `UserServiceClient`:
   ```bash
   mockgen -destination=18-client-mocking/mocks/mock_user_client.go \
           -package=mocks \
           github.com/hungpdn/go-grpc-101/pb/user/v1 UserServiceClient
   ```

3. **Run the Unit Tests**  
   ```bash
   cd 18-client-mocking
   go test -v ./...
   ```

   **Expected Output:**
   ```
   --- PASS: TestCreateOrder_Success (0.00s)
   --- PASS: TestCreateOrder_UserNotFound (0.00s)
   PASS
   ```
   Both tests pass instantly with zero network connections.

## 📝 Notes
- `gomock.Any()` matches any argument. Use specific matchers (like `expectedReq`) when you want to assert that the caller passed the exact expected arguments.
- `Times(1)` enforces that the mock method is called exactly once. If it is called 0 or 2+ times, `gomock` will automatically fail the test at `ctrl.Finish()`.
- **Key insight**: Mocking tests the *caller's behavior*, not the callee's. You are not testing whether `UserService` works — you are testing whether `OrderService` handles every possible response from `UserService` correctly.
