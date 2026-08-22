# 22 - xDS Proxyless Service Mesh (Control Plane & Client)

## 🎯 Purpose

In standard Cloud-Native architectures, Service Mesh tools (like Istio) inject an Envoy Proxy "sidecar" into every Pod to handle routing, load balancing, and retries. This adds network latency and consumes high CPU/Memory across the cluster.

gRPC supports **Proxyless Service Mesh**. By using the `xds:///` scheme, the gRPC client communicates directly with the Control Plane to dynamically download routing rules in real-time, completely eliminating the need for sidecar proxies. 

In this topic, we didn't just use a pre-built Istio setup. We built a **Mini Control Plane from scratch** using Envoy's `go-control-plane` library to understand the absolute core of how Service Mesh routing works under the hood.

## 🚀 How to Run & Test

You will need 3 separate terminals to simulate the full Service Mesh architecture.

1. **Start the Backend Server (The Data Plane)**

This is your actual business logic running on port `50055`.

```bash
go run 22-xds/server/main.go
```

2. **Start the xDS Control Plane (The Traffic Controller)**

This server runs on port `15010` and serves dynamic configuration via the xDS protocol.

```bash
go run 22-xds/controlplane/main.go
```

3. **Run the Proxyless Client**

Provide the `bootstrap.json` via environment variable so the gRPC client knows where the Control Plane is located.

```bash
export GRPC_XDS_BOOTSTRAP=$(pwd)/22-xds/bootstrap.json
go run 22-xds/client/main.go
```

**Expected Output on Client:**

```json
{"level":"INFO","msg":"Success","name":"xDSRouted User"}
```

## 📝 Notes & Principal Insights

gRPC's implementation of xDS is incredibly strict compared to Envoy. During development, we navigated several core architectural constraints:

* **The HCM Wrapper:** gRPC requires the `RouteConfiguration` to be strictly wrapped inside an `HttpConnectionManager` (HCM).
* **The Router Filter:** Even with HCM, gRPC will throw `http filters list is empty` unless you explicitly define `envoy.filters.http.router` as the terminal filter.
* **Locality & Health:** If endpoints lack geographic `Locality` (Region/Zone) or `HealthStatus`, gRPC will aggressively drop them, resulting in a `produced zero addresses` error.
* **Weighted Balancing:** Even with valid IPs, if the Locality lacks a `LoadBalancingWeight`, the gRPC load balancer will refuse to route traffic (`no targets to pick from`).

By manually configuring the chain (`Endpoint -> Locality -> Weight -> Cluster -> Route -> HCM -> Filter -> Listener`), you have essentially learned how the brains of Istio and Google Cloud Traffic Director operate!
