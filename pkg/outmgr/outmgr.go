// Package outmgr provides a text output manager with Immediate and Buffered modes.
// Buffered mode accumulates until Flush (or Drop clears without writing). Immediate
// mode writes through to sinks on each Printf/Println. Extensible via Sink.
package outmgr

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

// Mode controls whether Printf writes through or buffers.
type Mode int

const (
	// ModeBuffered accumulates output until Flush (default).
	ModeBuffered Mode = iota
	// ModeImmediate writes each Printf/Println to sinks immediately.
	ModeImmediate
)

// Sink receives flushed or immediate output.
type Sink interface {
	Write(p []byte) (int, error)
	Flush() error
}

// Manager routes text to sinks with buffering or immediate write-through.
type Manager struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	sinks  []Sink
	mode   Mode
}

// New creates a Manager that flushes to the given sinks (default ModeBuffered).
func New(sinks ...Sink) *Manager {
	cp := make([]Sink, len(sinks))
	copy(cp, sinks)
	return &Manager{sinks: cp, mode: ModeBuffered}
}

// ConsoleSink returns a Sink that writes to os.Stdout.
func ConsoleSink() Sink {
	return &writerSink{w: os.Stdout}
}

type writerSink struct {
	w io.Writer
}

func (s *writerSink) Write(p []byte) (int, error) { return s.w.Write(p) }
func (s *writerSink) Flush() error {
	if f, ok := s.w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// BufferSink is an in-memory Sink useful for tests.
type BufferSink struct {
	mu  sync.Mutex
	Buf bytes.Buffer
}

func (s *BufferSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Buf.Write(p)
}

func (s *BufferSink) Flush() error { return nil }

func (s *BufferSink) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Buf.String()
}

// SetMode switches between buffered and immediate output.
func (m *Manager) SetMode(mode Mode) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
}

// Mode returns the current output mode.
func (m *Manager) Mode() Mode {
	if m == nil {
		return ModeBuffered
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mode
}

// Printf formats and either buffers or writes through based on mode.
func (m *Manager) Printf(format string, args ...interface{}) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mode == ModeImmediate {
		var b bytes.Buffer
		fmt.Fprintf(&b, format, args...)
		m.writeSinksLocked(b.Bytes())
		return
	}
	fmt.Fprintf(&m.buf, format, args...)
}

// Println appends args like fmt.Println.
func (m *Manager) Println(args ...interface{}) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mode == ModeImmediate {
		var b bytes.Buffer
		fmt.Fprintln(&b, args...)
		m.writeSinksLocked(b.Bytes())
		return
	}
	fmt.Fprintln(&m.buf, args...)
}

// Write implements io.Writer (respects mode).
func (m *Manager) Write(p []byte) (int, error) {
	if m == nil {
		return 0, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mode == ModeImmediate {
		m.writeSinksLocked(p)
		return len(p), nil
	}
	return m.buf.Write(p)
}

func (m *Manager) writeSinksLocked(payload []byte) {
	if len(payload) == 0 {
		return
	}
	for _, s := range m.sinks {
		if s == nil {
			continue
		}
		_, _ = s.Write(payload)
		_ = s.Flush()
	}
}

// Flush writes the buffered content to every sink, then clears the buffer.
func (m *Manager) Flush() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	payload := append([]byte(nil), m.buf.Bytes()...)
	m.buf.Reset()
	sinks := append([]Sink(nil), m.sinks...)
	m.mu.Unlock()

	if len(payload) == 0 {
		return nil
	}

	var firstErr error
	for _, s := range sinks {
		if s == nil {
			continue
		}
		if _, err := s.Write(payload); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := s.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Drop discards the in-memory buffer without writing to sinks.
func (m *Manager) Drop() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buf.Reset()
}

// BufferedLen returns the current buffer size (for tests).
func (m *Manager) BufferedLen() int {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.Len()
}

// PrintfLogger adapts a Manager to Printf-style logger interfaces (e.g. risk.Logger).
type PrintfLogger struct {
	M *Manager
}

func (l PrintfLogger) Printf(format string, args ...interface{}) {
	m := l.M
	if m == nil {
		m = Default()
	}
	m.Printf(format, args...)
}

// AsLogger returns a PrintfLogger backed by m.
func AsLogger(m *Manager) PrintfLogger {
	return PrintfLogger{M: m}
}

var (
	defaultMu sync.RWMutex
	defaultM  *Manager
)

func init() {
	defaultM = New(ConsoleSink())
}

// SetDefault replaces the package-level default manager.
func SetDefault(m *Manager) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if m == nil {
		defaultM = New(ConsoleSink())
		return
	}
	defaultM = m
}

// Default returns the package-level manager (never nil).
func Default() *Manager {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultM
}

// SetMode sets the default manager's mode.
func SetMode(mode Mode) {
	Default().SetMode(mode)
}

// Printf writes to the default manager.
func Printf(format string, args ...interface{}) {
	Default().Printf(format, args...)
}

// Println writes to the default manager.
func Println(args ...interface{}) {
	Default().Println(args...)
}

// Flush flushes the default manager.
func Flush() error {
	return Default().Flush()
}

// Drop clears the default manager's buffer without writing.
func Drop() {
	Default().Drop()
}
