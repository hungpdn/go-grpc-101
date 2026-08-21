# 15 - Multiplexing (cmux)

## 🎯 Purpose
Sometimes your infrastructure (like certain Cloud Providers or strict firewalls) only allows you to expose a **single port** to the public internet, but you want to serve both **gRPC (HTTP/2)** and a RESTful JSON API (**HTTP/1.1** via gRPC-Gateway).

This topic demonstrates how to solve this using **Connection Multiplexing** with the `soheilhy/cmux` library.
When a request hits the single port, `cmux` inspects the first few bytes (the Byte-header). If it detects HTTP/2 gRPC traffic, it routes the connection to the gRPC server. If it detects standard HTTP/1.1 traffic, it routes it to the REST server. Both servers run concurrently on the exact same physical port!

## 🚀 How to Run & Test

1. **Start the Multiplexed Server**  
   Open a terminal and run the server. It will listen on a single port `:8080`:
   ```bash
   go run 15-multiplexing/server/main.go
   ```

2. **Test as a REST API (HTTP/1.1)**  
   Since it also serves the gRPC-Gateway on the same port, you can use `curl` to send a standard HTTP request:
   ```bash
   curl -X GET http://localhost:8080/v1/users/usr_http_123
   ```
   *(Assuming your proto has the `google.api.http` annotations set up for `/v1/users/{user_id}`)*. You should get a JSON response.

3. **Test as a gRPC API (HTTP/2)**  
   Use `grpcurl` to hit the exact same port using native gRPC:
   ```bash
   grpcurl -plaintext -d '{"user_id": "usr_grpc_999"}' localhost:8080 user.v1.UserService/GetUser
   ```
   You should get a Protobuf-decoded JSON response. Both protocols work seamlessly on port 8080!

## 📝 Notes
- `cmux` is extremely powerful but introduces a slight latency overhead since it has to buffer the initial bytes of every new connection to determine the protocol.
- Make sure to define the matching rules in the correct order! You must match `HTTP2` (gRPC) before falling back to `HTTP1Fast()`.
