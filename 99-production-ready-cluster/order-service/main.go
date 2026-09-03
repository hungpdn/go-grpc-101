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

	"buf.build/go/protovalidate"
	pbOrder "github.com/hungpdn/go-grpc-101/pb/order/v99"
	pbUser "github.com/hungpdn/go-grpc-101/pb/user/v99"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
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

func RateLimiterInterceptor(limiter *rate.Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow() {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded on %s", info.FullMethod)
		}
		return handler(ctx, req)
	}
}

type orderServer struct {
	pbOrder.UnimplementedOrderServiceServer
	userClient pbUser.UserServiceClient
	hostname   string
	validator  protovalidate.Validator
}

func (s *orderServer) CreateOrder(ctx context.Context, req *pbOrder.CreateOrderRequest) (*pbOrder.CreateOrderResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Validation failed: %v", err)
	}

	userResp, err := s.userClient.GetUser(ctx, &pbUser.GetUserRequest{UserId: req.GetUserId()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch user: %v", err)
	}

	return &pbOrder.CreateOrderResponse{
		OrderId:   "ORD-" + req.GetItemId(),
		UserName:  userResp.GetName(),
		HandledBy: s.hostname,
	}, nil
}

func main() {
	hostname, _ := os.Hostname()
	v, _ := protovalidate.New()

	// 1. Initialize OpenTelemetry Tracing
	shutdown := initTelemetry("order-service")
	defer shutdown()

	// 2. Start Prometheus Metrics
	metricsServer := &http.Server{Addr: ":8083"}
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("Order Service metrics listening on :8083")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// 3. Connect to User Service via xDS
	userConn, err := grpc.NewClient(
		"xds:///user-service",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	defer userConn.Close()
	userClient := pbUser.NewUserServiceClient(userConn)

	// 4. Rate Limiter
	limiter := rate.NewLimiter(rate.Limit(500), 1000)

	// 5. Start gRPC Server with Health Check & Interceptors
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              2 * time.Hour,
			Timeout:           20 * time.Second,
		}),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(RateLimiterInterceptor(limiter)),
	)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	pbOrder.RegisterOrderServiceServer(grpcServer, &orderServer{
		userClient: userClient,
		hostname:   hostname,
		validator:  v,
	})

	lis, err := net.Listen("tcp", ":50068")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		log.Printf("Order Service gRPC listening on :50068 (Pod: %s)", hostname)
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Order Service shutting down gracefully...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	ctxShutdown, cancelMetrics := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelMetrics()
	_ = metricsServer.Shutdown(ctxShutdown)

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("Order Service gRPC connections drained cleanly.")
	case <-time.After(10 * time.Second):
		log.Println("Order Service drain timeout exceeded, forcing stop...")
		grpcServer.Stop()
	}
}
