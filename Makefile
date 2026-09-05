.PHONY: lint gen tidy test dep build-all cluster-build cluster-deploy cluster-teardown

# Update the protobuf dependencies
dep:
	buf dep update proto

# Lint the protobuf files against industry standards
lint:
	buf lint proto

# Generate the Go code using buf.gen.yaml
gen:
	buf generate proto

tidy:
	go mod tidy

# Run all unit tests
test:
	go test -v ./...

# Build all Go binaries in the repo to verify compilation (useful for CI)
build-all:
	go build ./...

# =======================================================
# Topic 99: Production-Ready Cluster on Minikube
# =======================================================

# Build all 4 distroless Docker images (must run `eval $(minikube docker-env)` first)
cluster-build:
	DOCKER_BUILDKIT=1 docker build -t xds-controlplane:latest -f 99-production-ready-cluster/xds-control-plane/Dockerfile .
	DOCKER_BUILDKIT=1 docker build -t api-gateway:latest      -f 99-production-ready-cluster/api-gateway/Dockerfile .
	DOCKER_BUILDKIT=1 docker build -t order-service:latest    -f 99-production-ready-cluster/order-service/Dockerfile .
	DOCKER_BUILDKIT=1 docker build -t user-service:latest     -f 99-production-ready-cluster/user-service/Dockerfile .

# Deploy all K8s resources: namespace, configmap, secrets, RBAC, services, networking, monitoring
cluster-deploy:
	kubectl apply -f 99-production-ready-cluster/k8s/

# Teardown: Delete all cluster resources (keeps Minikube running)
cluster-teardown:
	kubectl delete -f 99-production-ready-cluster/k8s/ --ignore-not-found