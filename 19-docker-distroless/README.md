# 19 - Multi-stage Docker & Distroless

## 🎯 Purpose
In previous topics, we focused on writing Go code. But how do you package a gRPC service for Kubernetes safely and efficiently?

Using standard Docker images like `golang:latest` or `ubuntu` in production is a massive anti-pattern. They contain package managers, shells, and system utilities that create a huge attack surface for hackers (CVEs) and bloat the image size to over 800MB.

This topic demonstrates the enterprise-standard deployment method: **Multi-stage Builds with Google Distroless**.
- **Multi-stage:** Stage 1 uses a heavy Go environment to compile the code. Stage 2 copies *only* the compiled binary into the final image, discarding the heavy toolchain.
- **Distroless:** The final image (`gcr.io/distroless/static-debian11`) contains absolutely nothing except the CA certificates and timezone data. No `bash`, no `curl`, no `apt`. 

This approach drops your image size from ~800MB down to roughly **15-40MB**, and effectively neutralizes entire classes of security vulnerabilities.

## 🚀 How to Run & Test

1. **Build the Docker Image**

    Run this command from the repository root:

    ```bash
    docker build -t grpc-distroless-demo -f 19-docker-distroless/Dockerfile .
    ```

2. **Verify the Image Size**

    Check how incredibly small the resulting image is:

    ```bash
    docker images | grep grpc-distroless-demo
    ```

3. **Run the Container**

    Start the gRPC server inside the container, mapping port 50051 to your host:

    ```bash
    docker run --rm -p 50051:50051 --name grpc-server-instance grpc-distroless-demo:latest
    ```

4. **Test the Server**

    Use `grpcurl` or the client from Topic 1 to verify the server is running perfectly inside the isolated container:

    ```bash
    grpcurl -plaintext -d '{"user_id": "usr_docker"}' localhost:50051 user.v1.UserService/GetUser
    ```

## 📝 Notes

* **CGO_ENABLED=0**: This environment variable in the builder stage is absolutely critical. It tells the Go compiler to statically link all C libraries. If you forget this, your binary will crash inside the Distroless image complaining about missing `libc`.
* **Debugging:** Because Distroless has no shell (`/bin/sh`), you cannot use `docker exec -it <container> sh` to debug production issues. If you need a shell for debugging in a staging environment, use the `:debug` tag (e.g., `gcr.io/distroless/static-debian11:debug`), which includes a busybox shell.
* **Security:** Notice the `USER nonroot:nonroot` directive. Even if an attacker somehow exploits your gRPC code, they will have zero root privileges inside an empty container.
