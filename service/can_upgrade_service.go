package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kabirz/modhandlergo/internal/canhal"
	"github.com/kabirz/modhandlergo/internal/canmanager"
	"github.com/kabirz/modhandlergo/internal/uartmanager"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// CANUpgradeService provides firmware upgrade operations for the frontend.
type CANUpgradeService struct {
	app      *application.App
	common   *CommonService
	canMgr   *canmanager.Manager
	uartMgr  *uartmanager.Manager
}

// NewCANUpgradeService creates a new firmware upgrade service.
func NewCANUpgradeService(common *CommonService) *CANUpgradeService {
	return &CANUpgradeService{
		common:  common,
		uartMgr: uartmanager.New(),
	}
}

// ServiceStartup is called when the Wails app starts.
func (s *CANUpgradeService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	s.app = application.Get()

	// Wire callbacks to emit events
	s.uartMgr.SetLogCallback(func(msg string) {
		if s.app != nil {
			s.app.Event.Emit("uart:log", fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05.000"), msg))
		}
	})
	s.uartMgr.SetProgressCallback(func(pct int) {
		if s.app != nil {
			s.app.Event.Emit("uart:progress", pct)
		}
	})

	return nil
}

// --- Adapter & Channel ---

// DetectCANDevices returns a list of available CAN device channels.
func (s *CANUpgradeService) DetectCANDevices() ([]int, error) {
	mgr := s.ensureCANManager()
	if mgr == nil {
		return nil, fmt.Errorf("CAN HAL 未初始化")
	}
	return mgr.DetectDevices()
}

// --- CAN Operations ---

// ConnectCAN connects to a CAN device.
func (s *CANUpgradeService) ConnectCAN(channel int, baudIndex int) error {
	mgr := s.ensureCANManager()
	if mgr == nil {
		return fmt.Errorf("CAN HAL 未初始化")
	}
	mgr.SetLogCallback(func(msg string) {
		if s.app != nil {
			s.app.Event.Emit("can:log", fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05.000"), msg))
		}
	})
	mgr.SetProgressCallback(func(pct int) {
		if s.app != nil {
			s.app.Event.Emit("can:progress", pct)
		}
	})
	if err := mgr.Connect(channel, canhal.BaudRate(baudIndex)); err != nil {
		return err
	}
	// Notify other pages (e.g. CAN Command) about the new CAN connection
	if s.app != nil {
		s.app.Event.Emit("can:connected", channel)
	}
	return nil
}

// DisconnectCAN disconnects from CAN.
func (s *CANUpgradeService) DisconnectCAN() {
	if s.canMgr != nil {
		s.canMgr.Disconnect()
	}
	// Notify other pages that CAN has been disconnected
	if s.app != nil {
		s.app.Event.Emit("can:disconnected", 0)
	}
}

// CANFirmwareUpgrade starts a firmware upgrade over CAN.
func (s *CANUpgradeService) CANFirmwareUpgrade(filePath string, testMode bool) error {
	mgr := s.ensureCANManager()
	if mgr == nil {
		return fmt.Errorf("CAN HAL 未初始化")
	}
	return mgr.FirmwareUpgrade(filePath, testMode)
}

// CANGetFirmwareVersion queries the firmware version over CAN.
func (s *CANUpgradeService) CANGetFirmwareVersion() (string, error) {
	mgr := s.ensureCANManager()
	if mgr == nil {
		return "", fmt.Errorf("CAN HAL 未初始化")
	}
	version, err := mgr.GetFirmwareVersion()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d.%d.%d", (version>>24)&0xFF, (version>>16)&0xFF, (version>>8)&0xFF), nil
}

// CANBoardReboot sends a reboot command over CAN.
func (s *CANUpgradeService) CANBoardReboot() error {
	mgr := s.ensureCANManager()
	if mgr == nil {
		return fmt.Errorf("CAN HAL 未初始化")
	}
	return mgr.BoardReboot()
}

// --- UART Operations ---

// DetectSerialPorts returns available serial ports.
func (s *CANUpgradeService) DetectSerialPorts() ([]uartmanager.SerialPortInfo, error) {
	return s.uartMgr.EnumPorts()
}

// ConnectUART opens a serial port.
func (s *CANUpgradeService) ConnectUART(portName string, baudRate int) error {
	return s.uartMgr.Connect(portName, baudRate)
}

// DisconnectUART closes the serial port.
func (s *CANUpgradeService) DisconnectUART() {
	s.uartMgr.Disconnect()
}

// UARTFirmwareUpgrade starts a firmware upgrade over UART.
func (s *CANUpgradeService) UARTFirmwareUpgrade(filePath string, testMode bool) error {
	return s.uartMgr.FirmwareUpgrade(filePath, testMode)
}

// UARTGetFirmwareVersion queries firmware version over UART.
func (s *CANUpgradeService) UARTGetFirmwareVersion() (string, error) {
	version, err := s.uartMgr.GetFirmwareVersion()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d.%d.%d", (version>>24)&0xFF, (version>>16)&0xFF, (version>>8)&0xFF), nil
}

// UARTBoardReboot sends a reboot command over UART.
func (s *CANUpgradeService) UARTBoardReboot() error {
	return s.uartMgr.BoardReboot()
}

func (s *CANUpgradeService) ensureCANManager() *canmanager.Manager {
	if s.canMgr == nil {
		s.canMgr = s.common.CreateManager()
	}
	return s.canMgr
}

// OpenFirmwareFile opens a native file dialog to select a firmware file.
// Returns the selected file path, or empty string if cancelled.
func (s *CANUpgradeService) OpenFirmwareFile() (string, error) {
	if s.app == nil {
		return "", fmt.Errorf("app not initialized")
	}
	dialog := s.app.Dialog.OpenFile()
	dialog.SetTitle("选择固件文件")
	dialog.AddFilter("固件文件 (*.bin)", "*.bin")
	dialog.AddFilter("所有文件 (*.*)", "*.*")
	result, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return result, nil
}
