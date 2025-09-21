# uniqid

A **super-fast**, **collision-free**, and **time-sortable** unique ID generator for Go.  
Inspired by **YouTube video IDs** and **Twitter Snowflake**, but optimized for simplicity and speed.

[![Go Reference](https://pkg.go.dev/badge/github.com/aprakasa/uniqid.svg)](https://pkg.go.dev/github.com/aprakasa/uniqid)
![Go](https://img.shields.io/badge/go-%3E%3D1.21-blue)
![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)
[![Go CI](https://github.com/aprakasa/uniqid/actions/workflows/go.yml/badge.svg)](https://github.com/aprakasa/uniqid/actions/workflows/go.yml)


## ✨ Features

- 🔑 **11-character, URL-safe IDs** using alphabet `A–Z, a–z, 0–9, - _`
- ⚡ **Blazing fast**: ~35ns/op (~34M IDs/sec/core)
- 📈 **Monotonic & time-sortable** → always ordered by creation time
- 🧑‍🤝‍🧑 **Shard-aware**: supports up to 1024 nodes
- 🧵 **Thread-safe**: safe for concurrent goroutines
- ✅ **100% test coverage** with extensive edge-case testing

## 💡 Use Cases

`uniqid` is designed for scenarios where you need **short, fast, and ordered unique IDs**.

### 🔑 Database Primary Keys
- Replace integer autoincrement with globally unique, time-sortable IDs.
- Reduce contention in distributed systems (safe across shards/nodes).
- Shorter than UUID/ULID → smaller index size → faster queries.

### 🌐 Distributed Systems
- Generate unique IDs across 1024 nodes without coordination.
- Guaranteed monotonic ordering within each node.
- Perfect for message brokers, logs, and event streams.

### 🔗 URL Shortener & Public IDs
- 11 characters only → ideal for short URLs, invite codes, QR codes.
- URL-safe alphabet (`A–Z, a–z, 0–9, - _`).

### 📱 Mobile & Edge Devices
- Fast enough to run on low-powered devices (35ns/op).
- No external dependency, works offline.

### 📊 Analytics & Logging
- Time-sortable IDs make it easy to analyze data by generation time.
- Collision-free under heavy load (~34M IDs/sec/core).


## 📦 Install
```bash
go get github.com/aprakasa/uniqid
```

## 🚀 Quick Start

For most use cases, you can use the `Gen` function to get a unique ID directly.

```go
package main

import (
	"fmt"
	"github.com/aprakasa/uniqid"
	"log"
)

func main() {
	// Get a unique ID with the default configuration
	id, err := uniqid.Gen()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(id)

	// Or, generate an ID with a custom shard ID
	id, err = uniqid.Gen(&uniqid.Config{ShardID: 2})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(id)
}
```

## 📝 Advanced Usage

If you need to generate many IDs in a tight loop, it's more performant to create a generator instance once and reuse it.

```go
package main

import (
    "fmt"
    "github.com/aprakasa/uniqid"
)

func main() {
    // Create a generator with ShardID = 1
    gen, _ := uniqid.New(&uniqid.Config{ShardID: 1})

    // Generate unique IDs
    for i := 0; i < 5; i++ {
        fmt.Println(gen.Next())
    }
    // Example output:
    // Ab3Xyz0LmN_
    // Ab3Xyz0LmN0
    // Ab3Xyz0LmN1
}
```

## 📖 Documentation

Full API reference is available on [pkg.go.dev](https://pkg.go.dev/github.com/aprakasa/uniqid).

- [Gen](https://pkg.go.dev/github.com/aprakasa/uniqid#Gen)  
  Generate a new unique ID with an optional config (recommended for simplicity).

- [New](https://pkg.go.dev/github.com/aprakasa/uniqid#New)  
  Create a new ID generator with optional configuration (shard ID, custom epoch).

- [Generator](https://pkg.go.dev/github.com/aprakasa/uniqid#Generator)  
  A safe, concurrent generator for unique IDs.

- [Generator.Next](https://pkg.go.dev/github.com/aprakasa/uniqid#Generator.Next)  
  Generate a new 11-character unique ID.


## 📊 Benchmark
```bash
cpu: 11th Gen Intel(R) Core(TM) i9-11900H @ 2.50GHz
BenchmarkUniqID-16     34000000    35.8 ns/op     16 B/op   1 allocs/op
```

| Generator  | ns/op ↓  | allocs/op ↓ | Bytes/op ↓ |
| ---------- | -------- | ----------- | ---------- |
| **uniqid** | \~36 ns  | 1           | 16         |

## 🧪 Coverage
```bash
ok  	github.com/aprakasa/uniqid	1.598s	coverage: 100.0% of statements
```

## 💡 Why uniqid?

- ✅ Safer & shorter than UUID (11 chars vs 36 chars)
- ✅ Faster than ULID or KSUID
- ✅ Easy to sort by time
- ✅ Perfect for databases, distributed systems, and URLs

## 📜 License

MIT License