# High-Impact Performance Improvements

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce p99 latency and eliminate GC pauses by implementing 5 high-impact optimizations.

**Architecture:** Replace stdlib HTTP + goccy/go-json with custom HTTP/1.1 server and hand-written JSON scanner. Disable GC after startup. Add bbox pruning to IVF search. Add prefetch to assembly.

**Tech Stack:** Go, AVX2 assembly, Unix domain sockets

---

### Task 1: Disable GC after startup

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: Add GC disable after index loading**

In `cmd/server/main.go`, add imports and GC disable after line 29 (`log.Println("ivf index ready")`):

```go
import (
	"math"
	"runtime"
	"runtime/debug"
	// ... existing imports
)

// After idx load and before mccRisk load:
runtime.GC()
debug.SetGCPercent(-1)
debug.SetMemoryLimit(math.MaxInt64)
```

**Step 2: Verify build**

Run: `go build ./cmd/server`

**Step 3: Commit**

---

### Task 2: Hand-written JSON scanner

**Files:**
- Create: `internal/handler/json.go`
- Modify: `internal/handler/fraud.go`
- Modify: `internal/vectorize/vectorize.go`

**Step 1: Change vectorize.Payload to use string timestamps instead of time.Time**

Modify `internal/vectorize/vectorize.go`:
- Change `Transaction.RequestedAt` from `time.Time` to `string`
- Change `LastTransaction.Timestamp` from `time.Time` to `string`
- In `Vectorize()`, use `parseHourWeekday(p.Transaction.RequestedAt)` instead of `.UTC().Hour()` / `.UTC().Weekday()`
- For `LastTransaction`, parse minutes difference from two RFC3339 strings directly

The updated types:

```go
type Transaction struct {
	Amount       float32
	Installments int
	RequestedAt  string
}

type LastTransaction struct {
	Timestamp     string
	KmFromCurrent float32
}
```

The updated Vectorize function should call `parseHourWeekday(p.Transaction.RequestedAt)` for dims 3,4 and use a new `parseMinutesBetween(later, earlier string) float32` for dim 5.

Add `parseMinutesBetween`:

```go
func parseMinutesBetween(later, earlier string) float32 {
	if len(later) < 19 || len(earlier) < 19 {
		return 0
	}
	ly := int(later[0]-'0')*1000 + int(later[1]-'0')*100 + int(later[2]-'0')*10 + int(later[3]-'0')
	lm := int(later[5]-'0')*10 + int(later[6]-'0')
	ld := int(later[8]-'0')*10 + int(later[9]-'0')
	lh := int(later[11]-'0')*10 + int(later[12]-'0')
	lmin := int(later[14]-'0')*10 + int(later[15]-'0')
	ls := int(later[17]-'0')*10 + int(later[18]-'0')

	ey := int(earlier[0]-'0')*1000 + int(earlier[1]-'0')*100 + int(earlier[2]-'0')*10 + int(earlier[3]-'0')
	em := int(earlier[5]-'0')*10 + int(earlier[6]-'0')
	ed := int(earlier[8]-'0')*10 + int(earlier[9]-'0')
	eh := int(earlier[11]-'0')*10 + int(earlier[12]-'0')
	emin := int(earlier[14]-'0')*10 + int(earlier[15]-'0')
	es := int(earlier[17]-'0')*10 + int(earlier[18]-'0')

	lTotal := toUnixDays(ly, lm, ld)*1440 + lh*60 + lmin + ls/60
	eTotal := toUnixDays(ey, em, ed)*1440 + eh*60 + emin + es/60

	diff := lTotal - eTotal
	if diff < 0 {
		diff = 0
	}
	return float32(diff)
}

func toUnixDays(y, m, d int) int {
	if m <= 2 {
		y--
		m += 12
	}
	return 365*y + y/4 - y/100 + y/400 + (153*(m-3)+2)/5 + d - 719469
}
```

**Step 2: Create hand-written JSON scanner**

Create `internal/handler/json.go` with a scanner struct and `ParsePayload(buf []byte, p *vectorize.Payload) error` function.

The scanner has: `buf []byte`, `pos int`, and methods: `done()`, `peek()`, `advance()`, `skipWS()`, `expect()`, `expectLiteral()`, `readString()`, `readFloat()`, `readInt()`, `readBool()`, `skipValue()`.

`ParsePayload` parses the known JSON shape directly into `vectorize.Payload`. For each top-level key, switch on `string(key)`:
- `"id"`: skip (not used downstream)
- `"transaction"`: parse nested object with `amount` (float), `installments` (int), `requested_at` (string)
- `"customer"`: parse `avg_amount` (float), `tx_count_24h` (int), `known_merchants` (string array)
- `"merchant"`: parse `id` (string), `mcc` (string), `avg_amount` (float)
- `"terminal"`: parse `is_online` (bool), `card_present` (bool), `km_from_home` (float)
- `"last_transaction"`: if `null` set nil, else parse `timestamp` (string), `km_from_current` (float)

Float parsing: manual mantissa+fraction+exponent, zero allocation.
Int parsing: digit-by-digit, zero allocation.
String parsing: return `[]byte` alias into buf, caller copies via `string(...)` only when needed.

Reference implementation: see `/tmp/rinha-backend-26/internal/http/json.go` — adapt the same approach.

**Step 3: Update fraud.go handler**

Replace `gojson.Unmarshal` call with `ParsePayload(buf.Bytes(), &p)` where `p` is a `vectorize.Payload` directly.
Remove the `fraudRequest` struct entirely.
Remove `time.Parse` calls.
Remove the `gojson` import.

The updated `FraudScore` method:

```go
func (h *Handler) FraudScore(w http.ResponseWriter, r *http.Request) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.ReadFrom(r.Body)

	var p vectorize.Payload
	if err := ParsePayload(buf.Bytes(), &p); err != nil {
		bufPool.Put(buf)
		w.Header().Set("Content-Type", "application/json")
		w.Write(precomputed[0])
		return
	}
	bufPool.Put(buf)

	fraudCount := h.svc.FraudCount(p)
	w.Header().Set("Content-Type", "application/json")
	w.Write(precomputed[fraudCount])
}
```

Note: on parse error, return `approved: true, fraud_score: 0` instead of HTTP 400 (error weight 5 > FP weight 1).

**Step 4: Remove goccy/go-json dependency**

Run:
```bash
go mod tidy
```

If `goccy/go-json` is no longer imported anywhere, it will be removed from `go.mod`/`go.sum`.

**Step 5: Run tests**

Run: `go test ./internal/handler/ ./internal/vectorize/ ./internal/service/`

Fix any test failures from the type changes (string vs time.Time).

**Step 6: Commit**

---

### Task 3: Bbox pruning in IVF search

**Files:**
- Modify: `internal/ivf/scan.go`
- Modify: `internal/ivf/index.go`

**Step 1: Add bbox check before scanning a cluster**

In `internal/ivf/scan.go`, modify `scanCluster` to check bbox distance before scanning:

```go
func (idx *Index) scanCluster(query [Dim]int16, cluster int, state *ivfSearchState) {
	if len(idx.ivf.bboxMin) >= (cluster+1)*Dim {
		if idx.bboxDist(query, cluster, state.bestDist[4]) > state.bestDist[4] {
			return
		}
	}
	if useIVFAVX2 {
		idx.scanBlocksAVX2(query, cluster, state)
	} else {
		idx.scanBlocksScalar(query, cluster, state)
	}
}
```

This skips entire clusters whose bounding box is farther than the current 5th-best distance. The `bboxDist` function already exists in `repair.go`.

**Step 2: Run tests**

Run: `go test ./internal/ivf/`

**Step 3: Commit**

---

### Task 4: Custom HTTP/1.1 server

**Files:**
- Create: `internal/server/server.go`
- Modify: `cmd/server/main.go`
- Modify: `internal/handler/fraud.go`

**Step 1: Create custom HTTP server**

Create `internal/server/server.go`:

```go
package server

import (
	"net"
	"os"
	"sync"
)

type Handler func(path []byte, body []byte) []byte

type Server struct {
	ln      net.Listener
	handler Handler
}

const maxRequestSize = 8 * 1024

var readBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 4096)
		return &b
	},
}

func Listen(socketPath string, handler Handler) (*Server, error) {
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(socketPath, 0o666)
	srv := &Server{ln: ln, handler: handler}
	go srv.acceptLoop()
	return srv, nil
}

func ListenTCP(addr string, handler Handler) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := &Server{ln: ln, handler: handler}
	go srv.acceptLoop()
	return srv, nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	bufRef := readBufPool.Get().(*[]byte)
	defer readBufPool.Put(bufRef)
	buf := *bufRef
	used := 0

	for {
		var headEnd int
		for {
			if used >= len(buf) {
				if used >= maxRequestSize {
					return
				}
				newBuf := make([]byte, len(buf)*2)
				copy(newBuf, buf[:used])
				buf = newBuf
				*bufRef = buf
			}
			n, err := conn.Read(buf[used:])
			if n > 0 {
				used += n
				if idx := indexHeaderEnd(buf[:used]); idx >= 0 {
					headEnd = idx + 4
					break
				}
			}
			if err != nil {
				return
			}
		}

		path, contentLen := parseRequestLine(buf[:headEnd])
		bodyEnd := headEnd + contentLen
		for used < bodyEnd {
			if used >= len(buf) {
				newBuf := make([]byte, len(buf)*2)
				copy(newBuf, buf[:used])
				buf = newBuf
				*bufRef = buf
			}
			n, err := conn.Read(buf[used:])
			if n > 0 {
				used += n
			}
			if err != nil {
				return
			}
		}

		resp := s.handler(path, buf[headEnd:bodyEnd])
		if _, err := conn.Write(resp); err != nil {
			return
		}

		copy(buf, buf[bodyEnd:used])
		used -= bodyEnd
	}
}

func indexHeaderEnd(b []byte) int {
	for i := 0; i+3 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' && b[i+2] == '\r' && b[i+3] == '\n' {
			return i
		}
	}
	return -1
}

func parseRequestLine(buf []byte) (path []byte, contentLen int) {
	i := 0
	for i < len(buf) && buf[i] != ' ' {
		i++
	}
	i++
	pathStart := i
	for i < len(buf) && buf[i] != ' ' {
		i++
	}
	path = buf[pathStart:i]

	cl := findContentLength(buf)
	return path, cl
}

func findContentLength(buf []byte) int {
	for i := 0; i+16 < len(buf); i++ {
		if (buf[i] == 'C' || buf[i] == 'c') && isContentLength(buf[i:]) {
			j := i + 16
			for j < len(buf) && buf[j] == ' ' {
				j++
			}
			n := 0
			for j < len(buf) && buf[j] >= '0' && buf[j] <= '9' {
				n = n*10 + int(buf[j]-'0')
				j++
			}
			return n
		}
	}
	return 0
}

func isContentLength(b []byte) bool {
	const name = "content-length: "
	if len(b) < len(name) {
		return false
	}
	for i := 0; i < 15; i++ {
		c := b[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != name[i] {
			return false
		}
	}
	return true
}
```

**Step 2: Create pre-rendered HTTP responses**

Add to `internal/server/server.go` or a separate file `internal/server/response.go`:

```go
package server

import "fmt"

var (
	RespReady [7][]byte
)

func init() {
	bodies := [6]string{
		`{"approved":true,"fraud_score":0.0}`,
		`{"approved":true,"fraud_score":0.2}`,
		`{"approved":true,"fraud_score":0.4}`,
		`{"approved":false,"fraud_score":0.6}`,
		`{"approved":false,"fraud_score":0.8}`,
		`{"approved":false,"fraud_score":1.0}`,
	}
	for i, body := range bodies {
		RespReady[i] = []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	}
	RespReady[6] = []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
}
```

**Step 3: Update cmd/server/main.go**

Replace `net/http` with custom server. The handler function receives `(path, body)` and returns `[]byte`:

```go
handler := func(path []byte, body []byte) []byte {
	if len(path) == 6 && path[1] == 'r' { // /ready
		return server.RespReady[6]
	}
	var p vectorize.Payload
	if err := handler.ParsePayload(body, &p); err != nil {
		return server.RespReady[0]
	}
	fraudCount := svc.FraudCount(p)
	return server.RespReady[fraudCount]
}
```

Use `server.Listen(socketPath, handler)` or `server.ListenTCP(":"+port, handler)`.

Keep `select{}` at the end to block main goroutine.

**Step 4: Move ParsePayload to its own package or keep in handler**

`ParsePayload` should be accessible from `cmd/server/main.go`. Either:
- Move it to `internal/handler/` and export it, or
- Move it to its own package `internal/parse/`

Simplest: keep in `internal/handler/` as an exported function.

**Step 5: Run tests and verify build**

Run: `go build ./cmd/server && go test ./...`

**Step 6: Commit**

---

### Task 5: Block prefetching in assembly

**Files:**
- Modify: `internal/ivf/dist_amd64.s`

**Step 1: Add PREFETCHT0 to block32 processing**

In `quantizedBlock32DistancesAVX2`, add a `PREFETCHT0` instruction before each `BLOCK8` macro to prefetch the next block's data. Each block is 224 bytes (14 dims × 8 lanes × 2 bytes).

After `BLOCK8(0, 0)`, before `BLOCK8(224, 64)`, add:
```asm
PREFETCHT0 448(BX)
```

After `BLOCK8(224, 64)`, before `BLOCK8(448, 128)`, add:
```asm
PREFETCHT0 672(BX)
```

After `BLOCK8(448, 128)`, before `BLOCK8(672, 192)`, add:
```asm
PREFETCHT0 896(BX)   // prefetch next 32-vector group
```

**Step 2: Add prefetch in scanBlocksAVX2**

In `internal/ivf/scan.go`, add a prefetch hint by touching the next block pointer before calling the AVX2 distance function. This is done in Go since we can't easily add cross-function prefetch in assembly:

In the `scanBlocksAVX2` loop, before calling `quantizedBlock32DistancesAVX2`, prefetch the next block:

```go
for ; block+4 <= blockEnd; block += 4 {
	blockPtr := unsafe.Add(blocks, block*blockStride*2)
	if block+8 <= blockEnd {
		nextPtr := unsafe.Add(blocks, (block+4)*blockStride*2)
		_ = *(*int16)(nextPtr)
	}
	quantizedBlock32DistancesAVX2(&query[0], blockPtr, &dist32[0])
	// ... rest of loop
```

**Step 3: Verify build on amd64**

Run: `GOOS=linux GOARCH=amd64 go build ./cmd/server`

**Step 4: Commit**

---

## Execution Order

1. Task 1 (GC disable) — trivial, immediate win
2. Task 3 (bbox pruning) — small change, big search speedup
3. Task 5 (prefetch) — small assembly change
4. Task 2 (JSON scanner) — medium effort, eliminates parse overhead
5. Task 4 (custom HTTP) — largest change, eliminates stdlib overhead
