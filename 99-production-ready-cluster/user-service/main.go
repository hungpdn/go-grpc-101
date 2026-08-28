package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v99"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func initTelemetry(serviceName string) func() {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		log.Printf("Failed to create resource: %v", err)
	}

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "jaeger.monitoring:4317"
	}

	traceExporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		log.Printf("Warning: failed to init OTLP trace exporter (%s): %v", endpoint, err)
		return func() {}
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	return func() {
		_ = tracerProvider.Shutdown(context.Background())
	}
}

type userServer struct {
	pb.UnimplementedUserServiceServer
	hostname string
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	return &pb.GetUserResponse{
		UserId: req.GetUserId(),
		Name:   "Production User",
		Email:  "prod@example.com - Handled by: " + s.hostname,
	}, nil
}

func main() {
	hostname, _ := os.Hostname()

	// 1. Initialize OpenTelemetry Tracing
	shutdown := initTelemetry("user-service")
	defer shutdown()

	// 2. Start Prometheus Metrics Server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("User Service metrics listening on :8082")
		if err := http.ListenAndServe(":8082", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 3. Configure gRPC Server with OTel and Keepalive
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              2 * time.Hour,
		Timeout:           20 * time.Second,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // OTel Tracing
	)

	pb.RegisterUserServiceServer(grpcServer, &userServer{hostname: hostname})

	lis, err := net.Listen("tcp", ":50069")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("User Service gRPC listening on :50069 (Pod: %s)", hostname)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
