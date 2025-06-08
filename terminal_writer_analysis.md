# TerminalWriter Performance Analysis

## Identified Allocations

### 1. sync.Pool Usage (Lines 52-56, 90-99)
- **Current**: Pool allocates `make([]byte, 0, 512)` in New function
- **Issue**: This is good - reuses buffers to avoid allocations
- **Status**: ✅ Already optimized

### 2. String Conversions from []byte
Multiple string conversions that allocate:
- **Line 85**: `msg = string(b[pos : pos+msgLen])` - Allocates for message
- **Line 146**: `key = string(b[pos : pos+keyLen])` - Allocates for field keys
- **Line 259**: `string(b[2 : 2+strLen])` in decodeFieldValue - Allocates for string fields
- **Line 320**: `return string(b)` in escapeString - Allocates when escaping

### 3. fmt.Sprintf Calls
Multiple fmt.Sprintf calls that always allocate:
- **Line 228**: `fmt.Sprintf("%d", int64(v))` - For int values
- **Line 240**: `fmt.Sprintf("%d", v)` - For uint values
- **Line 246**: `fmt.Sprintf("%.3f", f)` - For float32 values
- **Line 253**: `fmt.Sprintf("%.3f", f)` - For float64 values
- **Line 266**: `fmt.Sprintf("%x", b[2:2+dataLen])` - For bytes values
- **Line 314**: `fmt.Sprintf("%q", s)` - Fallback in escapeString

### 4. Append Operations
Multiple append operations that may cause reallocation:
- **Lines 106-111**: Appending color codes and level string
- **Lines 115-117**: Appending timestamp
- **Line 120**: Appending message
- **Lines 125-128**: Padding spaces in loop (inefficient)
- **Lines 157-165**: Appending field keys with colors
- **Line 169**: Appending field values
- **Line 174**: Appending newline

### 5. escapeString Function (Lines 289-321)
- **Good**: Uses stack buffer `var buf [128]byte` for small strings
- **Bad**: Falls back to `fmt.Sprintf("%q", s)` which allocates
- **Bad**: Final `return string(b)` allocates a new string

### 6. Color String Concatenations
- Color constants are good (compile-time constants)
- But appending them repeatedly can cause buffer growth

## Most Critical Performance Issues

1. **fmt.Sprintf for numeric conversions** - These are the worst offenders as they:
   - Allocate memory for the formatted string
   - Use reflection internally
   - Are called for every numeric field

2. **String conversions from []byte** - Each conversion allocates a new string

3. **Inefficient padding loop** (lines 125-128) - Appends one space at a time

4. **escapeString fallback** - Falls back to fmt.Sprintf for complex strings

## Optimization Suggestions

### 1. Replace fmt.Sprintf with Custom Formatters
```go
// Use strconv.AppendInt instead of fmt.Sprintf
func appendInt(dst []byte, v int64) []byte {
    return strconv.AppendInt(dst, v, 10)
}

// Use strconv.AppendUint
func appendUint(dst []byte, v uint64) []byte {
    return strconv.AppendUint(dst, v, 10)
}

// Use strconv.AppendFloat
func appendFloat32(dst []byte, v float32) []byte {
    return strconv.AppendFloat(dst, float64(v), 'f', 3, 32)
}

func appendFloat64(dst []byte, v float64) []byte {
    return strconv.AppendFloat(dst, v, 'f', 3, 64)
}
```

### 2. Avoid String Conversions
```go
// Instead of converting to string, work with []byte directly
// Store field keys as []byte in the logger to avoid conversions
// Or use unsafe string conversion for read-only strings:
func unsafeString(b []byte) string {
    return *(*string)(unsafe.Pointer(&b))
}
```

### 3. Optimize Padding
```go
// Pre-allocate padding spaces
var spaces = []byte("                                        ") // 40 spaces

// Use single append instead of loop
if padding > 0 && padding <= len(spaces) {
    buf = append(buf, spaces[:padding]...)
}
```

### 4. Optimize escapeString
```go
// Increase stack buffer size
var buf [256]byte

// Use custom hex encoding instead of fmt.Sprintf
const hexTable = "0123456789abcdef"
func appendHex(dst []byte, src []byte) []byte {
    for _, b := range src {
        dst = append(dst, hexTable[b>>4], hexTable[b&0x0f])
    }
    return dst
}
```

### 5. Pre-size Buffer Based on Estimated Output
```go
// Estimate output size to avoid reallocations
estimatedSize := 22 + // header
    5 + // level string
    1 + len(termTimeFormat) + 3 + // timestamp with brackets
    len(msg) + // message
    termMsgJust + // potential padding
    fieldCount * 20 + // estimate for fields
    1 // newline

buf := w.buf.Get().([]byte)
if cap(buf) < estimatedSize {
    buf = make([]byte, 0, estimatedSize)
}
```

### 6. Use Lookup Tables for Common Values
```go
// Pre-format common values
var (
    boolTrue  = []byte("true")
    boolFalse = []byte("false")
    levelStrings = map[Level][]byte{
        LevelDebug: []byte("DEBUG"),
        LevelInfo:  []byte("INFO "),
        // etc...
    }
)
```

### 7. Batch Color Code Appends
```go
// Instead of multiple appends, combine color codes
coloredLevel := []byte(color + levelStr + colorReset)
buf = append(buf, coloredLevel...)
```

## Priority Optimization Plan

1. **High Priority**: Replace all fmt.Sprintf calls with strconv.Append* functions
2. **High Priority**: Optimize numeric field formatting in decodeFieldValue
3. **Medium Priority**: Use unsafe string conversions for read-only data
4. **Medium Priority**: Fix padding loop inefficiency
5. **Low Priority**: Optimize escapeString for edge cases
6. **Low Priority**: Pre-size buffers based on content

## Expected Performance Improvements

- **Eliminating fmt.Sprintf**: 50-70% reduction in allocations for numeric fields
- **Avoiding string conversions**: 20-30% reduction in allocations
- **Optimized padding**: Minor improvement but better for high-volume logs
- **Overall**: Could achieve near-zero allocations per log entry with these optimizations