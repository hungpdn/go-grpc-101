# 01 - Unary RPC

## 🎯 Purpose
This topic demonstrates the most basic type of gRPC communication: **Unary RPC**.
In a Unary RPC, the client sends a single request to the server and receives a single response back, much like a normal function call or a standard HTTP REST API request.

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server:
   ```bash
   go run 01-unary-rpc/server/main.go
   ```
   The server will start listening on port `50051`.

2. **Run the Client**  
   Open another terminal and run the client to send a request:
   ```bash
   go run 01-unary-rpc/client/main.go
   ```
   You should see the client sending a request (e.g., a User ID) and printing out the response received from the server.

## 📝 Notes
- Ensure you have run `make generate` from the root directory before running this example to generate all necessary Go files from the Protocol Buffers.
- Unary RPC is the default and most common choice when you don't need to stream data.
