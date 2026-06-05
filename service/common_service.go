package service

import (
	"fmt"
	"sync"

	"github.com/kabirz/modhandlergo/internal/canhal"
	"github.com/kabirz/modhandlergo/internal/candispatcher"
	"github.com/kabirz/modhandlergo/internal/canmanager"
	"github.com/kabirz/modhandlergo/internal/cancommand"
)

// AdapterInfo describes a CAN adapter type for the frontend.
type AdapterInfo struct {
	Type  int    `json:"type"`
	Name  string `json:"name"`
	Available bool `json:"available"`
}

// CommonService holds shared CAN infrastructure and adapter selection state.
// It is the first service registered and provides shared instances to other services.
type CommonService struct {
	mu         sync.Mutex
	backend    canhal.Backend
	dispatcher *candispatcher.Dispatcher
	adapterType canhal.Adapter
}

// NewCommonService creates the common service with default adapter.
func NewCommonService() *CommonService {
	return &CommonService{}
}

// GetAdapterType returns the current adapter type.
func (s *CommonService) GetAdapterType() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int(s.adapterType)
}

// SetAdapterType switches the CAN adapter. Must be called when disconnected.
func (s *CommonService) SetAdapterType(adapterType int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	at := canhal.Adapter(adapterType)
	if at < 0 || at >= canhal.AdapterCount {
		return fmt.Errorf("invalid adapter type: %d", adapterType)
	}

	// Create new backend
	backend := canhal.NewBackend(at)
	if backend == nil {
		return fmt.Errorf("adapter type %d not available on this platform", adapterType)
	}

	// Create new dispatcher
	dispatcher := candispatcher.New(backend)

	s.adapterType = at
	s.backend = backend
	s.dispatcher = dispatcher

	return nil
}

// GetBackend returns the current CAN backend.
func (s *CommonService) GetBackend() canhal.Backend {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend
}

// GetDispatcher returns the current dispatcher.
func (s *CommonService) GetDispatcher() *candispatcher.Dispatcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dispatcher
}

// GetAvailableAdapters returns adapters available on this platform.
func (s *CommonService) GetAvailableAdapters() []AdapterInfo {
	adapters := canhal.AvailableAdapters()
	result := make([]AdapterInfo, len(adapters))
	for i, a := range adapters {
		result[i] = AdapterInfo{
			Type:      int(a),
			Name:      adapterName(a),
			Available: true,
		}
	}
	return result
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

func adapterName(a canhal.Adapter) string {
	switch a {
	case canhal.AdapterPCAN:
		return "PCAN"
	case canhal.AdapterSocketCAN:
		return "SocketCAN"
	default:
		return "Unknown"
	}
}

// Startup initializes the default adapter.
func (s *CommonService) Startup() error {
	adapters := canhal.AvailableAdapters()
	if len(adapters) > 0 {
		return s.SetAdapterType(int(adapters[0]))
	}
	return nil
}
