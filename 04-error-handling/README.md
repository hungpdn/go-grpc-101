# 04 - Error Handling

## 🎯 Purpose
This topic demonstrates how to properly handle errors in gRPC. Instead of generic errors, gRPC uses standard status codes (e.g., `NotFound`, `InvalidArgument`, `Internal`). Additionally, it shows how to attach **Error Details** (Rich Errors) to provide more context to the client, such as field violations or localized error messages.

## 🚀 How to Run & Test

1. **Start the Server**  
   Open a terminal and run the gRPC server:
   ```bash
   go run 04-error-handling/server/main.go
   ```

2. **Run the Client**  
   Open another terminal and run the client:
   ```bash
   go run 04-error-handling/client/main.go
   ```
   The client will intentionally trigger error scenarios (like sending an invalid ID). 
   - You will see how the server returns a gRPC status code and attaches error details (e.g., `errdetails.BadRequest`).
   - The client will parse these details to display exactly what went wrong.

## 📝 Notes
- Use `status.Error` or `status.Errorf` to return basic gRPC status codes.
- Use `st.WithDetails()` to attach protobuf message details to the error before returning it.
- Never rely on plain Go string errors (`errors.New`) because the client won't be able to distinguish the error type properly.
