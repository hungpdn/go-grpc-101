# 02 - Streaming RPC

## 🎯 Purpose
This topic demonstrates the streaming capabilities of gRPC. There are three types of streaming covered in this example:
1. **Server Streaming RPC**: The client sends a single request and the server responds with a stream of messages.
2. **Client Streaming RPC**: The client sends a stream of messages and the server responds with a single message after receiving all of them.
3. **Bidirectional Streaming RPC**: Both the client and the server send a stream of messages to each other concurrently.

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server:
   ```bash
   go run 02-streaming-rpc/server/main.go
   ```

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 02-streaming-rpc/client/main.go
   ```
   You will see the output for all three types of streaming methods executed sequentially:
   - Client receives multiple messages from the server (Server Stream).
   - Client sends multiple messages, then gets a summary response (Client Stream).
   - Both exchange messages continuously (Bidirectional Stream).

## 📝 Notes
- Streaming is highly effective for transferring large datasets, real-time data feeds, or continuous communication without the overhead of opening multiple connections.
- Ensure that you use `Send()` and `Recv()` properly, and watch out for `io.EOF` which signals the end of a stream.
