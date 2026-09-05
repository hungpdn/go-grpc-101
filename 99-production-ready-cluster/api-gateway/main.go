package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize OpenTelemetry Tracing
	shutdownTelemetry := initTelemetry("api-gateway")
	defer shutdownTelemetry()

	// 2. Start Prometheus Metrics Server on :8081
	metricsServer := &http.Server{Addr: ":8081"}
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("API Gateway metrics listening on :8081")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 3. Setup Circuit Breaker with 5-second recovery timeout
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "OrderServiceCB",
		MaxRequests: 100,
		Timeout:     5 * time.Second,
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
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
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
	mux := runtime.NewServeMux()
	if err := pbOrder.RegisterOrderServiceHandler(ctx, mux, conn); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	mux.HandlePath("GET", "/ping", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "pong")
	})
	httpServer := &http.Server{Handler: mux}

	go func() {
		log.Println("HTTP Gateway starting...")
		if err := httpServer.Serve(httpL); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP gateway closed: %v", err)
		}
	}()

	// 7. Start gRPC Server (Fallback for native gRPC clients)
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	go func() {
		log.Println("gRPC Server starting...")
		if err := grpcServer.Serve(grpcL); err != nil && err != grpc.ErrServerStopped {
			log.Printf("gRPC server closed: %v", err)
		}
	}()

	// 8. Serve cmux in background
	go func() {
		log.Println("API Gateway cmux listening on :8080")
		if err := m.Serve(); err != nil && err != cmux.ErrListenerClosed && err != net.ErrClosed {
			log.Printf("cmux server error: %v", err)
		}
	}()

	// 9. Handle Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("API Gateway shutting down gracefully...")

	// Drain HTTP and Metrics servers
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelShutdown()

	_ = httpServer.Shutdown(ctxShutdown)
	_ = metricsServer.Shutdown(ctxShutdown)

	// Close cmux listener and graceful stop grpc
	m.Close()
	grpcServer.GracefulStop()

	log.Println("API Gateway stopped cleanly.")
}
