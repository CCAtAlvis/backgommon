package outmgr

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBufferUntilFlush(t *testing.T) {
	sink := &BufferSink{}
	m := New(sink)

	m.Printf("hello %s\n", "world")
	m.Println("line2")

	if sink.String() != "" {
		t.Fatalf("sink received data before Flush: %q", sink.String())
	}
	if m.BufferedLen() == 0 {
		t.Fatal("expected buffered content")
	}

	if err := m.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	want := "hello world\nline2\n"
	if got := sink.String(); got != want {
		t.Fatalf("after Flush: got %q want %q", got, want)
	}
	if m.BufferedLen() != 0 {
		t.Fatalf("buffer not cleared: %d", m.BufferedLen())
	}

	// Second flush with no new writes is a no-op
	if err := m.Flush(); err != nil {
		t.Fatalf("second Flush: %v", err)
	}
	if got := sink.String(); got != want {
		t.Fatalf("second Flush mutated sink: %q", got)
	}
}

func TestFlushFansOutToMultipleSinks(t *testing.T) {
	a, b := &BufferSink{}, &BufferSink{}
	m := New(a, b)
	m.Printf("payload-%d", 42)
	if err := m.Flush(); err != nil {
		t.Fatal(err)
	}
	want := "payload-42"
	if a.String() != want || b.String() != want {
		t.Fatalf("a=%q b=%q want %q", a.String(), b.String(), want)
	}
}

func TestDefaultDelegates(t *testing.T) {
	sink := &BufferSink{}
	prev := Default()
	SetDefault(New(sink))
	defer SetDefault(prev)

	Printf("a=%d\n", 1)
	Println("b")
	if sink.String() != "" {
		t.Fatal("default wrote before Flush")
	}
	if err := Flush(); err != nil {
		t.Fatal(err)
	}
	got := sink.String()
	if !strings.Contains(got, "a=1\n") || !strings.Contains(got, "b\n") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestAsLogger(t *testing.T) {
	sink := &BufferSink{}
	m := New(sink)
	log := AsLogger(m)
	log.Printf("risk %s\n", "event")
	_ = m.Flush()
	if sink.String() != "risk event\n" {
		t.Fatalf("got %q", sink.String())
	}
}

func TestManagerImplementsWriter(t *testing.T) {
	sink := &BufferSink{}
	m := New(sink)
	n, err := io.WriteString(m, "via writer\n")
	if err != nil || n != 11 {
		t.Fatalf("WriteString: n=%d err=%v", n, err)
	}
	_ = m.Flush()
	if sink.String() != "via writer\n" {
		t.Fatalf("got %q", sink.String())
	}
}

// Ensure ConsoleSink is constructible without writing (no Flush of empty).
func TestConsoleSinkNoopEmptyFlush(t *testing.T) {
	m := New(ConsoleSink())
	if err := m.Flush(); err != nil {
		t.Fatal(err)
	}
}

func TestStdoutCaptureOnFlush(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	m := New(ConsoleSink())
	m.Printf("buffered-only\n")

	if m.BufferedLen() == 0 {
		t.Fatal("expected buffer")
	}

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	if err := m.Flush(); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	got := <-done
	if got != "buffered-only\n" {
		t.Fatalf("stdout got %q", got)
	}
}

func TestImmediateModeWriteThrough(t *testing.T) {
	sink := &BufferSink{}
	m := New(sink)
	m.SetMode(ModeImmediate)
	m.Printf("now-%d\n", 1)
	if sink.String() != "now-1\n" {
		t.Fatalf("immediate should write through: %q", sink.String())
	}
	if m.BufferedLen() != 0 {
		t.Fatalf("immediate should not buffer: %d", m.BufferedLen())
	}
}

func TestDropClearsWithoutWriting(t *testing.T) {
	sink := &BufferSink{}
	m := New(sink)
	m.Printf("discard me\n")
	m.Drop()
	if m.BufferedLen() != 0 {
		t.Fatal("Drop should clear buffer")
	}
	if err := m.Flush(); err != nil {
		t.Fatal(err)
	}
	if sink.String() != "" {
		t.Fatalf("Drop then Flush should write nothing: %q", sink.String())
	}
}

func TestSetModePackageHelpers(t *testing.T) {
	sink := &BufferSink{}
	prev := Default()
	SetDefault(New(sink))
	defer SetDefault(prev)

	SetMode(ModeImmediate)
	Printf("live\n")
	if sink.String() != "live\n" {
		t.Fatalf("got %q", sink.String())
	}
	SetMode(ModeBuffered)
	Printf("held\n")
	Drop()
	_ = Flush()
	if sink.String() != "live\n" {
		t.Fatalf("after Drop: %q", sink.String())
	}
}
