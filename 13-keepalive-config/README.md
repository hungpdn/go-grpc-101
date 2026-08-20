# 13 - Keepalive & Connection Management

## 🎯 Purpose
Network intermediaries like Load Balancers, Routers, or Firewalls often automatically terminate idle TCP (HTTP/2) connections after a certain period of inactivity (Idle Timeout). This can cause silent connection drops and errors when the client tries to reuse a stale connection.
This topic demonstrates how to configure **gRPC Keepalive** to prevent this:
- **Client Parameters**: The client periodically sends lightweight PING frames to the server (e.g., every 10 seconds) to signal to the Load Balancer that the connection is still active.
- **Server Parameters & Enforcement Policy**: The server can also ping the client to check if it's alive, drop connections that have been idle for too long (to free up memory), and enforce rate limits on client PINGs to protect against DDoS.

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server:
   ```bash
   go run 13-keepalive-config/server/main.go
   ```
   The server is configured to drop connections idle for more than 5 minutes and enforce a minimum PING interval from clients.

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 13-keepalive-config/client/main.go
   ```
   - The client will make a successful initial request.
   - It will then sleep for 1 minute. During this time, the client's keepalive background process will automatically send PING frames every 10 seconds.
   - After waking up, the second request succeeds immediately with zero cold-start delay because the connection was actively kept alive!

## 📝 Notes
- Without Keepalive, the client's connection might be silently dropped by a network proxy during the 1-minute sleep, causing the second request to fail or hang.
- Always configure an `EnforcementPolicy` on the server if you allow clients to send Keepalive PINGs, otherwise malicious clients could spam PINGs and overload the server.
- `PermitWithoutStream: true` is crucial if you want to keep the connection alive even when there are no active RPC streams happening.
