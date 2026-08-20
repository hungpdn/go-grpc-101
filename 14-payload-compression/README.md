# 14 - Payload Compression

## 🎯 Purpose
Sending enormous datasets (e.g., thousands of records) over the internal network or the internet can exhaust bandwidth and significantly increase latency.
This topic demonstrates how to solve this by enabling **Gzip Compression** directly in the gRPC Client.

When enabled, gRPC will transparently compress the Protobuf payload on the sender side and decompress it on the receiver side. This can reduce the transmitted data size by up to 80% with minimal CPU overhead.

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server:
   ```bash
   go run 14-payload-compression/server/main.go
   ```
   *Note: In Go gRPC, as long as `google.golang.org/grpc/encoding/gzip` is imported anywhere in the server, the server will automatically understand and decompress gzip requests, and it will respond using the same compression if requested by the client.*

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 14-payload-compression/client/main.go
   ```
   - The client will generate a massively bloated 50KB string payload.
   - Because `grpc.UseCompressor(gzip.Name)` is configured, the gRPC library will compress this string down to a few hundred bytes before transmitting it over the network.
   - The server will receive it, decompress it seamlessly, and process the request.

## 📝 Notes
- You must import the compressor package (e.g., `_ "google.golang.org/grpc/encoding/gzip"`) in your server/client binaries so they can register themselves with the gRPC framework.
- Do not compress small payloads (like a few bytes). The overhead of running the compression algorithm and adding gzip headers will actually make the payload larger and slower to process. Only use it for heavy data transfers.
