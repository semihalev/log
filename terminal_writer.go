package zlog

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	termTimeFormat = "01-02|15:04:05"
	termMsgJust    = 40
)

// Color codes for terminal output
const (
	colorReset   = "\x1b[0m"
	colorRed     = "\x1b[31m"
	colorGreen   = "\x1b[32m"
	colorYellow  = "\x1b[33m"
	colorBlue    = "\x1b[34m"
	colorMagenta = "\x1b[35m"
	colorCyan    = "\x1b[36m"
	colorGray    = "\x1b[37m"
	colorBold    = "\x1b[1m"
)

// TerminalWriter decodes binary log format and outputs beautiful terminal format
type TerminalWriter struct {
	out        *os.File
	useColor   bool
	timeFormat string
	buf        sync.Pool
}

// NewTerminalWriter creates a new terminal writer
func NewTerminalWriter(out *os.File) *TerminalWriter {
	return &TerminalWriter{
		out:        out,
		useColor:   isTerminal(out.Fd()),
		timeFormat: termTimeFormat,
		buf: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 512)
			},
		},
	}
}

// Writer returns a Writer function for the logger
func (w *TerminalWriter) Writer() Writer {
	return func(b []byte) error {
		return w.Write(b)
	}
}

// Write decodes binary log and outputs formatted text
func (w *TerminalWriter) Write(b []byte) error {
	if len(b) < 22 { // Minimum header size
		return fmt.Errorf("invalid log entry: too short")
	}

	// Decode binary header
	magic := binary.LittleEndian.Uint32(b[0:4])
	if magic != MagicHeader {
		return fmt.Errorf("invalid magic header")
	}

	// version := b[4]
	level := Level(b[5])
	// seq := binary.LittleEndian.Uint64(b[6:14])
	timestamp := binary.LittleEndian.Uint64(b[14:22])

	pos := 22

	// Get message length and message
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
	if bufInterface == nil {
		return fmt.Errorf("buffer pool returned nil")
	}
	buf := bufInterface.([]byte)
	buf = buf[:0]
	defer func() {
		w.buf.Put(buf)
	}()

	// Format output with colors
	color := w.getLevelColor(level)
	levelStr := w.getLevelString(level)

	if w.useColor && color != "" {
		buf = append(buf, color...)
		buf = append(buf, levelStr...)
		buf = append(buf, colorReset...)
	} else {
		buf = append(buf, levelStr...)
	}

	// Add timestamp
	t := time.Unix(0, int64(timestamp))
	buf = append(buf, '[')
	buf = t.AppendFormat(buf, w.timeFormat)
	buf = append(buf, "] "...)

	// Add message
	buf = append(buf, msg...)

	// Justify message for readability if we have fields
	if pos < len(b) && len(msg) < termMsgJust {
		padding := termMsgJust - len(msg)
		for i := 0; i < padding; i++ {
			buf = append(buf, ' ')
		}
	}

	// Decode fields if present
	if pos < len(b) {
		fieldCount := int(b[pos])
		pos++

		for i := 0; i < fieldCount && pos < len(b); i++ {
			if i > 0 {
				buf = append(buf, ' ')
			}

			// Decode field
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

			// Format key
			if w.useColor && color != "" {
				buf = append(buf, color...)
				buf = append(buf, key...)
				buf = append(buf, colorReset...)
				buf = append(buf, '=')
			} else {
				buf = append(buf, key...)
				buf = append(buf, '=')
			}

			// Format value based on type
			value := w.decodeFieldValue(b[pos:], fieldType)
			buf = append(buf, value...)
			pos += w.fieldValueSize(b[pos:], fieldType)
		}
	}

	buf = append(buf, '\n')

	// Write to output
	_, err := w.out.Write(buf)
	return err
}

// getLevelColor returns the color for a log level
func (w *TerminalWriter) getLevelColor(level Level) string {
	switch level {
	case LevelDebug:
		return colorCyan
	case LevelInfo:
		return colorGreen
	case LevelWarn:
		return colorYellow
	case LevelError:
		return colorRed
	case LevelFatal:
		return colorMagenta
	default:
		return ""
	}
}

// getLevelString returns the string representation of a level
func (w *TerminalWriter) getLevelString(level Level) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO "
	case LevelWarn:
		return "WARN "
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKN "
	}
}

// decodeFieldValue decodes a field value from binary
func (w *TerminalWriter) decodeFieldValue(b []byte, fieldType FieldType) string {
	switch fieldType {
	case FieldTypeInt:
		if len(b) >= 8 {
			// Big endian decoding to match encoding
			v := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
				uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
			return fmt.Sprintf("%d", int64(v))
		}
	case FieldTypeUint, FieldTypeBool:
		if len(b) >= 8 {
			v := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
				uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
			if fieldType == FieldTypeBool {
				if v == 0 {
					return "false"
				}
				return "true"
			}
			return fmt.Sprintf("%d", v)
		}
	case FieldTypeFloat32:
		if len(b) >= 4 {
			v := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
			f := *(*float32)(unsafe.Pointer(&v))
			return fmt.Sprintf("%.3f", f)
		}
	case FieldTypeFloat64:
		if len(b) >= 8 {
			v := uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
				uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
			f := *(*float64)(unsafe.Pointer(&v))
			return fmt.Sprintf("%.3f", f)
		}
	case FieldTypeString:
		if len(b) >= 2 {
			strLen := int(uint16(b[0])<<8 | uint16(b[1]))
			if len(b) >= 2+strLen {
				return escapeString(string(b[2 : 2+strLen]))
			}
		}
	case FieldTypeBytes:
		if len(b) >= 2 {
			dataLen := int(uint16(b[0])<<8 | uint16(b[1]))
			if len(b) >= 2+dataLen {
				return fmt.Sprintf("%x", b[2:2+dataLen])
			}
		}
	}
	return "?"
}

// fieldValueSize returns the size of a field value in bytes
func (w *TerminalWriter) fieldValueSize(b []byte, fieldType FieldType) int {
	switch fieldType {
	case FieldTypeInt, FieldTypeUint, FieldTypeBool, FieldTypeFloat64:
		return 8
	case FieldTypeFloat32:
		return 4
	case FieldTypeString, FieldTypeBytes:
		if len(b) >= 2 {
			return 2 + int(uint16(b[0])<<8|uint16(b[1]))
		}
	}
	return 0
}

// escapeString escapes a string for terminal output
func escapeString(s string) string {
	if !strings.ContainsAny(s, "\\\"\n\r\t ") {
		return s
	}

	// Use a stack buffer for small strings
	var buf [128]byte
	b := buf[:0]

	b = append(b, '"')
	for _, r := range s {
		switch r {
		case '\\', '"':
			b = append(b, '\\', byte(r))
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		case '\t':
			b = append(b, '\\', 't')
		default:
			if r < 128 && len(b) < len(buf)-2 {
				b = append(b, byte(r))
			} else {
				// Fallback to allocation
				return fmt.Sprintf("%q", s)
			}
		}
	}
	b = append(b, '"')

	return string(b)
}

// Convenience functions for creating terminal writers

// StdoutTerminal creates a terminal writer for stdout
func StdoutTerminal() Writer {
	return NewTerminalWriter(os.Stdout).Writer()
}

// StderrTerminal creates a terminal writer for stderr
func StderrTerminal() Writer {
	return NewTerminalWriter(os.Stderr).Writer()
}
