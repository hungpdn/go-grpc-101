# 05 - Interceptors

## 🎯 Purpose
This topic covers **gRPC Interceptors**, which act like middleware in REST APIs. Interceptors allow you to hook into the RPC lifecycle to execute common logic before or after a request is handled. Examples include:
- **Logging**: Logging every incoming request and outgoing response.
- **Authentication**: Validating tokens (like JWTs) before reaching the handler.
- **Recovery**: Catching panics to prevent the server from crashing.

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server:
   ```bash
   go run 05-interceptors/server/main.go
   ```

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 05-interceptors/client/main.go
   ```
   - On the server terminal, you will notice logs being printed out automatically by the interceptors for each request.
   - If the client fails authentication (e.g., sending an invalid token), the interceptor will reject the request before it even reaches the core business logic.

## 📝 Notes
- **Unary Interceptors** (`grpc.UnaryInterceptor`) intercept single request/response RPCs.
- **Stream Interceptors** (`grpc.StreamInterceptor`) intercept streaming RPCs.
- To use multiple interceptors at once, use `grpc-middleware` (e.g., `grpc.ChainUnaryInterceptor`).
