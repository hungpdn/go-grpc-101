# Stage 1: Build the static Go binary
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 is critical for Distroless static execution
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /main 99-production-ready-cluster/order-service/main.go

# Stage 2: Distroless minimal image
FROM gcr.io/distroless/static-debian12:latest

COPY --from=builder /main /main

# Standardize running the binary
ENTRYPOINT ["/main"]