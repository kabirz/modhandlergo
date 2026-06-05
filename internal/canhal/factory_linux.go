//go:build linux

package canhal

import (
	"time"

	"github.com/kabirz/modhandlergo/internal/canhal/socketcan"
)

// socketcanAdapter wraps socketcan.Backend to satisfy canhal.Backend interface.
type socketcanAdapter struct {
	inner *socketcan.Backend
}

func (a *socketcanAdapter) Name() string                          { return a.inner.Name() }
func (a *socketcanAdapter) DetectDevices() ([]int, error)         { return a.inner.DetectDevices() }
func (a *socketcanAdapter) IsConnected() bool                     { return a.inner.IsConnected() }
func (a *socketcanAdapter) Channel() int                          { return a.inner.Channel() }
func (a *socketcanAdapter) Close() error                          { return a.inner.Close() }
func (a *socketcanAdapter) Disconnect() error                     { return a.inner.Disconnect() }

func (a *socketcanAdapter) Connect(channel int, baud BaudRate) error {
	return a.inner.Connect(channel, socketcan.BaudRate(baud))
}

func (a *socketcanAdapter) Write(frame *Frame) error {
	sf := &socketcan.Frame{ID: frame.ID, DLC: frame.DLC, Data: frame.Data}
	sf.Flags = socketcan.FrameFlags(frame.Flags)
	return a.inner.Write(sf)
}

func (a *socketcanAdapter) Read(timeout time.Duration) (*Frame, error) {
	sf, err := a.inner.Read(timeout)
	if err != nil || sf == nil {
		return nil, err
	}
	return &Frame{ID: sf.ID, DLC: sf.DLC, Data: sf.Data, Flags: FrameFlags(sf.Flags)}, nil
}

func (a *socketcanAdapter) SetFilter(fromID, toID uint32) error {
	return a.inner.SetFilter(fromID, toID)
}

// NewBackend creates a CAN backend for the current platform.
func NewBackend(adapter Adapter) Backend {
	switch adapter {
	case AdapterSocketCAN:
		return &socketcanAdapter{inner: socketcan.New()}
	default:
		return nil
	}
}

// AvailableAdapters returns the list of adapters available on this platform.
func AvailableAdapters() []Adapter {
	return []Adapter{AdapterSocketCAN}
}
