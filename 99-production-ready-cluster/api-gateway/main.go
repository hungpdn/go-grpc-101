package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pbOrder "github.com/hungpdn/go-grpc-101/pb/order/v99"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soheilhy/cmux"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/xds"
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

func main() {
	ctx := context.Background()

	// 1. Initialize OpenTelemetry Tracing
	shutdown := initTelemetry("api-gateway")
	defer shutdown()

	// 2. Start Prometheus Metrics Server on :8081
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("API Gateway metrics listening on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 3. Setup Circuit Breaker with 5-second recovery timeout
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "OrderServiceCB",
		MaxRequests: 100,
		Timeout:     5 * time.Second, // Fixed 5s timeout
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("CIRCUIT BREAKER STATE CHANGE: %s -> %s", from.String(), to.String())
		},
	})

	// 4. Connect to Order Service via xDS with OTel tracing & Circuit Breaker
	conn, err := grpc.NewClient(
		"xds:///order-service",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()), // Trace propagation
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			_, cbErr := cb.Execute(func() (interface{}, error) {
				return nil, invoker(ctx, method, req, reply, cc, opts...)
			})
			return cbErr
		}),
	)
	if err != nil {
		log.Fatalf("Failed to dial order service: %v", err)
	}
	defer conn.Close()

	// 5. Setup cmux on :8080 (Multiplexes gRPC HTTP/2 and REST HTTP/1.1)
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen on 8080: %v", err)
	}
	m := cmux.New(lis)
	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any())

	// 6. Start HTTP Server (gRPC-Gateway)
	go func() {
		mux := runtime.NewServeMux()
		if err := pbOrder.RegisterOrderServiceHandler(ctx, mux, conn); err != nil {
			log.Fatalf("Failed to register gateway: %v", err)
		}
		log.Println("HTTP Gateway starting...")
		if err := http.Serve(httpL, mux); err != nil {
			log.Printf("HTTP gateway closed: %v", err)
		}
	}()

	// 7. Start gRPC Server (Fallback for native gRPC clients)
	go func() {
		grpcServer := grpc.NewServer(
			grpc.StatsHandler(otelgrpc.NewServerHandler()),
		)
		log.Println("gRPC Server starting...")
		if err := grpcServer.Serve(grpcL); err != nil {
			log.Printf("gRPC server closed: %v", err)
		}
	}()

	// 8. Serve cmux
	log.Println("API Gateway cmux listening on :8080")
	if err := m.Serve(); err != nil {
		log.Fatalf("cmux server error: %v", err)
	}
}
