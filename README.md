# zlog - The Fastest Zero-Allocation Logging Library for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/semihalev/zlog.svg)](https://pkg.go.dev/github.com/semihalev/zlog)
[![Go Report Card](https://goreportcard.com/badge/github.com/semihalev/zlog)](https://goreportcard.com/report/github.com/semihalev/zlog)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-83.7%25-brightgreen.svg)](https://github.com/semihalev/zlog)

The world's fastest logging library for Go with **true zero allocations**, achieving an incredible **7.28 nanoseconds** per log operation. Benchmarks prove it's **5.2x faster than Zap** and **2.5x faster than Zerolog**. Built from the ground up for Go 1.23+ with a focus on extreme performance.

## 🚀 Performance

```
BenchmarkNanoLogger-10              147687733      7.425 ns/op      0 B/op    0 allocs/op
BenchmarkUltimateLogger-10           53423659     18.84 ns/op      0 B/op    0 allocs/op
BenchmarkStructuredLogger-10         20530777     59.08 ns/op      0 B/op    0 allocs/op
BenchmarkZeroAllocLogger-10          19329052     56.89 ns/op    256 B/op    1 allocs/op
BenchmarkUltralog-10                 17233688     70.05 ns/op    256 B/op    1 allocs/op
BenchmarkDisabledDebug-10          1000000000      1.281 ns/op      0 B/op    0 allocs/op
```

## ✨ Features

- **True Zero Allocations**: Multiple loggers achieve 0 B/op, 0 allocs/op
- **Extreme Performance**: 7.425 ns/op - faster than just copying a string!
- **Lock-Free Design**: Uses atomic operations for thread-safe, contention-free logging
- **Cache-Line Aligned**: Structures optimized for CPU cache efficiency (64 bytes)
- **Beautiful Terminal Output**: Colored, formatted output for development
- **Structured Logging**: Type-safe fields without interface boxing
- **Multiple Writers**: Terminal, Memory-mapped files, Async ring buffer
- **Binary Format**: Compact binary encoding for maximum throughput
- **Go 1.23+ Optimized**: Built using the latest Go features and runtime optimizations

## 📦 Installation

```bash
go get github.com/semihalev/zlog
```

Requires Go 1.23 or later.

## 🎯 Quick Start

### Global Logger (Easy Migration from v0.x)

```go
package main

import "github.com/semihalev/zlog"

func main() {
    // v0.x compatible - accepts any key-value pairs
    zlog.Info("Application starting")
    zlog.Info("User logged in", "username", "john", "user_id", 12345)
    zlog.Error("Connection failed", "host", "localhost", "port", 5432)
    
    // Configure global logger
    zlog.SetLevel(zlog.LevelWarn)  // Only Warn, Error, Fatal will be logged
    
    // Or use typed fields for better performance (0 allocations)
    zlog.Error("Database error",
        zlog.String("host", "localhost"),
        zlog.Int("port", 5432),
        zlog.String("error", "connection refused"))
}
```

The global logger intelligently handles both styles:
- **Any values**: `zlog.Info("msg", "key", value, ...)` - v0.x compatible
- **Typed fields**: `zlog.Info("msg", zlog.String("key", "val"))` - Zero allocations

### Basic Logging

```go
package main

import "github.com/semihalev/zlog"

func main() {
    // Create logger instance with beautiful terminal output
    logger := zlog.New()
    logger.SetWriter(zlog.StdoutTerminal())
    
    // Basic logging
    logger.Debug("Application starting...")
    logger.Info("Server initialized successfully")
    logger.Warn("Configuration not found, using defaults")
    logger.Error("Failed to connect to database")
    logger.Fatal("Critical error, shutting down") // Exits with code 1
}
```

### Structured Logging

```go
// Create structured logger with zero allocations
logger := zlog.NewStructured()
logger.SetWriter(zlog.StdoutTerminal())

// Log with typed fields - 0 allocations thanks to buffer pool!
logger.Info("User logged in",
    zlog.String("username", "john_doe"),
    zlog.Int("user_id", 12345),
    zlog.Bool("admin", true),
    zlog.Float64("session_time", 30.5))

logger.Error("Request failed",
    zlog.String("method", "POST"),
    zlog.String("path", "/api/users"),
    zlog.Int("status", 500),
    zlog.Uint64("duration_ns", 1234567))
```

### Zero-Allocation Logging

```go
// For absolute maximum performance
logger := zlog.NewUltimateLogger()

// 18.84 ns/op with true zero allocations
logger.Info("Ultra-fast logging")
logger.Debug("This is incredibly fast")

// Or use NanoLogger for 7.425 ns/op!
nano := zlog.NewNanoLogger(nil)
var buf [256]byte
nano.Info(buf[:], "Fastest possible logging")
```

## 🏗️ Architecture

### Logger Types

1. **Logger** - Basic ultra-fast logger (73 ns/op)
   - Simple and fast for basic logging needs
   - Binary format output
   - Configurable log levels

2. **StructuredLogger** - Type-safe structured logging (59 ns/op, 0 allocs)
   - Typed fields without interface boxing
   - Zero-allocation field encoding with buffer pool
   - Perfect for production systems

3. **ZeroAllocLogger** - Uses ZeroWriter interface (56 ns/op)
   - Stack-allocated buffers only
   - Special interface to prevent heap escapes
   - Ideal for hot paths

4. **UltimateLogger** - Direct memory writes (18.84 ns/op, 0 allocs)
   - Lock-free ring buffer
   - Memory-mapped output
   - For extreme throughput requirements

5. **NanoLogger** - The absolute fastest (7.425 ns/op, 0 allocs)
   - Caller provides buffer
   - Minimal overhead
   - For the most demanding applications

### Writers

- **StdoutTerminal/StderrTerminal** - Beautiful colored terminal output
- **StdoutWriter/StderrWriter** - Basic standard output
- **DiscardWriter** - Discard all output (benchmarking)
- **MMapWriter** - Memory-mapped files for zero-syscall writes
- **AsyncWriter** - Lock-free ring buffer for async logging
- **Custom Writers** - Implement the simple `Writer` interface

## 🎨 Terminal Output

The terminal writer provides beautiful, colored output:

```
DEBUG[01-02|15:04:05] Application starting...
INFO [01-02|15:04:05] Server initialized successfully
WARN [01-02|15:04:05] Config not found, using defaults
ERROR[01-02|15:04:05] Database connection failed         error="timeout" retry=3
```

Colors:
- `DEBUG` - Cyan
- `INFO` - Green  
- `WARN` - Yellow
- `ERROR` - Red
- `FATAL` - Magenta

## 🔧 Advanced Usage

### Memory-Mapped File Logging

```go
// Create memory-mapped file writer for zero-syscall logging
mmap, err := zlog.NewMMapWriter("/var/log/app.log", 100*1024*1024) // 100MB
if err != nil {
    panic(err)
}
defer mmap.Close()

logger := zlog.New()
logger.SetWriter(mmap.Writer())
```

### Asynchronous Logging

```go
// Create async writer with ring buffer
async := zlog.NewAsyncWriter(zlog.StdoutTerminal(), 8192)
defer async.Close()

logger := zlog.New()
logger.SetWriter(async.Writer())

// Logs are written asynchronously
logger.Info("This won't block")
```

### Custom Writers

```go
// Implement your own writer
type CustomWriter struct{}

func (w CustomWriter) Write(p []byte) error {
    // Your custom logic here
    return nil
}

logger := zlog.New()
logger.SetWriter(CustomWriter{}.Write)
```

### Log Levels

```go
logger := zlog.New()

// Set minimum log level
logger.SetLevel(zlog.LevelWarn) // Only Warn, Error, Fatal will be logged

// Check current level
if logger.GetLevel() <= zlog.LevelDebug {
    // Expensive debug operation
}
```

### Field Types

All field types are available with zero allocations:

```go
logger.Info("event",
    zlog.String("name", "John"),
    zlog.Int("age", 30),
    zlog.Int64("id", 123456789),
    zlog.Uint("count", 42),
    zlog.Uint64("total", 9999999),
    zlog.Float32("score", 98.5),
    zlog.Float64("precision", 3.14159265359),
    zlog.Bool("active", true),
    zlog.Bytes("data", []byte{0x01, 0x02, 0x03}))
```

## 🏆 Benchmarks

Run on Apple M4:

```bash
$ go test -bench=. -benchmem

BenchmarkNanoLogger-10              147687733      7.425 ns/op      0 B/op    0 allocs/op
BenchmarkNanoLoggerWithOutput-10    155354494      7.958 ns/op      0 B/op    0 allocs/op
BenchmarkUltimateLogger-10           53423659     18.84 ns/op       0 B/op    0 allocs/op
BenchmarkUltimateLoggerParallel-10   18684834     65.53 ns/op       0 B/op    0 allocs/op
BenchmarkStructuredLogger-10         20530777     59.08 ns/op       0 B/op    0 allocs/op
BenchmarkStructuredLogger5Fields-10  13193077     80.72 ns/op       0 B/op    0 allocs/op
BenchmarkStructuredLogger10Fields-10 10458685    110.7 ns/op        0 B/op    0 allocs/op
BenchmarkZeroAllocLogger-10          19329052     56.89 ns/op     256 B/op    1 allocs/op
BenchmarkUltralog-10                 17233688     70.05 ns/op     256 B/op    1 allocs/op
BenchmarkAsyncWriter-10               7035481    184.6 ns/op      544 B/op    2 allocs/op
BenchmarkMMapWriter-10               11997840     94.14 ns/op     256 B/op    1 allocs/op
```

## 📊 Comparison with Other Loggers

Comprehensive benchmarks on Apple M4 with Go 1.23 (structured logging with 5 fields):

| Logger | ns/op | B/op | allocs/op | vs zlog |
|--------|------:|-----:|----------:|--------:|
| **zlog (Nano)** | **7.28** | **0** | **0** | **1.0x** |
| **zlog (Ultimate)** | **17.04** | **0** | **0** | **2.3x** |
| **zlog (Typed)** | **66.40** | **0** | **0** | **baseline** |
| **zlog (v0.x compat)** | **74.24** | **0** | **0** | **1.1x** |
| Zerolog | 165.2 | 0 | 0 | 2.5x slower |
| Zap | 346.0 | 320 | 1 | 5.2x slower |
| slog (stdlib) | 602.5 | 120 | 3 | 9.1x slower |
| Logrus | 1455 | 1416 | 25 | 21.9x slower |

**Key Achievement**: zlog is **5.2x faster than Zap** and **2.5x faster than Zerolog** while maintaining zero allocations!

## 🔬 How It Works

### Zero Allocations

1. **Stack-allocated buffers**: All temporary buffers are allocated on the stack
2. **Buffer pools**: StructuredLogger uses sync.Pool to eliminate allocations
3. **Direct memory writes**: Use `unsafe` for direct memory manipulation
4. **No interface boxing**: Typed fields avoid `interface{}` allocations
5. **Binary format**: Compact encoding reduces memory usage
6. **Lock-free atomics**: Avoid mutex allocations

### Performance Techniques

- **Cache-line alignment**: 64-byte aligned structures
- **Atomic operations**: Lock-free level checks and updates
- **Memory-mapped I/O**: Zero-syscall writes to files
- **Ring buffers**: Lock-free async logging
- **Inlining**: Critical paths are inlined by the compiler
- **Direct syscalls**: Using Go's runtime linkname for nanotime()

## 🔄 Migration from v0.x

The new version provides full backward compatibility while offering better performance:

### v0.x Style (Still Works!)
```go
// Old code continues to work
zlog.Info("User action", "username", "john", "action", "login", "ip", "192.168.1.1")
zlog.Error("Database error", "error", err, "query", sqlQuery)
```

### New Style (Better Performance)
```go
// Use typed fields for zero allocations
zlog.Info("User action",
    zlog.String("username", "john"),
    zlog.String("action", "login"),
    zlog.String("ip", "192.168.1.1"))
```

### Performance Comparison
- **v0.x style with common types**: 57.62 ns/op, 0 allocations
- **Typed fields**: 59.08 ns/op, 0 allocations
- Both are significantly faster than other loggers!

## 🧪 Testing

The library has **85.3% test coverage** and passes all tests including race detection:

```bash
$ go test -race ./...
ok  github.com/semihalev/zlog  1.886s

$ go test -cover ./...
ok  github.com/semihalev/zlog  0.520s  coverage: 85.3% of statements
```

## 📝 Examples

### High-Performance HTTP Server

```go
// Create the fastest possible logger for request logging
logger := zlog.NewUltimateLogger()

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    
    // Your handler logic here
    
    // Log with 18.84 ns overhead
    logger.Info(fmt.Sprintf("%s %s %d %dns", 
        r.Method, r.URL.Path, 200, time.Since(start).Nanoseconds()))
})
```

### Production Service

```go
// Structured logger for production with terminal output
logger := zlog.NewStructured()
logger.SetWriter(zlog.StdoutTerminal())

// Log with rich context
logger.Info("service started",
    zlog.String("version", "1.0.0"),
    zlog.String("env", "production"),
    zlog.Int("pid", os.Getpid()),
    zlog.String("node", hostname))

// Log errors with context
logger.Error("database query failed",
    zlog.String("query", query),
    zlog.String("error", err.Error()),
    zlog.Float64("duration_ms", duration.Seconds()*1000))
```

See more examples in [example_test.go](example_test.go) and [demo/main.go](demo/main.go).

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with ❤️ for the Go community
- Inspired by the need for truly zero-allocation logging
- Special thanks to all contributors

---

**Note**: This logger uses `unsafe` operations for maximum performance. While thoroughly tested, please evaluate if this fits your risk tolerance for production systems.
