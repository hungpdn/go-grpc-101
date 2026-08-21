# 21 - Custom Load Balancer (Consistent Hashing)

## 🎯 Purpose

By default, gRPC provides `pick_first` (sends all traffic to one server) and `round_robin` (distributes traffic evenly). 

However, in massive scale systems like Multiplayer Games or Stateful Caches, you often need **Sticky Sessions**. If a user's data is cached on Server A, you want all subsequent requests from that user to go to Server A. If they go to Server B, it results in a cache miss and high latency.

This topic demonstrates how to write a Custom gRPC Load Balancer. We implement a **Consistent Hashing** algorithm that reads the `user_id` from the gRPC metadata, hashes it, and uses the modulo operator to consistently route that specific user to the exact same server every time.

## 🚀 How to Run & Test

1. **Start 3 Server Instances**  

Open 3 separate terminals to start 3 backend servers on different ports:

```bash
PORT=50071 go run 21-custom-lb/server/main.go
PORT=50072 go run 21-custom-lb/server/main.go
PORT=50073 go run 21-custom-lb/server/main.go
```

2. **Run the Custom Client**

In a 4th terminal, run the client:

```bash
go run 21-custom-lb/client/main.go
```

**Expected Output:**

```text
--- Routing User A ---
User A result handled_by="Handled by Server [:50072]"
User A result handled_by="Handled by Server [:50072]"
User A result handled_by="Handled by Server [:50072]"
--- Routing User B ---
User B result handled_by="Handled by Server [:50071]"
User B result handled_by="Handled by Server [:50071]"
User B result handled_by="Handled by Server [:50071]"
```

## 📝 Notes

* **The Picker:** The `Pick()` method is executed on the client-side for *every single RPC*. It must be extremely fast and thread-safe. Never put blocking calls (like DB lookups or network calls) inside `Pick()`.

* **Rebalancing:** When a server crashes, the `Builder.Build()` method is re-triggered automatically with the new list of surviving servers. Our modulo logic (`hash % len(servers)`) will seamlessly recalculate and route traffic to the remaining servers.

* **Advanced Insight:** Hashing algorithm using modulo in the example above is basic. In reality, if the number of servers changes (from 3 to 2), the entire index will be reversed (called Hash Ring Collapse). At the Enterprise level, people will use Consistent Hashing Ring libraries (such as Google's Jump Hash algorithm) to ensure that when a node dies, only a small fraction of users are redirected, while other users still connect to the correct old server!
