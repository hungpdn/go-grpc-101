# 26 - Graceful Shutdown & Kubernetes Lifecycle

## 🎯 Purpose
During production operations (such as Canary rollouts or Kubernetes Rolling Updates), Pods are constantly terminated and replaced.

If a gRPC server terminates abruptly or handles shutdown naively:
1. **In-Flight RPC Dropping:** Active long-running queries or transactions are severed instantly, returning `UNAVAILABLE` or `connection reset by peer` to customers.
2. **K8s Asynchronous Race Condition:** Kubernetes removes the Pod IP from the `Endpoints` list asynchronously. If the server stops accepting connections before the routing tables propagate, incoming requests hit a closed port.
3. **The Infinite Hang Trap:** Standard `grpcServer.GracefulStop()` blocks until **all** active RPCs and streams finish. If a rogue client keeps an open streaming connection, the server hangs until Kubernetes forcibly kills it with `SIGKILL` (default 30s).

This topic demonstrates the **Production 4-Stage Graceful Shutdown Pattern**:
- Trap OS termination signals (`SIGINT`, `SIGTERM`).
- Immediately mark the standard **gRPC Health Check as `NOT_SERVING`** (signaling Kubernetes Readiness Probes to stop sending new traffic).
- Execute `GracefulStop()` in a background goroutine so in-flight requests finish cleanly.
- Enforce a **Timeout Guard** (Fallback to `grpcServer.Stop()`) to guarantee the server exits before Kubernetes triggers `SIGKILL`.

---

## 🚀 How to Run & Test (The Zero-Drop Experiment)

We designed this example so you can directly see graceful draining in action:

1. **Start the Server**
   In **Terminal 1**, run the server:
   ```bash
   go run 26-graceful-shutdown/server/main.go
   ```
   The server is running on `:50075`. Its `GetUser` method is deliberately programmed to take **3 full seconds** of processing.

2. **Run the Client**
   In **Terminal 2**, start the client:
   ```bash
   go run 26-graceful-shutdown/client/main.go
   ```

3. **Kill the Server IMMEDIATELY**
   As soon as you run the client, switch to **Terminal 1** within 1–2 seconds and press **`Ctrl + C`**!

### 📊 Observed Results

- **In the Server Terminal:**
  ```json
  {"level":"INFO","msg":"⏳ Received request, processing heavy work (will take 3 seconds)..."}
  ^C
  {"level":"WARN","msg":"🛑 Termination signal received! Initiating graceful shutdown...","signal":"interrupt"}
  {"level":"INFO","msg":"Step 1/2: Health check marked as NOT_SERVING. No new traffic will route here."}
  {"level":"INFO","msg":"Step 2/2: GracefulStop initiated. Draining all in-flight requests..."}
  {"level":"INFO","msg":"✅ Heavy work completed successfully!"}
  {"level":"INFO","msg":"🎉 All active RPCs completed cleanly. Server terminated with ZERO dropped requests."}
  ```

- **In the Client Terminal:**
  ```json
  {"level":"INFO","msg":"📡 Sending request to server (Server will take 3s to process)..."}
  {"level":"INFO","msg":"🎉 SUCCESS! Server completed in-flight request before shutting down.","user_name":"Graceful User","elapsed_time":"3.004s"}
  ```

Despite you killing the server midway through execution, the HTTP/2 connection remained alive just long enough to deliver the completed response back to the client!

---

## 📝 Enterprise Notes

### 1. Kubernetes `readinessProbe` Integration
By registering `grpc_health_v1` and flipping the status to `NOT_SERVING` at the very beginning of the shutdown routine:
```go
healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
```
Kubernetes will immediately fail the next readiness probe and withdraw this pod from service routing before the socket completely closes.

### 2. Distroless Image Caveat
Many online tutorials recommend adding a `lifecycle.preStop` hook in Kubernetes:
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sleep", "5"]
```
**Watch out!** If you use Google **Distroless** images (Topic 19), there is **no `/bin/sh` or `/bin/sleep`** binary in the container! Attempting to execute `sleep` will crash the container lifecycle. In Distroless architectures, connection draining and timeouts must be managed directly inside Go application code, as demonstrated in this module.
