# 99 - Production-Ready gRPC Cluster

This capstone project demonstrates an enterprise-grade, highly concurrent microservices architecture in Go. It completely eliminates traditional sidecar proxies by leveraging a custom **xDS Control Plane** for native gRPC load balancing and dynamic Kubernetes endpoint discovery.

## 🏗️ Architecture

The system consists of three core services and a centralized brain:

* **API Gateway:** Multiplexes HTTP/1.1 (REST) and HTTP/2 (gRPC) on port `:8080` using `cmux`, protected by a Circuit Breaker.
* **xDS Control Plane:** Watches the K8s Endpoints API and dynamically pushes LDS, RDS, CDS, and EDS updates to gRPC clients.
* **Order Service:** Validates Protobuf `v99` requests, enforces Rate Limiting, and acts as an xDS client routing to the User Service.
* **User Service:** The leaf node returning simulated data and the resolving Pod hostname.

## ✨ Key Features

* **Native xDS Load Balancing:** Zero-proxy architecture for microsecond network latency.
* **Anti-OOM Kubernetes Tuning:** Implements `GOMEMLIMIT` and `GOGC` to force Go runtime garbage collection before K8s resource limits are breached.
* **Distroless Multi-stage Builds:** Final Docker images are ultra-secure and under 30MB.
* **Distributed Observability:** Full OpenTelemetry (OTel) trace propagation via Jaeger and Prometheus metrics scraping.

## 🚀 Deployment Guide

Ensure you have Minikube, Helm, and Docker installed.

```bash
# 1. Start cluster and point Docker to Minikube
minikube start --cpus=4 --memory=8g
eval $(minikube docker-env)

# 2. Build minimal images (use BuildKit to avoid 18-minute builds!)
DOCKER_BUILDKIT=1 docker build -t xds-controlplane:latest -f 99-production-ready-cluster/xds-control-plane/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t api-gateway:latest -f 99-production-ready-cluster/api-gateway/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t order-service:latest -f 99-production-ready-cluster/order-service/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t user-service:latest -f 99-production-ready-cluster/user-service/Dockerfile .

# 3. Deploy Observability Stack
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
kubectl create ns monitoring
helm upgrade --install prom prometheus-community/kube-prometheus-stack -n monitoring -f k8s/monitoring/prometheus-values.yaml

# 4. Deploy Microservices
kubectl apply -f k8s/
```

## 🔬 Observability & Load Testing

Stress-test the environment to observe the Circuit Breaker and Go GC in action using the provided K6 script.

```bash
# Open gateway port
kubectl port-forward svc/api-gateway 30080:8080

# Run the C10K spike test
k6 run loadtest/k6-script.js

```

Access Jaeger at `http://localhost:16686` and Grafana at `http://localhost:3000` via K8s port-forwarding to analyze traces and memory limits.
