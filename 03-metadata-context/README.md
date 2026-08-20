# 03 - Metadata & Context

## 🎯 Purpose
This topic covers how to pass and extract contextual data during a gRPC call:
- **Metadata (Headers and Trailers)**: Similar to HTTP headers, metadata is key-value data exchanged between the client and server.
- **Context Timeout/Deadlines**: Setting a timeout on the client side so the request fails quickly if the server takes too long to respond.

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server:
   ```bash
   go run 03-metadata-context/server/main.go
   ```

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 03-metadata-context/client/main.go
   ```
   The client will send a request with custom metadata and a deadline.
   - You should see the server logging the incoming metadata.
   - If the server sleeps for longer than the context deadline, you will see a `DeadlineExceeded` error on the client side.

## 📝 Notes
- **Headers** are sent before the message data.
- **Trailers** are sent at the end of the RPC response and are commonly used by the server to pass additional status or error information.
- Always use Context Deadlines in production to prevent hanging connections or resource exhaustion.
