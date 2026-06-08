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
	channel int
}

// NewCANCommandService creates a new CAN command service.
func NewCANCommandService(common *CommonService) *CANCommandService {
	return &CANCommandService{common: common, channel: -1}
}

// ServiceStartup is called when the Wails app starts.
func (s *CANCommandService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	s.app = application.Get()

	s.cmd = s.common.CreateCommand()
	if s.cmd != nil {
		s.cmd.SetFrameCallback(func(ev cancommand.FrameEvent) {
			if s.app != nil {
				// Convert []byte to []int to avoid base64 encoding in JSON
				data := make([]int, len(ev.Data))
				for i, b := range ev.Data {
					data[i] = int(b)
				}
				s.app.Event.Emit("can:frame", map[string]interface{}{
					"id":   ev.ID,
					"data": data,
					"dlc":  ev.DLC,
					"isTx": ev.IsTX,
				})
			}
		})
	}

	// Create CAN manager for LoRa config response parsing
	s.canMgr = s.common.CreateManager()

	return nil
}

// SetChannel syncs the CAN channel from the firmware upgrade page.
func (s *CANCommandService) SetChannel(channel int) {
	s.channel = channel
	if s.cmd != nil {
		s.cmd.SetChannel(channel)
	}
	if s.canMgr != nil {
		s.canMgr.Connect(channel, 0)
	}
}

// GetChannel returns the current CAN channel, or -1 if not connected.
func (s *CANCommandService) GetChannel() int {
	if s.channel >= 0 {
		return s.channel
	}
	if s.common != nil {
		return s.common.GetConnectedChannel()
	}
	return -1
}

// ensureChannel auto-initializes channel from shared state if not yet set.
func (s *CANCommandService) ensureChannel() {
	if s.channel < 0 && s.common != nil {
		if ch := s.common.GetConnectedChannel(); ch >= 0 {
			s.SetChannel(ch)
		}
	}
}

// SendFrame sends a CAN frame.
func (s *CANCommandService) SendFrame(canID uint32, data []byte, dlc int, isExtended, isRemote bool) error {
	if s.cmd == nil {
		return fmt.Errorf("CAN command module not initialized")
	}
	s.ensureChannel()
	return s.cmd.SendFrame(canID, data, dlc, isExtended, isRemote)
}

// StartMonitor starts the CAN bus monitor.
func (s *CANCommandService) StartMonitor() error {
	if s.cmd == nil {
		return fmt.Errorf("CAN command module not initialized")
	}
	// Auto-initialize channel from shared state if not yet set
	if s.channel < 0 && s.common != nil {
		if ch := s.common.GetConnectedChannel(); ch >= 0 {
			s.SetChannel(ch)
		}
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
	s.ensureChannel()
	dataBytes := parseHexBytes(dataHex)
	var frameData [8]byte
	frameData[0] = byte(cmd)
	copy(frameData[1:], dataBytes)
	return s.cmd.SendFrame(0x105, frameData[:], 8, false, false)
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
