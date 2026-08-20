# 07 - Reflection & Health Check

## 🎯 Purpose
In a production microservices environment, you will face two inevitable challenges:
1. **Debugging:** How do you test an API if you don't have the `.proto` file or a compiled Go client handy?
2. **Orchestration:** How does Kubernetes or an AWS Application Load Balancer know if your gRPC server is actually ready to receive traffic?

This topic demonstrates two essential extensions:
- **Server Reflection**: Allows tools (like `grpcurl` or Postman) to discover the services and methods available on the server dynamically.
- **Health Check API**: A standardized gRPC service used to determine if a server is healthy.

## 🚀 How to Run & Test

1. **Install `grpcurl`**
   `grpcurl` is the industry-standard tool for interacting with gRPC servers from the command line.
   ```bash
   # On macOS
   brew install grpcurl

   # On Linux or via Go
   go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
   ```

2. **Start the Server**
   ```bash
   go run 07-reflection-health/server/main.go
   ```

3. **Test Server Reflection**
   List all available services:
   ```bash
   grpcurl -plaintext localhost:50057 list
   ```
   Describe a specific service:
   ```bash
   grpcurl -plaintext localhost:50057 describe user.v1.UserService
   ```

4. **Call the Health Check**
   This is the exact command a Kubernetes `exec` probe would use:
   ```bash
   grpcurl -plaintext -d '{"service": "user.v1.UserService"}' localhost:50057 grpc.health.v1.Health/Check
   ```

5. **Execute a Business RPC Call**
   ```bash
   grpcurl -plaintext -d '{"user_id": "usr_999"}' localhost:50057 user.v1.UserService/GetUser
   ```

## 📝 Notes
- Reflection is incredibly useful for debugging but should generally be disabled or strictly access-controlled in public-facing production environments.
