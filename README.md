# gRPC 101 with Go

A comprehensive and practical guide to building gRPC applications in Go (Golang). 
This repository covers everything from the absolute basics (Unary RPCs) to advanced production-ready patterns, including Interceptors, Security (mTLS), gRPC Gateway, Observability, and Rate Limiting.

## 🚀 Getting Started

### Prerequisites

Before you begin, ensure you have the following tools installed:

- **Go**: Version 1.20 or higher.
- **buf**: A modern and fast tool for working with Protocol Buffers. This repository uses `buf` (instead of the traditional `protoc`) to manage, lint, and generate Go code.
  - Installation: [buf Installation Guide](https://buf.build/docs/installation)
- **Docker & Docker Compose** (Optional): Required only if you want to run the Observability example (Prometheus & Jaeger).

## 🛠️ Setup

1. **Clone the repository**:
   ```bash
   git clone <repo-url>
   cd go-grpc-101
   ```

2. **Install dependencies**:
   ```bash
   make tidy
   ```

3. **Generate Go code from `.proto` files**:
   ```bash
   make gen
   ```
   *(Note: This uses the `Makefile` to run `buf generate proto`, which reads `buf.gen.yaml` to generate the necessary Go gRPC stubs).*

## 📂 Project Structure

```
go-grpc-101/
├── README.md                  # general instructions, learning roadmap & tool installation
├── Makefile                   # command collection for generating protoc code, running, testing
├── go.mod
├── go.sum
│
├── proto/                     # Protocol Buffer definitions (*.proto files)
│   ├── user/
│   ├── order/
│   └── buf.yaml               # Buf configuration file
│
├── pb/                        # Generated Go source code from the .proto files
│   ├── user/
│   └── order/
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
│   └── server/                # gRPC server
│
├── 09-load-balancing/         # Topic 9: Client-side Load Balancing & Resolver
│   ├── client/
│   └── server/
│
├── 10-unit-testing/           # Topic 10: Testing with `bufconn` (In-memory) & Mock  
│   ├── service_test.go
│   └── servce.go
│
├── 11-observability/          # Topic 11: Prometheus Metrics & OpenTelemetry Tracing
│   ├── docker-compose.yml     # Running Prometheus, Jaeger
│   ├── client/
│   └── server/
│
├── 12-rate-limiting/          # Topic 12: Rate Limiting (Token Bucket & Interceptors)
│   ├── client/
│   └── server/
│
├── 13-keepalive-config/       # Topic 13: Keepalive & Connection Management
│   ├── client/
│   └── server/
│
└── 14-payload-compression/    # Topic 14: Payload Compression (Gzip)
    ├── client/
    └── server/
```
