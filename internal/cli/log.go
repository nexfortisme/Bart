package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
)

// LogManager captures os.Stdout into a ring buffer so bot operational logs
// are invisible during menu/chat mode and only streamed when the user
// selects "Watch logs".
type LogManager struct {
	mu      sync.Mutex
	lines   []string
	watchCh chan string
	RealOut *os.File // original stdout, used by menu UI
}

// Capture redirects os.Stdout to an internal pipe and starts buffering.
// Call this before launching any goroutines that write to stdout.
func Capture() *LogManager {
	realOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		// Pipe creation failed — return a no-op manager that writes through.
		return &LogManager{RealOut: realOut}
	}
	os.Stdout = w

	lm := &LogManager{
		lines:   make([]string, 0, 500),
		RealOut: realOut,
	}
	go lm.drain(r)
	return lm
}

func (lm *LogManager) drain(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		lm.mu.Lock()
		if len(lm.lines) >= 500 {
			lm.lines = lm.lines[1:]
		}
		lm.lines = append(lm.lines, line)
		ch := lm.watchCh
		lm.mu.Unlock()

		if ch != nil {
			select {
			case ch <- line:
			default: // drop if watch consumer is too slow
			}
		}
	}
}

// Watch replays buffered lines then streams new ones to RealOut until done is closed.
func (lm *LogManager) Watch(done <-chan struct{}) {
	lm.mu.Lock()
	recent := make([]string, len(lm.lines))
	copy(recent, lm.lines)
	ch := make(chan string, 256)
	lm.watchCh = ch
	lm.mu.Unlock()

	for _, line := range recent {
		fmt.Fprintln(lm.RealOut, line)
	}

	for {
		select {
		case line := <-ch:
			fmt.Fprintln(lm.RealOut, line)
		case <-done:
			lm.mu.Lock()
			lm.watchCh = nil
			lm.mu.Unlock()
			return
		}
	}
}
