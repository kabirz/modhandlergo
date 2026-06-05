//go:build windows

package canhal

import (
	"time"

	"github.com/kabirz/modhandlergo/internal/canhal/pcan"
)

// pcanAdapter wraps pcan.Backend to satisfy canhal.Backend interface.
type pcanAdapter struct {
	inner *pcan.Backend
}

func (a *pcanAdapter) Name() string                          { return a.inner.Name() }
func (a *pcanAdapter) DetectDevices() ([]int, error)         { return a.inner.DetectDevices() }
func (a *pcanAdapter) IsConnected() bool                     { return a.inner.IsConnected() }
func (a *pcanAdapter) Channel() int                          { return a.inner.Channel() }
func (a *pcanAdapter) Close() error                          { return a.inner.Close() }
func (a *pcanAdapter) Disconnect() error                     { return a.inner.Disconnect() }

func (a *pcanAdapter) Connect(channel int, baud BaudRate) error {
	return a.inner.Connect(channel, pcan.BaudRate(baud))
}

func (a *pcanAdapter) Write(frame *Frame) error {
	pf := &pcan.Frame{ID: frame.ID, DLC: frame.DLC, Data: frame.Data}
	pf.Flags = pcan.FrameFlags(frame.Flags)
	return a.inner.Write(pf)
}

func (a *pcanAdapter) Read(timeout time.Duration) (*Frame, error) {
	pf, err := a.inner.Read(timeout)
	if err != nil || pf == nil {
		return nil, err
	}
	return &Frame{ID: pf.ID, DLC: pf.DLC, Data: pf.Data, Flags: FrameFlags(pf.Flags)}, nil
}

func (a *pcanAdapter) SetFilter(fromID, toID uint32) error {
	return a.inner.SetFilter(fromID, toID)
}

// NewBackend creates a CAN backend for the current platform.
func NewBackend(adapter Adapter) Backend {
	switch adapter {
	case AdapterPCAN:
		return &pcanAdapter{inner: pcan.New()}
	default:
		return nil
	}
}

// AvailableAdapters returns the list of adapters available on this platform.
func AvailableAdapters() []Adapter {
	return []Adapter{AdapterPCAN}
}
