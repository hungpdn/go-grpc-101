# 16 - Circuit Breaking

## 🎯 Purpose
When a downstream service is struggling or completely down, continuing to send requests to it will not only fail but also exhaust your own system's resources (threads, memory, network connections) while waiting for timeouts. This can lead to a domino effect where your entire architecture collapses (Cascading Failure).

This topic demonstrates how to protect your system using the **Circuit Breaker** pattern (via the `sony/gobreaker` library):
- **Closed State**: Normal operation. Requests flow through.
- **Open State**: If failures exceed a certain threshold, the circuit "trips". Subsequent requests fail *immediately* on the client side without ever hitting the network.
- **Half-Open State**: After a cooldown timeout, the breaker lets a limited number of test requests through to see if the server has recovered. If successful, it closes the circuit again; otherwise, it trips open again.

## 🚀 How to Run & Test

1. **Start the Unstable Server**  
   Open a terminal and run the server. It is intentionally programmed to fail the first 4 requests before recovering.
   ```bash
   go run 16-circuit-breaking/server/main.go
   ```

2. **Run the Client**  
   Open another terminal and run the client. The client will fire 10 requests, one per second.
   ```bash
   go run 16-circuit-breaking/client/main.go
   ```

   **Expected Behavior:**
   - **Req 1, 2, 3**: Server fails (`Internal error: database connection lost`).
   - **Circuit Trips**: Breaker changes state from `closed` to `open`.
   - **Req 4, 5, 7, 8**: Client rejects the requests *instantly* (`CIRCUIT BREAKER IS OPEN`). The network is untouched!
   - **Req 6**: The breaker enters `half-open` after the 3-second timeout and allows Req 6 to pass through as a probe. The server still fails it.
   - **Req 9, 10**: The server has finally recovered. The probe request succeeds, the breaker closes, and traffic flows normally!

## 📝 Notes
- **Crucial Rule:** Never trip a circuit breaker on business-logic errors (e.g., `InvalidArgument`, `NotFound`, `PermissionDenied`). You should only trip it on infrastructure or network failures (e.g., `Unavailable`, `DeadlineExceeded`, `Internal`).
- The interceptor implementation allows you to transparently wrap any gRPC call with circuit-breaking logic.
