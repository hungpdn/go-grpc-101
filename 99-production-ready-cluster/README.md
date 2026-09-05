# 99 - Production-Ready gRPC Cluster

This project demonstrates an enterprise-grade, highly concurrent microservices architecture in Go. It completely eliminates traditional sidecar proxies by leveraging a custom **xDS Control Plane** for native gRPC load balancing and dynamic Kubernetes endpoint discovery.

## 🏗️ Architecture

The system consists of three core services and a centralized brain:

* **Ingress (Nginx):** Single production entrypoint routing external traffic (`api.grpc-cluster.local`) to the API Gateway.
* **API Gateway:** Multiplexes HTTP/1.1 (REST) and HTTP/2 (gRPC) on port `:8080` using `cmux`, protected by a Circuit Breaker.
* **xDS Control Plane:** Watches the K8s Endpoints API and dynamically pushes LDS, RDS, CDS, and EDS updates to gRPC clients.
* **Order Service:** Validates Protobuf `v99` requests, enforces Rate Limiting, autoscales with **HPA** under load, and acts as an xDS client routing to the User Service.
* **User Service:** Leaf node returning simulated data and resolving Pod hostname.

```
[K6 / External Clients]
       │
       ▼
┌──────────────────────────────────────────┐
│      Nginx Ingress Controller            │  (api.grpc-cluster.local)
└──────────────────┬───────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────┐
│             API Gateway                  │  :8080 (cmux → gRPC + REST)
│  gRPC-Gateway + cmux + Circuit Breaker   │  Protected by HPA & PDB
│  + Rate Limiter + OTel + Keepalive       │  (ConfigMap mounted xDS bootstrap)
└──────────┬───────────────────────────────┘
           │ xds:///order-service
           ▼
┌──────────────────────────────┐
│  xDS Control Plane           │  :15010 (Watches K8s Endpoints API)
│  (Dynamic, real-time update) │  Auto-discovers new pods when HPA scales!
└──────────┬───────────────────┘
           │ Routes to discovered pods
     ┌─────┴──────┐
     ▼            ▼
┌─────────┐  ┌─────────┐
│ Order-1 │  │ Order-2 │  :50068 (Validation + Rate Limit + OTel + Keepalive)
└────┬────┘  └────┬────┘  [Autoscaled by HPA up to 6 replicas]
     └──────┬─────┘
            │ xds:///user-service
            ▼
┌──────────────────────────────┐
│  xDS Control Plane           │  (Same server, same port, different snapshot)
└──────────┬───────────────────┘
     ┌─────┴──────┐
     ▼            ▼
┌─────────┐  ┌─────────┐
│  User-1 │  │  User-2 │  :50069 (OTel + Keepalive + Prometheus + PDB)
└─────────┘  └─────────┘
```

## ✨ Enterprise Production Features

1. **Native xDS Load Balancing:** Zero-proxy architecture for microsecond network latency.
2. **Centralized ConfigMap & Secrets:** xDS client configuration decoupled into `configmap.yaml` and credentials into `secret.yaml`.
3. **Horizontal Pod Autoscaling (HPA):** `hpa.yaml` automatically scales `order-service` (2 → 6 replicas) when CPU > 70% under K6 load.
4. **Pod Disruption Budget (PDB):** `pdb.yaml` guarantees at least 1 healthy replica during node maintenance or rolling updates.
5. **Zero-Trust NetworkPolicy:** `network-policy.yaml` restricts inter-service traffic: only the Gateway can call `order-service`, and only `order-service` can call `user-service`.
6. **High-Availability Anti-Affinity:** Pod anti-affinity rules prevent replicas of the same service from colocation on the same physical node.
7. **Liveness & Readiness Probes:** Native gRPC Health Probes (`grpc_health_v1`) ensure zero-downtime routing during rolling deployments.
8. **Anti-OOM Kubernetes Tuning:** `GOMEMLIMIT` and `GOGC` force Go GC cycles before K8s hard limits are hit.
9. **Distroless Multi-Stage Builds:** Ultra-secure containers under 30MB with zero package managers or shells.
10. **Distributed Observability:** OpenTelemetry (OTel) distributed traces exported to Jaeger and metrics scraped by Prometheus.

---

## 🚀 Deployment Guide

### Prerequisites

- Minikube with at least 4 CPUs and 8GB RAM.
- Docker or Podman.
- Helm & kubectl.

```bash
# 1. Start cluster and enable required Minikube addons
minikube start --cpus=4 --memory=8g
minikube addons enable ingress
minikube addons enable metrics-server

# 2. Point your environment to Minikube's container daemon
# If using Docker:
eval $(minikube docker-env)

# If using Podman:
# eval $(minikube podman-env)

# 3. Build minimal images (or run `make cluster-build`)
DOCKER_BUILDKIT=1 docker build -t xds-controlplane:latest -f 99-production-ready-cluster/xds-control-plane/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t api-gateway:latest -f 99-production-ready-cluster/api-gateway/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t order-service:latest -f 99-production-ready-cluster/order-service/Dockerfile .
DOCKER_BUILDKIT=1 docker build -t user-service:latest -f 99-production-ready-cluster/user-service/Dockerfile .

# 4. Deploy Observability Stack (Jaeger + Prometheus + Grafana)
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

kubectl create ns monitoring

helm upgrade --install jaeger jaegertracing/jaeger -n monitoring -f 99-production-ready-cluster/k8s/monitoring/jaeger-values.yaml
helm upgrade --install prom prometheus-community/kube-prometheus-stack -n monitoring -f 99-production-ready-cluster/k8s/monitoring/prometheus-values.yaml

# 5. Deploy all Microservices & K8s Governance (ConfigMap, Secret, NetworkPolicy, HPA, PDB, Ingress)
kubectl config use-context minikube
kubectl apply -f 99-production-ready-cluster/k8s/
```

---

## 🔬 Observability, Autoscaling & Load Testing

### 1. Test via Ingress
The current Ingress rule uses the host `localhost`. To expose the Minikube
Ingress controller on `127.0.0.1:80`, keep this command running in a separate
terminal:

```bash
minikube tunnel
```

Then test the API with the matching `Host` header:

```bash
curl -X POST http://127.0.0.1/v1/orders \
  -H "Host: localhost" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "usr_prod_1", "item_id": "item_galaxy", "quantity": 2}'
```

The tunnel process must remain running while making requests. `minikube
tunnel` does not change the Ingress host rule; sending
`Host: api.grpc-cluster.local` to the current manifest will not match the
`localhost` rule.

To use `api.grpc-cluster.local` instead, change `spec.rules[].host` in
`k8s/ingress.yaml`, map the name to the Minikube IP, and call the name directly:

```bash
echo "127.0.0.1 api.grpc-cluster.local" | sudo tee -a /etc/hosts

curl -X POST http://api.grpc-cluster.local/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": "usr_prod_1", "item_id": "item_galaxy", "quantity": 2}'
```

If curl reports `Failed to connect to 127.0.0.1 port 80`, the tunnel is not
currently providing the local listener. Restart `minikube tunnel` and verify
that the terminal remains open. You can also inspect the controller and rule:

```bash
kubectl get ingress -A -o wide
kubectl -n ingress-nginx get svc ingress-nginx-controller
```

### 2. Monitor Autoscaling & Mesh Real-Time Balancing
In a separate terminal, watch HPA and Pod scaling:
```bash
kubectl get hpa -w
```

Run the K6 spike test (10,000 RPS):
```bash
k6 run 99-production-ready-cluster/loadtest/k6-script.js
```

**What happens behind the scenes:**
1. CPU spikes on `order-service`.
2. HPA spins up new `order-service` pods (up to 6 replicas).
3. The custom `xds-control-plane` detects new pod IPs in real-time via the K8s Endpoints informer.
4. `api-gateway` immediately begins load balancing requests across all 6 pods without restart!

### 3. Open Dashboards
```bash
# Jaeger Tracing UI
kubectl port-forward -n monitoring svc/jaeger 16686:16686

# Grafana Monitoring Dashboard (admin / admin123)
kubectl port-forward -n monitoring svc/prom-grafana 3000:80
```
- Access Jaeger at `http://localhost:16686` to trace end-to-end distributed requests.
- Access Grafana at `http://localhost:3000` to inspect CPU, GC frequency, and gRPC metrics.

## References

- https://github.com/envoyproxy/go-control-plane
- https://github.com/grpc/proposal/blob/master/A27-xds-global-load-balancing.md#xdsclient-and-bootstrap-file 
