package service

import (
	"context"
	"fmt"

	"github.com/kabirz/modhandlergo/internal/cancommand"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// CANCommandService provides CAN frame send/receive and bus monitoring for the frontend.
type CANCommandService struct {
	app     *application.App
	common  *CommonService
	cmd     *cancommand.Command
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
	return nil
}

// SetChannel syncs the CAN channel from the firmware upgrade page.
func (s *CANCommandService) SetChannel(channel int) {
	if s.cmd != nil {
		s.cmd.SetChannel(channel)
	}
}

// SendFrame sends a CAN frame.
func (s *CANCommandService) SendFrame(canID uint32, data []byte, dlc int, isExtended, isRemote bool) error {
	if s.cmd == nil {
		return fmt.Errorf("CAN 命令模块未初始化")
	}
	return s.cmd.SendFrame(canID, data, dlc, isExtended, isRemote)
}

// SendQuickCommand sends a quick command by index.
func (s *CANCommandService) SendQuickCommand(index int) error {
	if s.cmd == nil {
		return fmt.Errorf("CAN 命令模块未初始化")
	}
	cmds := s.cmd.GetQuickCommands()
	if index < 0 || index >= len(cmds) {
		return fmt.Errorf("无效的快捷命令索引: %d", index)
	}
	qc := cmds[index]
	return s.cmd.SendFrame(qc.CanID, qc.Data[:], int(qc.DLC), qc.IsExtended, qc.IsRemote)
}

// StartMonitor starts the CAN bus monitor.
func (s *CANCommandService) StartMonitor() error {
	if s.cmd == nil {
		return fmt.Errorf("CAN 命令模块未初始化")
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

// IsMonitoring returns whether the bus monitor is active.
func (s *CANCommandService) IsMonitoring() bool {
	if s.cmd == nil {
		return false
	}
	return s.cmd.IsMonitoring()
}

// GetQuickCommands returns the list of quick commands.
func (s *CANCommandService) GetQuickCommands() []cancommand.QuickCommand {
	if s.cmd == nil {
		return nil
	}
	return s.cmd.GetQuickCommands()
}

// GetFrameLabel returns a human-readable label for a CAN frame ID.
func (s *CANCommandService) GetFrameLabel(id uint32) string {
	return cancommand.FrameIDLabel(id)
}
