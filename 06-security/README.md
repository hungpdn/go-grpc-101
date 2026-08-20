# 06 - Security

## 🎯 Purpose
In a production environment, especially within a **Zero Trust Architecture**, internal services must not trust each other by default.
This topic demonstrates how to secure your gRPC endpoints using:
- **TLS (Transport Layer Security)**: Encrypting the connection so data cannot be intercepted in transit.
- **mTLS (Mutual TLS)**: Both the client and the server authenticate each other using cryptographic certificates.
- **Token-based Authentication**: Passing credentials via `credentials.PerRPCCredentials`, which automatically injects the token into every request *but strictly requires a secure TLS connection to prevent token leaks*.

## 🚀 How to Run & Test

1. **Generate the SSL/TLS Certificates**  
   We need a Certificate Authority (CA) and certificates for both the Server and the Client.
   Run these commands in your terminal at the root of the project:
   ```bash
   mkdir -p 06-security/certs
   cd 06-security/certs

   # 1. Generate CA (Certificate Authority)
   openssl req -x509 -newkey rsa:4096 -days 365 -nodes -keyout ca-key.pem -out ca-cert.pem -subj "/C=SG/O=YourOrg/CN=TestCA"

   # 2. Generate Server Cert & Key
   openssl req -newkey rsa:4096 -nodes -keyout server-key.pem -out server-req.pem -subj "/C=SG/O=YourOrg/CN=localhost"
   openssl x509 -req -in server-req.pem -days 365 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1")

   # 3. Generate Client Cert & Key
   openssl req -newkey rsa:4096 -nodes -keyout client-key.pem -out client-req.pem -subj "/C=SG/O=YourOrg/CN=client-service"
   openssl x509 -req -in client-req.pem -days 365 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out client-cert.pem

   # Cleanup request files
   rm *-req.pem
   cd ../..
   ```

2. **Start the Server**  
   Open a terminal and run the secure gRPC server:
   ```bash
   go run 06-security/server/main.go
   ```

3. **Run the Client**  
   Open another terminal and run the secure client:
   ```bash
   go run 06-security/client/main.go
   ```

## 📝 Notes
- The `certs/` directory contains demo/self-signed certificates purely for development purposes. **Do not use them in production**.
