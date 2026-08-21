# ==========================================
# STAGE 1: BUILDER
# ==========================================
# Use the official Golang image to build the binary.
# We use 'alpine' for a smaller download size during the build process.
FROM golang:1.26-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Download Go modules (Caching layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the Go app. 
# CRITICAL: CGO_ENABLED=0 ensures a static binary, which is required for Distroless!
# GOOS=linux and GOARCH=amd64 ensures it runs on standard Linux servers.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/grpc-server ./01-unary-rpc/server/main.go

# ==========================================
# STAGE 2: FINAL (DISTROLESS)
# ==========================================
# Use Google's Distroless static image as the final base.
# It contains NO shell, NO package manager, just the bare minimum to run static binaries.
FROM gcr.io/distroless/static-debian11

# Set the working directory
WORKDIR /

# Copy ONLY the compiled binary from the builder stage
COPY --from=builder /app/grpc-server /grpc-server

# Expose the gRPC port
EXPOSE 50051

# Define the non-root user (Distroless includes a default 'nonroot' user)
USER nonroot:nonroot

# Run the binary
ENTRYPOINT ["/grpc-server"]