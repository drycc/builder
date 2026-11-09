package sshd

import (
	"sync"
	"sync/atomic"
	"testing"
)

const (
	numConcurrents = 1000
)

func TestOpenCloseSerial(t *testing.T) {
	c := NewCircuit()
	if c.State() != OpenState {
		t.Fatalf("unexpected initial circuit state. want %s, got %s", OpenState, c.State())
	}
	if b := c.Close(); !b {
		t.Fatalf("tried to close the circuit but it said it was already closed")
	}
	if c.State() != ClosedState {
		t.Fatalf("unexpected circuit state. want %s, got %s", ClosedState, c.State())
	}
	if b := c.Open(); !b {
		t.Fatalf("tried to open the circuit but it said it was already opened")
	}
	if c.State() != OpenState {
		t.Fatalf("unexpected circuit state. want %s, got %s", OpenState, c.State())
	}
}

func TestOpenCloseConcurrent(t *testing.T) {
	tests := []struct {
		name         string
		initialState CircuitState
		operation    func(*Circuit) bool
		wantState    CircuitState
	}{
		{
			name:         "close",
			initialState: OpenState,
			operation:    (*Circuit).Close,
			wantState:    ClosedState,
		},
		{
			name:         "open",
			initialState: ClosedState,
			operation:    (*Circuit).Open,
			wantState:    OpenState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCircuit()
			if tt.initialState == ClosedState {
				c.Close()
			}

			var wg sync.WaitGroup
			var successes uint32
			for i := 0; i < numConcurrents; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if tt.operation(c) {
						atomic.AddUint32(&successes, 1)
					}
				}()
			}
			wg.Wait()

			if successes != 1 {
				t.Errorf("expected exactly one successful %s, got %d", tt.name, successes)
			}
			if got := c.State(); got != tt.wantState {
				t.Errorf("expected final state %s, got %s", tt.wantState, got)
			}
		})
	}
}
