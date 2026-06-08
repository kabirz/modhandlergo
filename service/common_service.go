package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/kabirz/modhandlergo/internal/cancommand"
	"github.com/kabirz/modhandlergo/internal/candispatcher"
	"github.com/kabirz/modhandlergo/internal/canhal"
	"github.com/kabirz/modhandlergo/internal/canmanager"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// notifyingBackend wraps a canhal.Backend and calls a callback after every Write.
type notifyingBackend struct {
	canhal.Backend
	onWrite func(*canhal.Frame)
}

func (n *notifyingBackend) Write(frame *canhal.Frame) error {
	// When a simulator callback is registered, route frames in-process only
	// and skip the physical CAN bus. This prevents PCAN from entering Bus-Off
	// due to thousands of un-ACKed writes during simulated firmware upgrade.
	if n.onWrite != nil {
		n.onWrite(frame)
		return nil
	}
	return n.Backend.Write(frame)
}

// CommonService holds shared CAN infrastructure and adapter selection state.
// It is the first service registered and provides shared instances to other services.
type CommonService struct {
	mu          sync.Mutex
	backend     canhal.Backend
	rawBackend  canhal.Backend
	dispatcher  *candispatcher.Dispatcher
	adapterType canhal.Adapter
	channel     int
	onFrameSent func(*canhal.Frame)
}

// NewCommonService creates the common service with default adapter.
func NewCommonService() *CommonService {
	s := &CommonService{}
	// Initialize default adapter immediately (synchronous, runs in main thread)
	adapters := canhal.AvailableAdapters()
	if len(adapters) > 0 {
		s.SetAdapterType(int(adapters[0]))
	}
	return s
}

// ServiceStartup is called by Wails v3 when the application starts.
// Already initialized in constructor, but this satisfies the interface.
func (s *CommonService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	return nil
}

// ServiceShutdown is called by Wails v3 when the application exits.
func (s *CommonService) ServiceShutdown() error {
	if s.backend != nil {
		s.backend.Close()
	}
	return nil
}

// SetAdapterType switches the CAN adapter. Must be called when disconnected.
func (s *CommonService) SetAdapterType(adapterType int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	at := canhal.Adapter(adapterType)
	if at < 0 || at >= canhal.AdapterCount {
		return fmt.Errorf("invalid adapter type: %d", adapterType)
	}

	backend := canhal.NewBackend(at)
	if backend == nil {
		return fmt.Errorf("adapter type %d not available on this platform", adapterType)
	}

	dispatcher := candispatcher.New(backend)
	wrapped := &notifyingBackend{Backend: backend, onWrite: s.onFrameSent}

	s.adapterType = at
	s.rawBackend = backend
	s.backend = wrapped
	s.dispatcher = dispatcher

	return nil
}

// SetConnectedChannel stores the currently connected CAN channel.
func (s *CommonService) SetConnectedChannel(ch int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channel = ch
}

// GetConnectedChannel returns the currently connected CAN channel, or -1.
func (s *CommonService) GetConnectedChannel() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.channel
}

// CreateManager creates a new CAN Manager using the shared backend.
func (s *CommonService) CreateManager() *canmanager.Manager {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend == nil || s.dispatcher == nil {
		return nil
	}
	return canmanager.New(s.backend, s.dispatcher)
}

// CreateCommand creates a new CAN Command using the shared backend.
func (s *CommonService) CreateCommand() *cancommand.Command {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend == nil || s.dispatcher == nil {
		return nil
	}
	return cancommand.New(s.backend, s.dispatcher)
}

// setOnFrameSent registers a callback invoked after every backend.Write.
// Used by the simulator to see frames sent by the PC tool.
func (s *CommonService) setOnFrameSent(cb func(*canhal.Frame)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onFrameSent = cb
	if nb, ok := s.backend.(*notifyingBackend); ok {
		nb.onWrite = cb
	}
}
