# 12 - Rate Limiting

## 🎯 Purpose
When exposing APIs (especially public ones), you need to protect your backend services from being overwhelmed by too many requests (either malicious DDoS or accidental traffic spikes).
This topic demonstrates how to implement **Rate Limiting** in gRPC using:
- **Interceptors**: Intercepting every incoming request to check the rate limit before processing it.
- **Token Bucket Algorithm**: A common rate-limiting algorithm that allows short bursts of traffic while enforcing a steady long-term rate. (e.g., using `golang.org/x/time/rate`).

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server. It is configured to only allow a certain number of requests per second.
   ```bash
   go run 12-rate-limiting/server/main.go
   ```

2. **Run the Client**  
   Open another terminal and run the client. The client will intentionally send requests in a rapid burst to trigger the limit.
   ```bash
   go run 12-rate-limiting/client/main.go
   ```
   - Initially, the server will process the requests normally.
   - Once the token bucket is exhausted, the server interceptor will reject further requests immediately, returning a `ResourceExhausted` (HTTP 429 equivalent) status code.
   - The client will log these errors.

## 📝 Notes
- In a distributed environment with multiple server instances, in-memory rate limiting like this only limits traffic *per instance*.
- For cluster-wide rate limiting, you would typically use a centralized store (like Redis) or an API Gateway / Service Mesh proxy (like Envoy or Nginx).
