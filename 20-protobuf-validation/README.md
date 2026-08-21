# 20 - Protobuf Validation

## 🎯 Purpose

In gRPC, there is a strict difference between **Business Validation** (e.g., "Does this email exist in our DB?") and **Schema Validation** (e.g., "Is this a valid email format?", "Is the age >= 18?").

Writing `if req.Age < 18 { return err }` inside your Go business logic leads to massive boilerplate and spaghetti code. 

This topic solves that by using **`buf.build/go/protovalidate`**. We define validation rules directly inside the `.proto` file. A gRPC Interceptor then automatically intercepts incoming requests, validates the payload against these rules, and immediately rejects bad data with an `InvalidArgument` (400) error before it ever reaches your business logic. The `.proto` file becomes a rock-solid Data Contract.

## 🚀 How to Run & Test

1. **Install Dependencies**

Update your Go modules to include the new validation library:

```bash
go get buf.build/go/protovalidate
go mod tidy
```

2. **Generate the Protobuf Code**

Make sure you have added `buf.build/bufbuild/protovalidate` to your `buf.yaml` dependencies, then run:

```bash
buf dep update proto
make gen
```

3. **Start the Validation Server**

Run the server which implements the validation interceptor:

```bash
go run 20-protobuf-validation/server/main.go
```

4. **Trigger an Invalid Request (The Client)**

In a new terminal, run the client. The client is programmed to send invalid data (name too short, bad email format, age < 18):

```bash
go run 20-protobuf-validation/client/main.go
```

**Expected Output on Client:**
```json
{
  "level": "ERROR",
  "msg": "server rejected the request",
  "error": "rpc error: code = InvalidArgument desc = validation error:\n - name: value length must be at least 3 runes [string.min_len]\n - email: value must be a valid email address [string.email]\n - age: value must be greater than or equal to 18 [int32.gte]"
}
```

## 📝 Notes

* **Zero Boilerplate:** Look at the server's business logic. It has zero `if` statements checking for empty strings. It assumes the data is 100% valid.
* **Backward Compatibility:** Never add strict validation rules to an existing production `.proto` file (e.g., `v1`). Older mobile clients sending historically "valid" (but now invalid) data will crash. Always introduce strict validations in a new API version (e.g., `v2`).
* **Performance:** `protovalidate` pre-compiles validation rules using reflection on startup, making the per-request validation extremely fast.
