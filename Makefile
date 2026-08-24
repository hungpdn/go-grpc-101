.PHONY: lint gen tidy test dep

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

test:
	go test -v ./...