# 08 - gRPC Gateway

## 🎯 Purpose
Often, you need to provide a RESTful JSON API for legacy systems or web clients that cannot use gRPC directly. Instead of writing and maintaining a separate HTTP server, you can use **gRPC-Gateway**.
It generates a reverse-proxy server that translates standard RESTful HTTP requests into gRPC calls and vice-versa, based on annotations added to your `.proto` files (`google.api.http`).

## 🚀 How to Run & Test

1. **Start the Server & Gateway**  
   Open a terminal and run the main entry point:
   ```bash
   go run 08-grpc-gateway/server/main.go
   ```
   *Note: This typically spins up both the gRPC server and the HTTP reverse proxy concurrently on different ports, or multiplexed on the same port.*

2. **Run the gRPC Client**  
   Open another terminal and verify standard gRPC works:
   ```bash
   go run 08-grpc-gateway/client/main.go
   ```

3. **Test the RESTful JSON API**  
   Since gRPC-Gateway is running, you can now use standard `curl` to hit the API as if it were a normal REST backend.
   ```bash
   curl -X GET http://localhost:8080/v1/users/123
   ```
   (Make sure to replace `8080` with the actual HTTP port configured in your main.go).

## 📝 Notes
- The translation happens dynamically: JSON body becomes Protobuf message, HTTP method maps to RPC method.
- You must include `google/api/annotations.proto` in your imports and annotate your RPCs in the `.proto` file to define the URL paths and methods.
