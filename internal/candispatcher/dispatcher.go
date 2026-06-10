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
	waitCh chan *canhal.Frame
	waitID uint32
	waitMu sync.Mutex

	// Pending buffer: holds frames that arrive before WaitFrame arms the waiter.
	// Uses a small slice so background goroutine frames (heartbeat, controller state)
	// don't clobber the response frame the upgrade protocol is waiting for.
	pendingBuf []*canhal.Frame
	pendingMu  sync.Mutex

	running atomic.Bool

	// Pool for reusing subscriber callback slices in readLoop.
	cbPool sync.Pool
}

// New creates a new Dispatcher for the given CAN backend.
func New(backend canhal.Backend) *Dispatcher {
	return &Dispatcher{
		backend:     backend,
		subscribers: make(map[uint64]func(*canhal.Frame)),
		waitCh:      make(chan *canhal.Frame, 1),
		cbPool: sync.Pool{
			New: func() any {
				s := make([]func(*canhal.Frame), 0, 8)
				return &s
			},
		},
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

// FeedFrame injects a frame into the dispatcher as if it was read from the backend.
// Used by the simulator to make response frames visible to WaitFrame and subscribers.
func (d *Dispatcher) FeedFrame(frame *canhal.Frame) {
	// 1. Try synchronous waiter first
	d.waitMu.Lock()
	if d.waitID != 0 && frame.ID == d.waitID {
		d.waitID = 0
		d.waitMu.Unlock()
		select {
		case d.waitCh <- frame:
		default:
		}
		goto fanout
	}
	d.waitMu.Unlock()

	// 2. Buffer in pending (multi-slot — background goroutines won't clobber responses)
	d.pendingMu.Lock()
	if len(d.pendingBuf) < 4 {
		d.pendingBuf = append(d.pendingBuf, frame)
	}
	d.pendingMu.Unlock()

	// 3. Re-check waiter (race: WaitFrame may have armed between step 1 and 2)
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

fanout:
	// 4. Fan out to all async subscribers
	sp := d.cbPool.Get().(*[]func(*canhal.Frame))
	cbs := (*sp)[:0]
	d.mu.RLock()
	for _, cb := range d.subscribers {
		cbs = append(cbs, cb)
	}
	d.mu.RUnlock()
	for _, cb := range cbs {
		cb(frame)
	}
	*sp = cbs
	d.cbPool.Put(sp)
}

// WaitFrame blocks until a frame with the expected ID arrives or timeout.
// Used by the firmware upgrade protocol for request/response.
func (d *Dispatcher) WaitFrame(expectedID uint32, timeout time.Duration) (*canhal.Frame, error) {
	// Check pending buffer first (response arrived before we started waiting).
	d.pendingMu.Lock()
	for i, p := range d.pendingBuf {
		if p.ID == expectedID {
			d.pendingBuf = append(d.pendingBuf[:i], d.pendingBuf[i+1:]...)
			d.pendingMu.Unlock()
			return p, nil
		}
	}
	d.pendingMu.Unlock()

	// Arm waiter.
	d.waitMu.Lock()
	d.waitID = expectedID
	// Drain any stale frame from channel.
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
			// No waiter armed — buffer one frame for late WaitFrame calls.
			// Only buffer if it matches a typical response ID (0x102 PlatformTx).
			d.pendingMu.Lock()
			if len(d.pendingBuf) < 4 {
				d.pendingBuf = append(d.pendingBuf, frame)
			}
			d.pendingMu.Unlock()
		}

		// 2. Fan out to all async subscribers (copy under RLock, invoke outside)
		//    Use sync.Pool to reuse the callback slice across iterations.
		sp := d.cbPool.Get().(*[]func(*canhal.Frame))
		cbs := (*sp)[:0]

		d.mu.RLock()
		for _, cb := range d.subscribers {
			cbs = append(cbs, cb)
		}
		d.mu.RUnlock()

		for _, cb := range cbs {
			cb(frame)
		}

		// Return the slice to the pool (reset length but keep capacity).
		*sp = cbs
		d.cbPool.Put(sp)
	}
}
