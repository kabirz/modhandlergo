// Package candispatcher provides a goroutine-based CAN frame read loop
// with channel-based pub/sub and synchronous frame waiting.
package candispatcher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kabirz/modhandlergo/internal/canhal"
)

// Dispatcher manages a single CAN read goroutine and distributes
// received frames to subscribers and synchronous waiters.
type Dispatcher struct {
	backend canhal.Backend
	cancel  context.CancelFunc

	// Pub/sub: subscribers receive every frame read.
	mu          sync.RWMutex
	subscribers map[uint64]func(*canhal.Frame)
	nextSubID   uint64

	// Synchronous wait: for request/response protocol (firmware upgrade).
	waitCh  chan *canhal.Frame
	waitID  uint32
	waitMu  sync.Mutex

	running atomic.Bool
}

// New creates a new Dispatcher for the given CAN backend.
func New(backend canhal.Backend) *Dispatcher {
	return &Dispatcher{
		backend:     backend,
		subscribers: make(map[uint64]func(*canhal.Frame)),
		waitCh:      make(chan *canhal.Frame, 1),
	}
}

// Start launches the goroutine read loop.
func (d *Dispatcher) Start() {
	if d.running.Load() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.running.Store(true)
	go d.readLoop(ctx)
}

// Stop signals the read loop to exit and waits for it to finish.
func (d *Dispatcher) Stop() {
	if !d.running.Load() {
		return
	}
	d.running.Store(false)
	if d.cancel != nil {
		d.cancel()
	}
}

// Subscribe registers a callback for every received frame.
// Returns an unsubscribe function.
func (d *Dispatcher) Subscribe(cb func(*canhal.Frame)) func() {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := d.nextSubID
	d.nextSubID++
	d.subscribers[id] = cb

	return func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		delete(d.subscribers, id)
	}
}

// WaitFrame blocks until a frame with the expected ID arrives or timeout.
// Used by the firmware upgrade protocol for request/response.
func (d *Dispatcher) WaitFrame(expectedID uint32, timeout time.Duration) (*canhal.Frame, error) {
	// Arm waiter before sending command.
	d.waitMu.Lock()
	d.waitID = expectedID
	// Drain any stale frame
	select {
	case <-d.waitCh:
	default:
	}
	d.waitMu.Unlock()

	select {
	case frame := <-d.waitCh:
		return frame, nil
	case <-time.After(timeout):
		d.waitMu.Lock()
		d.waitID = 0
		d.waitMu.Unlock()
		return nil, fmt.Errorf("wait for frame 0x%03X timed out", expectedID)
	}
}

// ReplaceBackend swaps the CAN backend (for adapter hot-switching).
// Must be called when disconnected.
func (d *Dispatcher) ReplaceBackend(backend canhal.Backend) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.backend = backend
}

func (d *Dispatcher) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if !d.running.Load() {
			return
		}

		frame, err := d.backend.Read(10 * time.Millisecond)
		if err != nil || frame == nil {
			continue
		}

		// 1. Check synchronous waiter
		d.waitMu.Lock()
		if d.waitID != 0 && frame.ID == d.waitID {
			d.waitID = 0
			d.waitMu.Unlock()
			select {
			case d.waitCh <- frame:
			default:
			}
		} else {
			d.waitMu.Unlock()
		}

		// 2. Fan out to all async subscribers (copy under RLock, invoke outside)
		d.mu.RLock()
		cbs := make([]func(*canhal.Frame), 0, len(d.subscribers))
		for _, cb := range d.subscribers {
			cbs = append(cbs, cb)
		}
		d.mu.RUnlock()
		for _, cb := range cbs {
			cb(frame)
		}
	}
}
