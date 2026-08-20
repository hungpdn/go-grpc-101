# 10 - Unit Testing & Mocking

## 🎯 Purpose
Testing gRPC services efficiently is crucial. Spinning up a real server on a real TCP port for every unit test is slow and prone to port conflicts.
This topic covers how to write unit tests for gRPC using:
- **`bufconn` (Buffered Connection)**: An in-memory connection provided by the `google.golang.org/grpc/test/bufconn` package. It allows the client to talk to the server directly in memory without binding to a real network port.
- **Mocking**: Generating mock clients (e.g., using `gomock` or standard mock implementations) so you can test business logic without relying on the actual server implementation.

## 🚀 How to Run & Test

1. **Run the Tests**  
   Open a terminal and run the standard Go test command in the `10-unit-testing` directory:
   ```bash
   cd 10-unit-testing
   go test -v ./...
   ```
   You should see the tests pass successfully, executing entirely in memory.

## 📝 Notes
- `bufconn.Listen(bufSize)` creates a listener that you pass to the gRPC server.
- You can override the client's dialer in `grpc.DialContext` using `grpc.WithContextDialer` to force the client to connect via the `bufconn` listener.
- This pattern is highly recommended for CI/CD pipelines as it makes tests fast and reliable.
