# gRPC 101 with Go

A practical introduction to gRPC using Go (Golang). This repository covers the basics from setting up your environment to implementing streaming and bidirectional communication.

## 🚀 Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go**: Version 1.18 or higher.
- **Protocol Buffer Compiler (`protoc`)**: Required to compile `.proto` files.
  - Installation instructions: [Protobuf Installation](https://protobuf.dev/getting-started/)
- **(Optional) buf**: A modern tool for working with Protocol Buffers. It simplifies workflows and validates schemas.
  - Installation instructions: [buf Installation](https://buf.build/docs/installation)

## 🛠️ Setup

1. **Clone the repository** (if you haven't already).

2. **Install Go dependencies**:
   ```bash
   go mod tidy
   ```

3. **Generate Go code from `.proto` files**:
   ```bash
   make generate
   ```
   *Note: This uses the `Makefile` to run `protoc` with the necessary plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, etc.).*

## 📂 Project Structure

```
go-grpc-101/
├── README.md                  # general instructions, learning roadmap & tool installation
├── Makefile                   # command collection for generating protoc code, running, testing
├── go.mod
├── go.sum
│
├── proto/                     # place to store all .proto files and generated Go code
│   ├── v1/
│   │   ├── user.proto
│   │   ├── order.proto
│   │   ├── user_grpc.pb.go
│   │   └── user.pb.go
│   └── buf.yaml               # (Optional) If using Buf CLI instead of protoc
│
├── 01-unary-rpc/              # Topic 1: basic Unary RPC
│   ├── client/
│   │   └── main.go
│   └── server/
│       └── main.go
│
├── 02-streaming-rpc/          # Topic 2: Server, Client & Bi-directional Streaming
│   ├── client/
│   │   └── main.go
│   └── server/
│       └── main.go
│
├── 03-metadata-context/       # Topic 3: Passing Metadata (Header/Trailer) & Timeout   
│   ├── client/
│   │   └── main.go
│   └── server/
│       └── main.go
│
├── 04-error-handling/         # Topic 4: gRPC Status Codes & Error Details
│   ├── client/
│   │   └── main.go
│   └── server/
│       └── main.go
│
├── 05-interceptors/           # Topic 5: Middleware (Auth, Logging, Recovery)
│   ├── middleware/            # place to write reusable Middleware functions
│   │   ├── auth.go            # Recovery (Panic handling) -> Logging/Tracing -> Authentication -> Validation
│   │   └── logging.go
│   ├── client/
│   └── server/
│
├── 06-security/               # Topic 6: TLS/mTLS & Token Authentication
│   ├── certs/                 # Contains demo SSL certificate files (.crt, .key) 
│   ├── client/
│   └── server/
│
├── 07-reflection-health/      # Topic 7: gRPC Reflection & Health Check API
│   └── server/
│       └── main.go
│
├── 08-grpc-gateway/           # Topic 8: Convert gRPC to RESTful JSON API
│   ├── proto/                 # Proto contains google.api.http annotations
│   ├── gateway/               # HTTP Reverse Proxy server
│   └── server/                # gRPC server
│
├── 09-load-balancing/         # Topic 9: Client-side Load Balancing & Resolver
│   ├── client/
│   └── server/
│
├── 10-unit-testing/           # Topic 10: Testing with `bufconn` (In-memory) & Mock  
│   ├── service_test.go
│   └── mock/
│
└── 11-observability/          # Topic 11: Prometheus Metrics & OpenTelemetry Tracing
    ├── docker-compose.yml     # Running Prometheus, Jaeger
    ├── client/
    └── server/
```

