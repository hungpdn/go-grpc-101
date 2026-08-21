# 17 - Retries & Hedging

## 🎯 Purpose
Networks are inherently unreliable. Packet drops or momentary service unavailability can cause a request to fail or hang. Writing custom `for` loops in your business logic to retry failed requests is tedious and clutters your codebase.

gRPC solves this gracefully by supporting **Transparent Retries and Hedging** natively inside the gRPC core using a JSON `ServiceConfig`. You can define different policies for different RPC methods in the same service:
- **Retries (Exponential Backoff)**: Used for `UpdateUser`. If the client encounters specific status codes (e.g., `UNAVAILABLE`), it automatically retries with increasing delays between attempts.
- **Hedging**: Used for `GetUser`. Designed for latency-critical applications. If the server takes too long to respond (e.g., > 50ms), the client proactively fires a second identical request in parallel. Whichever response comes back first is used, and the slower one is cancelled.

## 🚀 How to Run & Test

1. **Start the Unified Server**  
   Open a terminal and run the server. It is programmed to simulate two distinct scenarios: 
   - `GetUser`: Simulates network latency (hangs for 200ms on the first attempt).
   - `UpdateUser`: Simulates network failure (returns `UNAVAILABLE` for the first 2 attempts).
   ```bash
   go run 17-retries/server/main.go
   ```

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 17-retries/client/main.go
   ```

   **Expected Behavior:**

   **Scenario A: Hedging (`GetUser`)**
   - The client sends the first request. The server deliberately hangs for 200ms.
   - After exactly 50ms (the `HedgingDelay`), the client grows impatient and fires a second request in parallel.
   - The server processes the second request instantly and returns it.
   - The client accepts the fast response and immediately *cancels* the first slow request. In the server logs, you will see: `[GET] request CANCELED by client (Hedging won!)`.

   **Scenario B: Retries (`UpdateUser`)**
   - The client initiates the update. The server rejects the first two attempts with `UNAVAILABLE`.
   - The client automatically backs off (e.g., 100ms, then 200ms) and tries a third time.
   - The server accepts the third attempt. The client side only sees a single successful RPC call!

## 📝 Notes
- Notice that the Context Timeout (`context.WithTimeout`) defined by the client must be large enough to accommodate *all* the retry/hedging attempts and their backoff delays. If the overall timeout is reached, the RPC fails regardless of remaining attempts.
- **When to use which?** 
  - Hedging is great for reducing tail latency but increases load on your servers (because multiple identical requests are processed). Only use it for safe, **idempotent** operations (like `GET` requests).
  - Retries are safer for mutations (like `UPDATE` or `DELETE`), provided the operations are also designed to be idempotent on the backend.
