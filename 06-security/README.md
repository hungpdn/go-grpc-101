In a production environment, especially within a **Zero Trust Architecture**, internal services must not trust each other by default.

Standard TLS (like HTTPS) only proves the Server's identity to the Client. **Mutual TLS (mTLS)** ensures that the Server also verifies the Client's identity using cryptographic certificates.

Furthermore, injecting tokens manually via `metadata.AppendToOutgoingContext` (like we did in Topic 3) is error-prone. The idiomatic gRPC way is using **`credentials.PerRPCCredentials`**, which automatically injects the token into every request *but strictly requires a secure TLS connection to prevent token leaks*.

---

## Generate the SSL/TLS Certificates

We need a Certificate Authority (CA) and certificates for both the Server and the Client.

Run these commands in your terminal at the root of the project to generate local test certificates:

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
