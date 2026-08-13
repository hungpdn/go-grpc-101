.PHONY: lint gen tidy run-server run-client test

# Lint the protobuf files against industry standards
lint:
	buf lint proto

# Generate the Go code using buf.gen.yaml
gen:
	buf generate proto

tidy:
	go mod tidy

run-server:
	go run 01-unary-rpc/server/main.go

run-client:
	go run 01-unary-rpc/client/main.go

test:
	go test -v ./...