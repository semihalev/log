package zlog

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

// LogfmtWriter decodes binary log format and outputs logfmt format
// logfmt is human-readable and machine-parseable: key=value pairs
type LogfmtWriter struct {
	out io.Writer
	buf sync.Pool
}

// NewLogfmtWriter creates a new logfmt writer
func NewLogfmtWriter(out io.Writer) *LogfmtWriter {
	return &LogfmtWriter{
		out: out,
		buf: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 512)
			},
		},
	}
}

// Write decodes binary log and outputs logfmt format
func (w *LogfmtWriter) Write(b []byte) (int, error) {
	if len(b) < 22 { // Minimum header size
		return 0, fmt.Errorf("invalid log entry: too short")
	}

	// Decode binary header
	magic := binary.LittleEndian.Uint32(b[0:4])
	if magic != MagicHeader {
		return 0, fmt.Errorf("invalid magic header")
	}

	// version := b[4]
	level := Level(b[5])
	// seq := binary.LittleEndian.Uint64(b[6:14])
	timestamp := binary.LittleEndian.Uint64(b[14:22])

	pos := 22

	// Get message
	var msg string
	if pos < len(b) {
		msgLen := int(b[pos])
		pos++
		if pos+msgLen <= len(b) {
			msg = string(b[pos : pos+msgLen])
			pos += msgLen
		}
	}

	// Get buffer from pool
	bufInterface := w.buf.Get()
	buf := bufInterface.([]byte)
	buf = buf[:0]
	defer func() {
		w.buf.Put(buf)
	}()

	// Format timestamp
	t := time.Unix(0, int64(timestamp))
	buf = append(buf, "time="...)
	buf = t.AppendFormat(buf, time.RFC3339)

	// Add level
	buf = append(buf, " level="...)
	buf = append(buf, getLevelString(level)...)

	// Add message
	buf = append(buf, " msg="...)
	buf = appendQuoted(buf, msg)

	// Decode fields if present
	if pos < len(b) {
		fieldCount := int(b[pos])
		pos++

		for i := 0; i < fieldCount && pos < len(b); i++ {
			// Decode field key
			keyLen := int(b[pos])
			pos++
			if pos+keyLen > len(b) {
				break
			}
			key := string(b[pos : pos+keyLen])
			pos += keyLen

			if pos >= len(b) {
				break
			}

			fieldType := FieldType(b[pos])
			pos++

			// Add key
			buf = append(buf, ' ')
			buf = append(buf, key...)
			buf = append(buf, '=')

			// Decode and format value
			value := w.decodeFieldValue(b[pos:], fieldType)
			buf = append(buf, value...)
			pos += w.fieldValueSize(b[pos:], fieldType)
		}
	}

	buf = append(buf, '\n')

	// Write to output
	_, err := w.out.Write(buf)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// getLevelString returns the string representation of a level
func getLevelString(level Level) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// appendQuoted appends a quoted string if it contains spaces or special chars
func appendQuoted(buf []byte, s string) []byte {
	needsQuotes := false
	for _, c := range s {
		if c == ' ' || c == '"' || c == '=' || c == '\n' || c == '\r' {
			needsQuotes = true
			break
		}
	}

	if !needsQuotes && s != "" {
		return append(buf, s...)
	}

	// Quote the string
	buf = append(buf, '"')
	for _, c := range s {
		if c == '"' {
			buf = append(buf, '\\', '"')
		} else if c == '\\' {
			buf = append(buf, '\\', '\\')
		} else if c == '\n' {
			buf = append(buf, '\\', 'n')
		} else if c == '\r' {
			buf = append(buf, '\\', 'r')
		} else {
			buf = append(buf, byte(c))
		}
	}
	buf = append(buf, '"')
	return buf
}

// decodeFieldValue decodes a field value to string
func (w *LogfmtWriter) decodeFieldValue(b []byte, fieldType FieldType) string {
	switch fieldType {
	case FieldTypeInt:
		if len(b) < 8 {
			return "?"
		}
		// Big endian decoding to match encoding
		v := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
		return strconv.FormatInt(int64(v), 10)

	case FieldTypeUint:
		if len(b) < 8 {
			return "?"
		}
		// Big endian decoding to match encoding
		v := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
		return strconv.FormatUint(v, 10)

	case FieldTypeFloat32:
		if len(b) < 4 {
			return "?"
		}
		// Big endian decoding
		v := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		f := *(*float32)(unsafe.Pointer(&v))
		return strconv.FormatFloat(float64(f), 'g', -1, 32)

	case FieldTypeFloat64:
		if len(b) < 8 {
			return "?"
		}
		// Big endian decoding
		v := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
		f := *(*float64)(unsafe.Pointer(&v))
		return strconv.FormatFloat(f, 'g', -1, 64)

	case FieldTypeString:
		if len(b) < 2 {
			return "?"
		}
		// Big endian decoding for string length
		slen := int(uint16(b[0])<<8 | uint16(b[1]))
		if len(b) < 2+slen {
			return "?"
		}
		// Quote if needed
		var buf []byte
		return string(appendQuoted(buf, string(b[2:2+slen])))

	case FieldTypeBool:
		if len(b) < 8 {
			return "?"
		}
		// Big endian decoding
		v := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
		if v != 0 {
			return "true"
		}
		return "false"

	case FieldTypeBytes:
		if len(b) < 2 {
			return "?"
		}
		// Big endian decoding for bytes length
		blen := int(uint16(b[0])<<8 | uint16(b[1]))
		if len(b) < 2+blen {
			return "?"
		}
		// Format as hex string
		return fmt.Sprintf("%x", b[2:2+blen])

	default:
		return "?"
	}
}

// fieldValueSize returns the size of a field value in bytes
func (w *LogfmtWriter) fieldValueSize(b []byte, fieldType FieldType) int {
	switch fieldType {
	case FieldTypeInt, FieldTypeUint, FieldTypeFloat64, FieldTypeBool:
		return 8
	case FieldTypeFloat32:
		return 4
	case FieldTypeString, FieldTypeBytes:
		if len(b) < 2 {
			return 2
		}
		// Big endian decoding for length
		return 2 + int(uint16(b[0])<<8|uint16(b[1]))
	default:
		return 0
	}
}
