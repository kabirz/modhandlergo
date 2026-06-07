package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kabirz/modhandlergo/internal/cancommand"
	"github.com/kabirz/modhandlergo/internal/canmanager"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// CANCommandService provides CAN frame send/receive, bus monitoring,
// and LoRa remote configuration via CAN (0x105/0x106).
type CANCommandService struct {
	app     *application.App
	common  *CommonService
	cmd     *cancommand.Command
	canMgr  *canmanager.Manager
}

// NewCANCommandService creates a new CAN command service.
func NewCANCommandService(common *CommonService) *CANCommandService {
	return &CANCommandService{common: common}
}

// ServiceStartup is called when the Wails app starts.
func (s *CANCommandService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	s.app = application.Get()

	s.cmd = s.common.CreateCommand()
	if s.cmd != nil {
		s.cmd.SetFrameCallback(func(ev cancommand.FrameEvent) {
			if s.app != nil {
				s.app.Event.Emit("can:frame", ev)
			}
		})
	}

	// Create CAN manager for LoRa config response parsing
	s.canMgr = s.common.CreateManager()

	return nil
}

// SetChannel syncs the CAN channel from the firmware upgrade page.
func (s *CANCommandService) SetChannel(channel int) {
	if s.cmd != nil {
		s.cmd.SetChannel(channel)
	}
	if s.canMgr != nil {
		s.canMgr.Connect(channel, 0)
	}
}

// SendFrame sends a CAN frame.
func (s *CANCommandService) SendFrame(canID uint32, data []byte, dlc int, isExtended, isRemote bool) error {
	if s.cmd == nil {
		return fmt.Errorf("CAN command module not initialized")
	}
	return s.cmd.SendFrame(canID, data, dlc, isExtended, isRemote)
}

// StartMonitor starts the CAN bus monitor.
func (s *CANCommandService) StartMonitor() error {
	if s.cmd == nil {
		return fmt.Errorf("CAN command module not initialized")
	}
	s.cmd.StartMonitor()
	return nil
}

// StopMonitor stops the CAN bus monitor.
func (s *CANCommandService) StopMonitor() {
	if s.cmd != nil {
		s.cmd.StopMonitor()
	}
}

// --- LoRa config via CAN (0x105/0x106) ---

// SendLoraCommand sends a LoRa configuration command via CAN frame 0x105.
// dataHex is a hex-encoded string (e.g. "0102") for the payload bytes after cmd.
// The payload format matches the embedded firmware mod-can.h protocol.
func (s *CANCommandService) SendLoraCommand(cmd int, dataHex string) error {
	if s.cmd == nil {
		return fmt.Errorf("CAN command module not initialized")
	}
	dataBytes := parseHexBytes(dataHex)
	payload := append([]byte{byte(cmd)}, dataBytes...)
	return s.cmd.SendFrame(0x105, payload, len(payload), false, false)
}

// parseHexBytes converts a hex string like "010203" to []byte.
func parseHexBytes(hex string) []byte {
	var result []byte
	for i := 0; i+1 < len(hex); i += 2 {
		b, err := strconv.ParseUint(hex[i:i+2], 16, 8)
		if err == nil {
			result = append(result, byte(b))
		}
	}
	return result
}
