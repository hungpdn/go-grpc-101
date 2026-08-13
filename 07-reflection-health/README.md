In a production microservices environment, you will face two inevitable challenges:

1. **Debugging:** How do you test an API if you don't have the `.proto` file or a compiled Go client handy? (Unlike REST, you can't just `curl http://...`).
2. **Orchestration:** How does Kubernetes or an AWS Application Load Balancer know if your gRPC server is actually ready to receive traffic, or if it's deadlocked?

## Install `grpcurl`

`grpcurl` is the industry-standard tool for interacting with gRPC servers from the command line.

```bash
# On macOS
brew install grpcurl

# On Linux or via Go
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

```

*(Ensure your `$GOPATH/bin` is in your PATH if you use `go install`)*

---

### Run & Query (No Go Client Needed!)

Start your server in **Terminal 1**:

```bash
go run 07-reflection-health/server/main.go

```

Now, in **Terminal 2**, let's use `grpcurl` to explore and interact with the server. Because we enabled Reflection, `grpcurl` can dynamically discover everything.

**1. List all available services**

```bash
grpcurl -plaintext localhost:50057 list

```

*Output:*

```text
grpc.health.v1.Health
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
user.v1.UserService

```

*(Notice how the Health and Reflection services expose themselves alongside your business logic!)*

**2. Describe a specific service**
Want to know what methods `UserService` has, or what the request looks like, without reading the code?

```bash
grpcurl -plaintext localhost:50057 describe user.v1.UserService

```

*Output:*

```protobuf
user.v1.UserService is a service:
service UserService {
  rpc GetUser ( .user.v1.GetUserRequest ) returns ( .user.v1.GetUserResponse );
}

```

**3. Call the Health Check**
This is the exact command a Kubernetes `exec` probe or an Envoy proxy would use to verify your service is alive.

```bash
grpcurl -plaintext -d '{"service": "user.v1.UserService"}' localhost:50057 grpc.health.v1.Health/Check

```

*Output:*

```json
{
  "status": "SERVING"
}

```

**4. Execute a Business RPC Call**
You pass the payload as a JSON string, and `grpcurl` translates it into Protobuf under the hood.

```bash
grpcurl -plaintext -d '{"user_id": "usr_999"}' localhost:50057 user.v1.UserService/GetUser

```

*Output:*

```json
{
  "userId": "usr_999",
  "name": "Reflective Engineer",
  "email": "reflection@production.local"
}

```
