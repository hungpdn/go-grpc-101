# 09 - Load Balancing

## 🎯 Purpose
Unlike HTTP/1.1 where you can easily load balance per request, gRPC uses HTTP/2 which multiplexes many requests over a single long-lived TCP connection. If you simply put a standard L4 load balancer (like HAProxy or AWS NLB) in front, all requests from a single client will stick to the same backend server.
This topic demonstrates **Client-side Load Balancing** in gRPC, where the client is aware of multiple backend servers and distributes requests across them (e.g., Round Robin) on a per-RPC basis.

Reference: [gRPC Load Balancing](https://grpc.io/blog/grpc-load-balancing)

## 🚀 How to Run & Test

1. **Start Multiple Servers**  
   You need to start a few instances of the server on different ports. Open multiple terminals:
   ```bash
   go run 09-load-balancing/server/main.go -port 50051
   go run 09-load-balancing/server/main.go -port 50052
   go run 09-load-balancing/server/main.go -port 50053
   ```

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 09-load-balancing/client/main.go
   ```
   The client will send a burst of requests. If you check the server terminals, you should see the requests being distributed evenly among them (Round Robin).

## 📝 Notes
- A Custom Resolver (or a Name Resolver like DNS) is used by the client to discover all available backend IP addresses.
- The Load Balancer policy is configured via `grpc.WithDefaultServiceConfig('{"loadBalancingPolicy":"round_robin"}')`.
- For production, service meshes like Istio or L7 load balancers (Envoy/ALB) often handle this transparently, but client-side LB is highly efficient for internal service-to-service communication.
