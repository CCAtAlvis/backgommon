// Package worker defines the subprocess protocol between the daemon and strategy binaries.
//
// Wire format is JSON on stdin/stdout. Commands: describe | run | sweep.
package worker

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// DescribeResponse is returned by `binary describe`.
type DescribeResponse struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"` // JSON Schema-shaped object for UI forms
}

// RunRequest is sent on stdin to `binary run`.
type RunRequest struct {
	Config    json.RawMessage `json:"config"`
	OutputDir string          `json:"output_dir"`
}

// RunResponse is written to stdout by `binary run`.
type RunResponse struct {
	OK        bool               `json:"ok"`
	ReportDir string             `json:"report_dir,omitempty"`
	Metrics   map[string]float64 `json:"metrics,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// SweepRequest is sent on stdin to `binary sweep`.
type SweepRequest struct {
	Base      json.RawMessage `json:"base"`
	Cases     json.RawMessage `json:"cases,omitempty"` // explicit list of config objects; if set, axes ignored
	Axes      json.RawMessage `json:"axes,omitempty"`  // optional axes description for the worker
	OutputDir string          `json:"output_dir"`
	Workers   int             `json:"workers,omitempty"`
	SortBy    string          `json:"sort_by,omitempty"`
}

// SweepResponse is written to stdout by `binary sweep`.
type SweepResponse struct {
	OK          bool   `json:"ok"`
	SweepDir    string `json:"sweep_dir,omitempty"`
	SummaryPath string `json:"summary_path,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Handlers are implemented by a strategy worker main.
type Handlers struct {
	Describe func() (*DescribeResponse, error)
	Run      func(req RunRequest) (*RunResponse, error)
	Sweep    func(req SweepRequest) (*SweepResponse, error)
}

// Main dispatches os.Args for a worker binary. Call from strategy main when
// the first argument is describe|run|sweep; otherwise return false.
func Main(h Handlers) bool {
	if len(os.Args) < 2 {
		return false
	}
	switch os.Args[1] {
	case "describe", "run", "sweep":
		os.Exit(dispatch(h, os.Args[1], os.Stdin, os.Stdout, os.Stderr))
		return true
	default:
		return false
	}
}

func dispatch(h Handlers, cmd string, in io.Reader, out, errOut io.Writer) int {
	switch cmd {
	case "describe":
		if h.Describe == nil {
			fmt.Fprintln(errOut, "describe not implemented")
			return 1
		}
		resp, err := h.Describe()
		if err != nil {
			fmt.Fprintln(errOut, err.Error())
			return 1
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintln(errOut, err.Error())
			return 1
		}
		return 0
	case "run":
		if h.Run == nil {
			fmt.Fprintln(errOut, "run not implemented")
			return 1
		}
		var req RunRequest
		if err := json.NewDecoder(in).Decode(&req); err != nil {
			writeRunErr(out, fmt.Sprintf("decode run request: %v", err))
			return 1
		}
		resp, err := h.Run(req)
		if err != nil {
			writeRunErr(out, err.Error())
			return 1
		}
		if resp == nil {
			writeRunErr(out, "nil run response")
			return 1
		}
		if !resp.OK && resp.Error == "" {
			resp.Error = "run failed"
		}
		_ = json.NewEncoder(out).Encode(resp)
		if !resp.OK {
			return 1
		}
		return 0
	case "sweep":
		if h.Sweep == nil {
			fmt.Fprintln(errOut, "sweep not implemented")
			return 1
		}
		var req SweepRequest
		if err := json.NewDecoder(in).Decode(&req); err != nil {
			writeSweepErr(out, fmt.Sprintf("decode sweep request: %v", err))
			return 1
		}
		resp, err := h.Sweep(req)
		if err != nil {
			writeSweepErr(out, err.Error())
			return 1
		}
		if resp == nil {
			writeSweepErr(out, "nil sweep response")
			return 1
		}
		_ = json.NewEncoder(out).Encode(resp)
		if !resp.OK {
			return 1
		}
		return 0
	default:
		fmt.Fprintf(errOut, "unknown worker command: %s\n", cmd)
		return 2
	}
}

func writeRunErr(out io.Writer, msg string) {
	_ = json.NewEncoder(out).Encode(RunResponse{OK: false, Error: msg})
}

func writeSweepErr(out io.Writer, msg string) {
	_ = json.NewEncoder(out).Encode(SweepResponse{OK: false, Error: msg})
}

// Client invokes a worker binary.
type Client struct {
	Binary string
}

// Describe runs `binary describe`.
func (c *Client) Describe() (*DescribeResponse, error) {
	cmd := exec.Command(c.Binary, "describe")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("describe failed: %s", stringsTrim(ee.Stderr))
		}
		return nil, err
	}
	var resp DescribeResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse describe: %w\noutput: %s", err, string(out))
	}
	if resp.Name == "" {
		return nil, fmt.Errorf("describe: name is required")
	}
	return &resp, nil
}

// Run runs `binary run` with JSON stdin.
func (c *Client) Run(req RunRequest) (*RunResponse, []byte, error) {
	return c.runCmd("run", req)
}

// Sweep runs `binary sweep` with JSON stdin.
func (c *Client) Sweep(req SweepRequest) (*SweepResponse, []byte, error) {
	body, stderr, err := c.execJSON("sweep", req)
	if err != nil && body == nil {
		return nil, stderr, err
	}
	var resp SweepResponse
	if uerr := json.Unmarshal(body, &resp); uerr != nil {
		if err != nil {
			return nil, stderr, fmt.Errorf("%v; also parse: %v; stderr: %s", err, uerr, stringsTrim(stderr))
		}
		return nil, stderr, uerr
	}
	if err != nil && !resp.OK {
		return &resp, stderr, fmt.Errorf("%s", firstNonEmpty(resp.Error, stringsTrim(stderr), err.Error()))
	}
	if !resp.OK {
		return &resp, stderr, fmt.Errorf("%s", firstNonEmpty(resp.Error, "sweep failed"))
	}
	return &resp, stderr, nil
}

func (c *Client) runCmd(subcommand string, req RunRequest) (*RunResponse, []byte, error) {
	body, stderr, err := c.execJSON(subcommand, req)
	if err != nil && body == nil {
		return nil, stderr, err
	}
	var resp RunResponse
	if uerr := json.Unmarshal(body, &resp); uerr != nil {
		if err != nil {
			return nil, stderr, fmt.Errorf("%v; also parse: %v; stderr: %s", err, uerr, stringsTrim(stderr))
		}
		return nil, stderr, uerr
	}
	if err != nil && !resp.OK {
		return &resp, stderr, fmt.Errorf("%s", firstNonEmpty(resp.Error, stringsTrim(stderr), err.Error()))
	}
	if !resp.OK {
		return &resp, stderr, fmt.Errorf("%s", firstNonEmpty(resp.Error, "run failed"))
	}
	return &resp, stderr, nil
}

func (c *Client) execJSON(subcommand string, payload any) (stdout, stderr []byte, err error) {
	bin, err := filepath.Abs(c.Binary)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.Command(bin, subcommand)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	var outBuf, errBuf limitedBuffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	enc := json.NewEncoder(stdin)
	if err := enc.Encode(payload); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, errBuf.Bytes(), err
	}
	_ = stdin.Close()
	waitErr := cmd.Wait()
	return outBuf.Bytes(), errBuf.Bytes(), waitErr
}

// StartRun starts a run process without waiting (for cancelable jobs).
func StartRun(binary string, req RunRequest) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	bin, err := filepath.Abs(binary)
	if err != nil {
		return nil, nil, nil, err
	}
	cmd := exec.Command(bin, "run")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	go func() {
		_ = json.NewEncoder(stdin).Encode(req)
		_ = stdin.Close()
	}()
	return cmd, stdout, stderr, nil
}

// StartSweep starts a sweep process without waiting.
func StartSweep(binary string, req SweepRequest) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	bin, err := filepath.Abs(binary)
	if err != nil {
		return nil, nil, nil, err
	}
	cmd := exec.Command(bin, "sweep")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	go func() {
		_ = json.NewEncoder(stdin).Encode(req)
		_ = stdin.Close()
	}()
	return cmd, stdout, stderr, nil
}

type limitedBuffer struct {
	b []byte
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	const max = 2 << 20
	if len(l.b) < max {
		need := max - len(l.b)
		if len(p) < need {
			need = len(p)
		}
		l.b = append(l.b, p[:need]...)
	}
	return len(p), nil
}

func (l *limitedBuffer) Bytes() []byte { return l.b }

func stringsTrim(b []byte) string {
	s := string(b)
	if len(s) > 4000 {
		return s[:4000] + "…"
	}
	return s
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return "error"
}
