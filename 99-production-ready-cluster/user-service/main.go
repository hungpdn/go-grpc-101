package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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
	metricsServer := &http.Server{Addr: ":8082"}
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("User Service metrics listening on :8082")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 3. Configure gRPC Server with OTel, Keepalive, and Health Check
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              2 * time.Hour,
		Timeout:           20 * time.Second,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	pb.RegisterUserServiceServer(grpcServer, &userServer{hostname: hostname})

	lis, err := net.Listen("tcp", ":50069")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// 4. Run gRPC Server in background
	go func() {
		log.Printf("User Service gRPC listening on :50069 (Pod: %s)", hostname)
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 5. Graceful Shutdown with Signal Trapping & Timeout Guard
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("User Service shutting down gracefully...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Close metrics server
	ctxShutdown, cancelMetrics := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelMetrics()
	_ = metricsServer.Shutdown(ctxShutdown)

	// Drain active gRPC connections
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("User Service gRPC connections drained cleanly.")
	case <-time.After(10 * time.Second):
		log.Println("User Service drain timeout exceeded, forcing stop...")
		grpcServer.Stop()
	}
}
