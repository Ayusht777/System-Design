# Load Balancer — System Design

A TCP-level load balancer written in Go using **Round-Robin** distribution across backend HTTP servers.

---

## Project Structure

```
LoadBalancer/
├── loadbalancer/main/main.go   # Core load balancer (TCP proxy)
├── miniserver1/main/main.go    # Backend server 1 (port 8081)
├── miniserver2/main/main.go    # Backend server 2 (port 8082)
└── test/main/main.go           # Stress tester / benchmarking tool
```

---

## How It Works

The load balancer listens on `:8080` (TCP), accepts incoming connections, and forwards them to backend servers in round-robin order using raw TCP proxying via `io.Copy`.

```
Client → :8080 (Load Balancer) → :8081 (Server 1)
                               → :8082 (Server 2)
```

Each accepted connection is handled in its own goroutine. Two goroutines per connection bidirectionally pipe data between client and backend concurrently.

```go
// Current active approach — concurrent bidirectional TCP proxy
func ForwardRequest(clientRequest net.Conn) {
    currentServerUrl := BackendServers[CurrentServerIndex]
    connectToServer, _ := net.Dial("tcp", currentServerUrl)
    defer connectToServer.Close()

    var wg sync.WaitGroup
    wg.Add(2)

    CurrentServerIndex = (CurrentServerIndex + 1) % len(BackendServers) // round-robin

    go func() { io.Copy(connectToServer, clientRequest); wg.Done() }() // client → backend
    go func() { io.Copy(clientRequest, connectToServer); wg.Done() }() // backend → client

    wg.Wait()
}
```

---

## Running Locally

**1. Start backend servers**
```bash
cd miniserver1/main && go run main.go   # :8081
cd miniserver2/main && go run main.go   # :8082
```

**2. Start the load balancer**
```bash
cd loadbalancer/main && go run main.go  # :8080
```

**3. Run the stress tester**
```bash
cd test/main && go run main.go -n 10000 -c 1000 -url http://localhost:8080/
```

| Flag   | Default                    | Description              |
|--------|----------------------------|--------------------------|
| `-n`   | `10000`                    | Total requests to send   |
| `-c`   | `1000`                     | Concurrent workers       |
| `-url` | `http://localhost:8080/`   | Target URL               |

---

## Benchmark Results

> 10,000 requests · 1,000 concurrent workers

```
Total Requests  :  10000
Successful      :  10000
Failed          :  0
Total Duration  :  2.651s

Throughput
  Requests/sec  :  3772.05
  Requests/min  :  226323

Latency
  Min           :  4.198ms
  Avg           :  234.143ms
  P50 (median)  :  134.878ms
  P90           :  572.944ms
  P99           :  770.838ms
  Max           :  873.333ms
```

---

## Known Issues & Fixes

### 1. Race Condition on `CurrentServerIndex`

`CurrentServerIndex` is a plain `int` shared across goroutines with no synchronization. Under concurrent load, multiple goroutines read and write it simultaneously — causing **data races**.

```go
// ❌ Unsafe — concurrent goroutines can read/write simultaneously
var CurrentServerIndex = 0
CurrentServerIndex = (CurrentServerIndex + 1) % len(BackendServers)
```

**Fix — use `sync/atomic` or a `sync.Mutex`:**

```go
// ✅ Option A: atomic (fastest, lock-free)
var currentIndex int64

func nextServer() string {
    idx := atomic.AddInt64(&currentIndex, 1)
    return BackendServers[idx % int64(len(BackendServers))]
}

// ✅ Option B: mutex (clearer intent)
var (
    mu           sync.Mutex
    serverIndex  int
)

func nextServer() string {
    mu.Lock()
    defer mu.Unlock()
    url := BackendServers[serverIndex]
    serverIndex = (serverIndex + 1) % len(BackendServers)
    return url
}
```

---

### 2. `ForwardRequestWithoutThread` — Blocking I/O (Deprecated)

The original implementation blocks on `io.Copy` — the balancer stalls until the full response is received before it can accept another request.

```go
// ❌ Blocking — one request at a time, kills throughput
func ForwardRequestWithoutThread(clientRequest net.Conn) error {
    connectToServer, _ := net.Dial("tcp", currentServerUrl)
    io.Copy(connectToServer, clientRequest) // blocks here
    io.Copy(clientRequest, connectToServer) // then blocks here
    return nil
}
```

**Fix** — already implemented in `ForwardRequest`: run both `io.Copy` calls concurrently with goroutines + `sync.WaitGroup` (see active approach above).

---

### 3. No Health Checks

If a backend server is down, the balancer still routes traffic to it and the connection fails silently.

```go
// ❌ No check — dials dead server, logs error, drops request
connectToServer, err := net.Dial("tcp", currentServerUrl)
if err != nil {
    fmt.Println(err)
    return
}
```

**Fix — skip unhealthy backends:**

```go
// ✅ Try next server on dial failure
func nextHealthyServer(tried int) (net.Conn, string, error) {
    for i := 0; i < len(BackendServers); i++ {
        url := nextServer()
        conn, err := net.Dial("tcp", url)
        if err == nil {
            return conn, url, nil
        }
    }
    return nil, "", fmt.Errorf("all backends unavailable")
}
```

---

### 4. No Connection Timeout

`net.Dial` has no deadline. A slow or unresponsive backend holds the goroutine open indefinitely, eventually exhausting the goroutine pool under load.

```go
// ❌ Hangs forever if backend is slow/unresponsive
connectToServer, err := net.Dial("tcp", currentServerUrl)
```

**Fix — add dial and connection timeouts:**

```go
// ✅ Timeout-aware dial
connectToServer, err := net.DialTimeout("tcp", currentServerUrl, 2*time.Second)

// ✅ Set read/write deadline on the connection
connectToServer.SetDeadline(time.Now().Add(10 * time.Second))
```

---

## What's Next

- [ ] Atomic round-robin index (fix race condition)
- [ ] Health check loop — remove dead servers from rotation
- [ ] Connection + read/write timeouts
- [ ] Weighted round-robin (route more traffic to faster servers)
- [ ] Metrics endpoint (`/metrics`) — expose RPS, active connections, error rate