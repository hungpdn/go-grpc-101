package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func initTracing() func() {
	res := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName("api-gateway-client"))
	traceExporter, _ := otlptracegrpc.New(context.Background(), otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint("localhost:4317"))
	tracerProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter), trace.WithResource(res))
	otel.SetTracerProvider(tracerProvider)
	return func() { tracerProvider.Shutdown(context.Background()) }
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	defer initTracing()()

	// Create a gRPC client WITH the OpenTelemetry StatsHandler
	conn, _ := grpc.NewClient("localhost:50062",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()), // <--- INJECTS TRACE ID
	)
	defer conn.Close()
	client := pb.NewUserServiceClient(conn)

	// Simulate hitting the API 5 times
	for i := 0; i < 5; i++ {
		// Create a parent span to represent the entire request lifecycle
		tracer := otel.Tracer("client-tracer")
		ctx, span := tracer.Start(context.Background(), "HandleFrontendRequest")

		_, err := client.GetUser(ctx, &pb.GetUserRequest{UserId: "usr_obsv"})
		if err != nil {
			logger.Error("request failed", slog.String("error", err.Error()))
		}

		span.End() // Close the span
		time.Sleep(1 * time.Second)
	}

	logger.Info("finished sending requests")
}
