package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
)

// initTelemetry configures OpenTelemetry to send traces to Jaeger and metrics to Prometheus
func initTelemetry() func() {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("user-service"),
	)

	// 1. Configure Tracing (Export to Jaeger via OTLP)
	traceExporter, _ := otlptracegrpc.New(context.Background(), otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint("localhost:4317"))
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	// 2. Configure Metrics (Export to Prometheus)
	metricExporter, _ := prometheus.New()
	meterProvider := metric.NewMeterProvider(metric.WithReader(metricExporter), metric.WithResource(res))
	otel.SetMeterProvider(meterProvider)

	return func() {
		tracerProvider.Shutdown(context.Background())
		meterProvider.Shutdown(context.Background())
	}
}

type userServer struct {
	pb.UnimplementedUserServiceServer
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// The Trace ID is automatically extracted from the gRPC context by otelgrpc!
	return &pb.GetUserResponse{UserId: req.GetUserId(), Name: "Observed User"}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	shutdown := initTelemetry()
	defer shutdown()

	// 3. Start a sidecar HTTP server strictly for Prometheus to scrape metrics
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Info("Prometheus metrics exposed on :8081/metrics")
		http.ListenAndServe(":8081", nil)
	}()

	// 4. Start the gRPC Server with the OpenTelemetry StatsHandler
	listener, _ := net.Listen("tcp", ":50062")
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // <--- THE MAGIC LINE
	)

	pb.RegisterUserServiceServer(grpcServer, &userServer{})

	logger.Info("gRPC server starting on :50062")
	grpcServer.Serve(listener)
}
