# 99 - Production-Ready gRPC Cluster

This project demonstrates an enterprise-grade, highly concurrent microservices architecture in Go. It completely eliminates traditional sidecar proxies by leveraging a custom **xDS Control Plane** for native gRPC load balancing and dynamic Kubernetes endpoint discovery.

## 🏗️ Architecture

The system consists of three core services and a centralized brain:

* **API Gateway:** Multiplexes HTTP/1.1 (REST) and HTTP/2 (gRPC) on port `:8080` using `cmux`, protected by a Circuit Breaker.
* **xDS Control Plane:** Watches the K8s Endpoints API and dynamically pushes LDS, RDS, CDS, and EDS updates to gRPC clients.
* **Order Service:** Validates Protobuf `v99` requests, enforces Rate Limiting, and acts as an xDS client routing to the User Service.
* **User Service:** The leaf node returning simulated data and the resolving Pod hostname.

```
[K6 / Browser / curl]
       │
       │ HTTP/1.1 REST  or  gRPC (HTTP/2)
       ▼
┌──────────────────────────────────────────┐
│             API Gateway                  │  :8080 (cmux → gRPC + REST)
│  gRPC-Gateway + cmux + Circuit Breaker   │
│  + Rate Limiter + OTel + Keepalive       │
└──────────┬───────────────────────────────┘
           │ xds:///order-service
           ▼
┌──────────────────────────────┐
│  xDS Control Plane           │  :15010  (Watches K8s Endpoints API)
│  (Dynamic, real-time update) │  Auto-updates when any pod scales
└──────────┬───────────────────┘
           │ Routes to discovered pods
     ┌─────┴──────┐
     ▼            ▼
┌─────────┐  ┌─────────┐
│ Order-1 │  │ Order-2 │  :50068  (Validation + Rate Limit + OTel + Keepalive)
└────┬────┘  └────┬────┘
     └──────┬─────┘
            │ xds:///user-service
            ▼
┌──────────────────────────────┐
│  xDS Control Plane           │  (Same server, same port, different snapshot)
└──────────┬───────────────────┘
     ┌─────┴──────┐
     ▼            ▼
┌─────────┐  ┌─────────┐
│  User-1 │  │  User-2 │  :50069  (OTel + Keepalive + Prometheus)
└─────────┘  └─────────┘
```

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
eval $(minikube docker-env)  # Build images inside Minikube

# 2. Build minimal images (use BuildKit to avoid 18-minute builds!)
DOCKER_BUILDKIT=1 docker build -t xds-controlplane:latest -f 99-production-ready-cluster/xds-control-plane/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t api-gateway:latest -f 99-production-ready-cluster/api-gateway/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t order-service:latest -f 99-production-ready-cluster/order-service/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t user-service:latest -f 99-production-ready-cluster/user-service/Dockerfile .

# 3. Deploy Observability Stack
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

kubectl create ns monitoring

helm upgrade --install jaeger jaegertracing/jaeger -n monitoring -f 99-production-ready-cluster/k8s/monitoring/jaeger-values.yaml

helm upgrade --install prom prometheus-community/kube-prometheus-stack -n monitoring -f 99-production-ready-cluster/k8s/monitoring/prometheus-values.yaml

# 4. Deploy Microservices
kubectl apply -f 99-production-ready-cluster/k8s/
```

## 🔬 Observability & Load Testing

Stress-test the environment to observe the Circuit Breaker and Go GC in action using the provided K6 script.

```bash

# Forward Jaeger port
kubectl port-forward -n monitoring svc/jaeger 16686:16686

# Forward Grafana port
kubectl port-forward -n monitoring svc/prom-grafana 3000:80

# Open gateway port
kubectl port-forward svc/api-gateway 30080:8080

# Test microservices
curl -X POST http://localhost:30080/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": "usr_1", "item_id": "item_42", "quantity": 2}'

# Run the C10K spike test
k6 run loadtest/k6-script.js

```

Access Jaeger at `http://localhost:16686` and Grafana at `http://localhost:3000` via K8s port-forwarding to analyze traces and memory limits.
