package canmanager

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kabirz/modhandlergo/internal/candispatcher"
	"github.com/kabirz/modhandlergo/internal/canhal"
	"github.com/kabirz/modhandlergo/internal/upgrade"
)

// Manager handles CAN firmware upgrade operations: connect, version query,
// board reboot, and OTA firmware flashing.
type Manager struct {
	mu          sync.Mutex
	backend     canhal.Backend
	dispatcher  *candispatcher.Dispatcher
	channel     int
	virtualMode bool

	onLog      func(string)
	onProgress func(int)
}

// New creates a new CAN Manager.
func New(backend canhal.Backend, dispatcher *candispatcher.Dispatcher) *Manager {
	return &Manager{
		backend:    backend,
		dispatcher: dispatcher,
		channel:    canhal.InvalidChannel,
	}
}

// SetLogCallback sets the callback for log messages.
func (m *Manager) SetLogCallback(cb func(string)) {
	m.onLog = cb
}

// SetProgressCallback sets the callback for progress updates (0-100).
func (m *Manager) SetProgressCallback(cb func(int)) {
	m.onProgress = cb
}

func (m *Manager) log(msg string) {
	if m.onLog != nil {
		m.onLog(msg)
	}
}

// --- upgrade.Transport implementation ---

// SendCommand implements upgrade.Transport for CAN.
func (m *Manager) SendCommand(cmd uint32, param uint32) error {
	var data [8]byte
	EncodeCANFrameData(CANFrameData{Code: cmd, Val: param}, data[:])

	frame := &canhal.Frame{
		ID:    PlatformRx,
		DLC:   8,
		Data:  data,
		Flags: canhal.FlagStandard,
	}
	return m.backend.Write(frame)
}

// SendData implements upgrade.Transport for CAN.
func (m *Manager) SendData(data []byte) error {
	var frameData [8]byte
	copy(frameData[:], data)

	frame := &canhal.Frame{
		ID:    FWDataRx,
		DLC:   8,
		Data:  frameData,
		Flags: canhal.FlagStandard,
	}
	return m.backend.Write(frame)
}

// WaitForResponse implements upgrade.Transport for CAN.
func (m *Manager) WaitForResponse(timeout time.Duration) (uint32, uint32, error) {
	if m.dispatcher == nil {
		return 0, 0, fmt.Errorf("dispatcher not initialized")
	}

	frame, err := m.dispatcher.WaitFrame(PlatformTx, timeout)
	if err != nil {
		return 0, 0, err
	}

	d := DecodeCANFrameData(frame.Data[:])
	return d.Code, d.Val, nil
}

// IsConnected implements upgrade.Transport for CAN.
func (m *Manager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.channel != canhal.InvalidChannel
}

// --- Connection management ---

// requireConnected checks if CAN is connected, returning an error if not.
func (m *Manager) requireConnected() error {
	if m.channel == canhal.InvalidChannel {
		return fmt.Errorf("CAN disconnected, please reconnect")
	}
	return nil
}

// Connect establishes a CAN connection on the given channel and baud rate.
func (m *Manager) Connect(channel int, baud canhal.BaudRate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.channel != canhal.InvalidChannel {
		m.log("CAN connection already exists")
		return nil
	}

	if channel == VirtualChannel {
		m.channel = channel
		m.virtualMode = true
		m.log("Virtual CAN connected (test mode)")
		return nil
	}

	if m.backend == nil {
		return fmt.Errorf("CAN HAL not initialized")
	}

	if err := m.backend.Connect(channel, baud); err != nil {
		return fmt.Errorf("CAN init failed: %w", err)
	}

	m.channel = channel
	m.virtualMode = false
	m.log(fmt.Sprintf("CAN(id=%xh) connected (%s)", channel, m.backend.Name()))

	// Start dispatcher read thread — accepts ALL frames
	if m.dispatcher != nil {
		m.dispatcher.Start()
	}

	return nil
}

// Disconnect closes the CAN connection.
func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dispatcher != nil {
		m.dispatcher.Stop()
	}

	if m.virtualMode {
		m.log("Virtual CAN disconnected")
	} else if m.channel != canhal.InvalidChannel {
		m.log(fmt.Sprintf("CAN(id=%xh) disconnected", m.channel))
		if m.backend != nil {
			m.backend.Disconnect()
		}
	}
	m.channel = canhal.InvalidChannel
	m.virtualMode = false
}

// GetFirmwareVersion queries the board firmware version via CAN.
func (m *Manager) GetFirmwareVersion() (uint32, error) {
	m.mu.Lock()
	err := m.requireConnected()
	if err != nil {
		m.mu.Unlock()
		return 0, err
	}
	if m.virtualMode {
		m.mu.Unlock()
		m.log("Firmware version: v1.0.0 (virtual CAN)")
		return 0x01000000, nil
	}
	m.mu.Unlock()

	version, err := upgrade.GetVersion(m)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// BoardReboot sends a reboot command to the board.
func (m *Manager) BoardReboot() error {
	m.mu.Lock()
	err := m.requireConnected()
	if err != nil {
		m.mu.Unlock()
		return err
	}
	if m.virtualMode {
		m.mu.Unlock()
		m.log("Virtual board reboot successful")
		return nil
	}
	m.mu.Unlock()

	return upgrade.Reboot(m)
}

// FirmwareUpgrade performs a complete firmware upgrade over CAN.
func (m *Manager) FirmwareUpgrade(filePath string, testMode bool) error {
	m.mu.Lock()
	err := m.requireConnected()
	virtualMode := m.virtualMode
	m.mu.Unlock()

	if err != nil {
		return err
	}

	if virtualMode {
		return m.virtualFirmwareUpgrade(filePath)
	}

	return upgrade.RunUpgrade(m, filePath, testMode, 0x00, &upgrade.Callbacks{
		OnLog:      m.onLog,
		OnProgress: m.onProgress,
	})
}

// DetectDevices delegates to the CAN backend to detect available devices.
func (m *Manager) DetectDevices() ([]int, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("CAN HAL not initialized")
	}
	return m.backend.DetectDevices()
}

func (m *Manager) virtualFirmwareUpgrade(filePath string) error {
	m.log("Virtual CAN mode: simulating firmware upgrade...")

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open source firmware file")
	}
	defer f.Close()

	fi, _ := f.Stat()
	fileSize := fi.Size()
	m.log(fmt.Sprintf("Starting firmware upgrade, size: %d bytes", fileSize))
	m.log("Output file: virtual_firmware.bin")

	time.Sleep(500 * time.Millisecond)
	m.log("Flash erase complete")

	buf := make([]byte, 4096)
	var totalBytes int64
	for {
		n, err := f.Read(buf)
		if n == 0 || err != nil {
			break
		}
		totalBytes += int64(n)
		if totalBytes%64 == 0 || totalBytes == fileSize {
			m.progress(int(totalBytes * 100 / fileSize))
		}
	}

	time.Sleep(200 * time.Millisecond)
	m.log("Firmware send complete")
	time.Sleep(200 * time.Millisecond)
	m.log("Firmware confirm complete")

	m.progress(100)
	m.log(fmt.Sprintf("Virtual firmware saved (%d bytes)", totalBytes))
	return nil
}

func (m *Manager) progress(pct int) {
	if m.onProgress != nil {
		m.onProgress(pct)
	}
}
