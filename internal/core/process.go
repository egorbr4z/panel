// Package core supervises the proxy core subprocesses (Xray, sing-box):
// config generation, process lifecycle and log capture. Designed to stay light
// on a 1 vCPU host — supervision is a couple of goroutines per core, logs live
// in a bounded in-memory ring buffer (no disk log growth).
package core

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// State is the lifecycle state of a supervised process.
type State int32

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StateCrashed // crash-looping; supervisor gave up restarting
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateCrashed:
		return "crashed"
	default:
		return "unknown"
	}
}

// ringBuffer is a tiny thread-safe last-N-lines log buffer implementing io.Writer.
type ringBuffer struct {
	mu    sync.Mutex
	lines []string
	max   int
}

func newRingBuffer(max int) *ringBuffer { return &ringBuffer{max: max} }

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Split on newlines; callers (bufio.Scanner) already feed whole lines.
	r.lines = append(r.lines, string(p))
	if len(r.lines) > r.max {
		r.lines = r.lines[len(r.lines)-r.max:]
	}
	return len(p), nil
}

func (r *ringBuffer) Snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.lines))
	copy(out, r.lines)
	return out
}

// Process is a supervised core subprocess with crash-loop protection.
type Process struct {
	name string
	bin  string
	args []string

	mu       sync.Mutex
	cmd      *exec.Cmd
	stopping bool
	state    atomic.Int32
	logs     *ringBuffer

	// crash-loop tracking: timestamps of recent restarts.
	restartTimes []time.Time
	restarts     atomic.Int64
}

// NewProcess builds a supervised process. It is not started until Start.
func NewProcess(name, bin string, args ...string) *Process {
	return &Process{name: name, bin: bin, args: args, logs: newRingBuffer(300)}
}

// Name returns the process name.
func (p *Process) Name() string { return p.name }

// State returns the current lifecycle state.
func (p *Process) State() State { return State(p.state.Load()) }

// Restarts returns the cumulative restart count.
func (p *Process) Restarts() int64 { return p.restarts.Load() }

// Logs returns a snapshot of the recent log lines.
func (p *Process) Logs() []string { return p.logs.Snapshot() }

// SetArgs updates the launch arguments (used when the config path is fixed but
// flags change). Takes effect on the next (re)start.
func (p *Process) SetArgs(args ...string) {
	p.mu.Lock()
	p.args = args
	p.mu.Unlock()
}

// Start launches the process if it is not already running.
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.Load() == int32(StateRunning) || p.state.Load() == int32(StateStarting) {
		return nil
	}
	p.stopping = false
	return p.launchLocked()
}

// launchLocked starts the subprocess; caller must hold p.mu.
func (p *Process) launchLocked() error {
	p.state.Store(int32(StateStarting))
	cmd := exec.Command(p.bin, p.args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	go p.pump(stdout)
	go p.pump(stderr)

	if err := cmd.Start(); err != nil {
		p.state.Store(int32(StateStopped))
		p.logs.Write([]byte("failed to start: " + err.Error()))
		return err
	}
	p.cmd = cmd
	p.state.Store(int32(StateRunning))
	go p.supervise(cmd)
	return nil
}

// pump streams a reader line-by-line into the ring buffer.
func (p *Process) pump(r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 256*1024)
	for sc.Scan() {
		p.logs.Write([]byte(sc.Text()))
	}
}

// supervise waits for the process and restarts it on unexpected exit, with
// exponential backoff and crash-loop detection.
func (p *Process) supervise(cmd *exec.Cmd) {
	err := cmd.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopping {
		p.state.Store(int32(StateStopped))
		return
	}
	if err != nil {
		p.logs.Write([]byte("process exited: " + err.Error()))
	} else {
		p.logs.Write([]byte("process exited"))
	}

	// Crash-loop detection: >5 restarts within 60s -> give up.
	now := time.Now()
	p.restartTimes = append(p.restartTimes, now)
	cutoff := now.Add(-60 * time.Second)
	recent := p.restartTimes[:0]
	for _, t := range p.restartTimes {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	p.restartTimes = recent
	if len(p.restartTimes) > 5 {
		p.logs.Write([]byte("crash-loop detected, giving up restarts"))
		p.state.Store(int32(StateCrashed))
		return
	}

	// Exponential backoff capped at 30s.
	backoff := time.Duration(len(p.restartTimes)) * 2 * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}

	p.mu.Unlock()
	time.Sleep(backoff)
	p.mu.Lock()
	if p.stopping {
		p.state.Store(int32(StateStopped))
		return
	}
	p.restarts.Add(1)
	_ = p.launchLocked()
}

// Stop terminates the process and prevents auto-restart.
func (p *Process) Stop() {
	p.mu.Lock()
	p.stopping = true
	cmd := p.cmd
	p.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		p.state.Store(int32(StateStopped))
		return
	}
	// Try a graceful stop, then force-kill after a grace period.
	_ = cmd.Process.Signal(interruptSignal())
	done := make(chan struct{})
	go func() { _, _ = cmd.Process.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
	}
	p.state.Store(int32(StateStopped))
}

// Restart stops and starts the process (used after a config change).
func (p *Process) Restart(ctx context.Context) error {
	p.Stop()
	// Reset crash-loop window so an intentional restart isn't penalised.
	p.mu.Lock()
	p.restartTimes = nil
	p.mu.Unlock()
	return p.Start()
}
