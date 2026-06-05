// Package cancommand provides CAN frame send/receive, bus monitoring,
// and quick command functionality.
package cancommand

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/kabirz/modhandlergo/internal/canhal"
	"github.com/kabirz/modhandlergo/internal/candispatcher"
	"github.com/kabirz/modhandlergo/internal/canmanager"
)

// QuickCommand represents a preset CAN frame for one-click sending.
type QuickCommand struct {
	CanID      uint32  `json:"canId"`
	Data       [8]byte `json:"data"`
	DLC        uint8   `json:"dlc"`
	IsExtended bool    `json:"isExtended"`
	IsRemote   bool    `json:"isRemote"`
	Name       string  `json:"name"`
}

// FrameEvent represents a CAN frame for frontend display.
type FrameEvent struct {
	ID   uint32 `json:"id"`
	Data []byte `json:"data"`
	DLC  int    `json:"dlc"`
	IsTX bool   `json:"isTx"`
}

// Command manages CAN frame sending, bus monitoring, and quick commands.
type Command struct {
	mu          sync.Mutex
	backend     canhal.Backend
	dispatcher  *candispatcher.Dispatcher
	channel     int
	monitoring  atomic.Bool
	unsub       func()
	onFrame     func(FrameEvent)

	quickCommands []QuickCommand
}

// Default quick commands (matching C code's g_defaultCommands)
var defaultQuickCommands = []QuickCommand{
	{CanID: 0x101, Data: [8]byte{0x00, 0x00, 0x00, 0x00}, DLC: 8, Name: "Start Upgrade"},
	{CanID: 0x101, Data: [8]byte{0x01, 0x00, 0x00, 0x00}, DLC: 8, Name: "Confirm"},
	{CanID: 0x101, Data: [8]byte{0x02, 0x00, 0x00, 0x00}, DLC: 8, Name: "Query Version"},
	{CanID: 0x101, Data: [8]byte{0x03, 0x00, 0x00, 0x00}, DLC: 8, Name: "Reboot"},
}

// New creates a new CAN Command module.
func New(backend canhal.Backend, dispatcher *candispatcher.Dispatcher) *Command {
	cmd := &Command{
		backend:       backend,
		dispatcher:    dispatcher,
		channel:       canhal.InvalidChannel,
		quickCommands: make([]QuickCommand, len(defaultQuickCommands)),
	}
	copy(cmd.quickCommands, defaultQuickCommands)
	return cmd
}

// SetChannel sets the active CAN channel (synced from firmware upgrade page).
func (c *Command) SetChannel(channel int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channel = channel
}

// SetFrameCallback sets the callback for received/sent frames.
func (c *Command) SetFrameCallback(cb func(FrameEvent)) {
	c.onFrame = cb
}

// SendFrame sends a CAN frame with the given parameters.
func (c *Command) SendFrame(canID uint32, data []byte, dlc int, isExtended, isRemote bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel == canhal.InvalidChannel {
		return fmt.Errorf("CAN not connected")
	}
	if c.backend == nil {
		return fmt.Errorf("CAN HAL not initialized")
	}

	var frameData [8]byte
	if data != nil && dlc > 0 {
		copy(frameData[:], data)
	}

	frame := &canhal.Frame{
		ID:   canID,
		DLC:  uint8(dlc),
		Data: frameData,
	}
	if isExtended {
		frame.Flags = canhal.FlagExtended
	}
	if isRemote {
		frame.Flags |= canhal.FlagRemote
	}

	if err := c.backend.Write(frame); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}

	// Notify callback about TX frame
	if c.onFrame != nil {
		c.onFrame(FrameEvent{
			ID:   canID,
			Data: append([]byte{}, frameData[:dlc]...),
			DLC:  dlc,
			IsTX: true,
		})
	}

	return nil
}

// StartMonitor subscribes to CAN frames via the dispatcher.
func (c *Command) StartMonitor() {
	if c.monitoring.Load() || c.dispatcher == nil {
		return
	}
	c.monitoring.Store(true)
	c.unsub = c.dispatcher.Subscribe(func(frame *canhal.Frame) {
		if c.onFrame != nil && c.monitoring.Load() {
			c.onFrame(FrameEvent{
				ID:   frame.ID,
				Data: append([]byte{}, frame.Data[:frame.DLC]...),
				DLC:  int(frame.DLC),
				IsTX: false,
			})
		}
	})
}

// StopMonitor unsubscribes from CAN frames.
func (c *Command) StopMonitor() {
	if !c.monitoring.Load() {
		return
	}
	c.monitoring.Store(false)
	if c.unsub != nil {
		c.unsub()
		c.unsub = nil
	}
}

// IsMonitoring returns whether the bus monitor is active.
func (c *Command) IsMonitoring() bool {
	return c.monitoring.Load()
}

// GetQuickCommands returns the list of quick commands.
func (c *Command) GetQuickCommands() []QuickCommand {
	return c.quickCommands
}

// SetQuickCommands replaces the quick commands list.
func (c *Command) SetQuickCommands(cmds []QuickCommand) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.quickCommands = cmds
}

// ReplaceBackend swaps the CAN backend and dispatcher.
func (c *Command) ReplaceBackend(backend canhal.Backend, dispatcher *candispatcher.Dispatcher) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.monitoring.Load() {
		c.StopMonitor()
	}
	c.backend = backend
	c.dispatcher = dispatcher
}

// FrameIDLabel returns a human-readable label for known CAN frame IDs.
func FrameIDLabel(id uint32) string {
	labels := map[uint32]string{
		canmanager.PlatformRx:     "Control Command",
		canmanager.PlatformTx:     "Response Frame",
		canmanager.FWDataRx:       "Firmware Data",
		canmanager.Heartbeat:      "Heartbeat",
		canmanager.ControllerState: "Controller State",
		canmanager.LaserRanging:   "Laser Ranging",
		canmanager.CoordXY:        "X/Y Coordinates",
		canmanager.CoordZ:         "Z Coordinate",
		canmanager.LoraConfigCmd:  "LoRa Config",
		canmanager.LoraConfigResp: "LoRa Config Response",
	}
	if label, ok := labels[id]; ok {
		return label
	}
	return ""
}
