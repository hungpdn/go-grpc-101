# 11 - Observability

## 🎯 Purpose
Running applications in production requires visibility into what the system is doing. In gRPC, this is usually achieved through interceptors that record data about every RPC. This topic covers two main pillars of observability:
- **Metrics (Prometheus)**: Collecting quantitative data like request counts, error rates, and response latency histograms.
- **Distributed Tracing (OpenTelemetry / Jaeger)**: Tracking a single request as it traverses across multiple microservices to identify bottlenecks.

## 🚀 How to Run & Test

1. **Start the Infrastructure**  
   To view metrics and traces, you need Prometheus and Jaeger running. A `docker-compose.yml` is provided.
   ```bash
   cd 11-observability
   docker-compose up -d
   ```

2. **Start the Server**  
   Open a terminal and run the server (which will expose metrics on an HTTP port and send traces to Jaeger):
   ```bash
   go run 11-observability/server/main.go
   ```

3. **Run the Client**  
   Open another terminal and run the client to generate some traffic:
   ```bash
   go run 11-observability/client/main.go
   ```

4. **View the Dashboards**  
   - Open **Prometheus**: http://localhost:9090 (Search for gRPC metrics like `grpc_server_handled_total`). Search for `rpc_server_call_duration_seconds_count` . You will see exact counters of how many requests your server has processed, split by gRPC method and status code.
   - Open **Jaeger**: http://localhost:16686 (Search for traces from your gRPC service). Select `api-gateway-client` in the Service dropdown and click "Find Traces". You will see a beautiful timeline graph showing the client's parent span (`HandleFrontendRequest`) seamlessly connected to the server's child span (`user.v1.UserService/GetUser`), proving context propagation works!
   - Open **metrics** http://localhost:8081/metrics (Expose metrics from server)

## 📝 Notes
- OpenTelemetry (`go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`) is the modern standard for distributed tracing.
- Remember to tear down the infrastructure when done: `docker-compose down`.
