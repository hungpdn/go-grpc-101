# 23 - The C10K & C100K Problem (OS Tuning)

## 🎯 Purpose

The **C10K Problem** refers to the challenge of optimizing network sockets to handle 10,000 (or up to millions) of concurrent connections. 

Golang's `net` package and gRPC use an internal **netpoller** (based on Linux `epoll`). This allows a single OS thread to handle thousands of idle gRPC connections efficiently. The Go software is naturally ready for C100K. 

However, the underlying **Linux Operating System is not ready by default**. Linux has conservative safety limits. If you run a load test without tuning the OS, the OS kernel will actively drop connections, resulting in `too many open files` or `connection reset by peer` errors before your Go code even breaks a sweat.

## ⚙️ The Four Bottlenecks

1. **File Descriptors:** Every TCP connection is a file. Linux defaults to ~1024 open files per process. We must increase `fs.file-max` to support millions of connections.
2. **Connection Backlog:** If thousands of clients connect simultaneously, they sit in the OS queue until gRPC calls `Accept()`. The default queue (`somaxconn`) is only 128. We must increase it to `65535`.
3. **Port Exhaustion:** When your gRPC server acts as a client calling other microservices, it uses a random (ephemeral) port. We must widen `ip_local_port_range` to `1024 65535`.
4. **TIME_WAIT State:** Closed connections hold their ports in a `TIME_WAIT` state for ~60s. Under heavy load, you run out of ports. We enable `tcp_tw_reuse` to recycle them instantly.

## 🚀 How to Apply (Linux/Ubuntu)

To apply the configurations found in `optimized-sysctl.conf` to a physical Linux server or VM:

```bash
# Append the configs to sysctl.conf
cat optimized-sysctl.conf | sudo tee -a /etc/sysctl.conf

# Reload sysctl to apply instantly without reboot
sudo sysctl -p
```

## 🐳 How to Apply in Kubernetes

In modern Cloud-Native architectures, you don't edit the VM's OS directly. You pass these tunings at the Pod level using `securityContext`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: grpc-high-load-server
spec:
  securityContext:
    sysctls:
    - name: net.core.somaxconn
      value: "65535"
    - name: net.ipv4.tcp_tw_reuse
      value: "1"

```

*(Note: Some sysctls are considered "unsafe" by K8s and require cluster admin permission to enable on the Node kubelet).*

## 🛡️ Go Runtime Tuning (Anti-OOM)
When running high-load gRPC servers in Kubernetes, the OS network is not the only bottleneck. Spikes can cause sudden memory allocations, leading to Kubernetes **OOMKilled** events.

Always configure the Go runtime in your Pod's environment variables:

```yaml
env:
  # Instructs Go's Garbage Collector to strictly keep memory under this limit, 
  # triggering aggressive GC cycles before the K8s OOM Killer terminates the pod.
  # (Recommended: 90% of pod memory limit)
  - name: GOMEMLIMIT
    value: "900MiB" 
  
  # Optional: Tune GC frequency (Default is 100). 
  # Lower values (e.g., 50) use more CPU but keep RAM usage tighter.
  - name: GOGC
    value: "100"
```
