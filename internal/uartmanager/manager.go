// Package uartmanager provides serial port firmware upgrade functionality.
package uartmanager

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	serial "go.bug.st/serial"

	"github.com/kabirz/modhandlergo/internal/upgrade"
)

// Manager handles UART serial port firmware upgrade operations.
type Manager struct {
	mu       sync.Mutex
	port     serial.Port
	portName string
	baudRate int

	onLog      func(string)
	onProgress func(int)

	// Reusable read buffer for waitForResponse (avoids per-call allocation).
	readBuf [256]byte
}

// New creates a new UART Manager.
func New() *Manager {
	return &Manager{
		baudRate: 115200,
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

func (m *Manager) progress(pct int) {
	if m.onProgress != nil {
		m.onProgress(pct)
	}
}

// --- upgrade.Transport implementation ---

// SendCommand implements upgrade.Transport for UART.
func (m *Manager) SendCommand(cmd uint32, param uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port == nil {
		return fmt.Errorf("serial port not connected")
	}

	cmdData := upgrade.EncodeCommand(cmd, param)
	frame := make([]byte, 32)
	frameLen, err := BuildFrame(FrameTypeCmd, cmdData[:], frame)
	if err != nil {
		return fmt.Errorf("build command frame failed: %w", err)
	}

	m.port.ResetInputBuffer()
	if _, err := m.port.Write(frame[:frameLen]); err != nil {
		return fmt.Errorf("send command failed: %w", err)
	}
	return nil
}

// SendData implements upgrade.Transport for UART.
func (m *Manager) SendData(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port == nil {
		return fmt.Errorf("serial port not connected")
	}

	frame := make([]byte, 32)
	frameLen, err := BuildFrame(FrameTypeData, data, frame)
	if err != nil {
		return fmt.Errorf("build data frame failed: %w", err)
	}

	if _, err := m.port.Write(frame[:frameLen]); err != nil {
		return fmt.Errorf("send data failed: %w", err)
	}
	return nil
}

// WaitForResponse implements upgrade.Transport for UART.
// Uses the reusable readBuf field to avoid per-call allocation.
func (m *Manager) WaitForResponse(timeout time.Duration) (uint32, uint32, error) {
	m.mu.Lock()
	if m.port == nil {
		m.mu.Unlock()
		return 0, 0, fmt.Errorf("serial port not connected")
	}
	m.mu.Unlock()

	var bufPos int
	deadline := time.Now().Add(timeout)
	buf := m.readBuf[:]

	for time.Now().Before(deadline) {
		m.mu.Lock()
		if m.port == nil {
			m.mu.Unlock()
			return 0, 0, fmt.Errorf("serial port not connected")
		}
		m.port.SetReadTimeout(100 * time.Millisecond)
		n, readErr := m.port.Read(buf[bufPos:])
		m.mu.Unlock()

		if readErr != nil {
			continue
		}
		if n > 0 {
			bufPos += n

			fType, data, consumed, parseErr := ParseFrame(buf[:bufPos])
			if parseErr != nil {
				if consumed < 0 {
					discard := -consumed
					copy(buf, buf[discard:bufPos])
					bufPos -= discard
				}
				continue
			}
			if consumed > 0 {
				_ = fType
				if len(data) == 8 {
					code, val := DecodeResponse(data)
					return code, val, nil
				}
				copy(buf, buf[consumed:bufPos])
				bufPos -= consumed
			}
		}
	}

	return 0, 0, fmt.Errorf("timeout")
}

// IsConnected implements upgrade.Transport for UART.
func (m *Manager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.port != nil
}

// --- Connection management ---

// EnumPorts returns a list of available serial ports.
func (m *Manager) EnumPorts() ([]SerialPortInfo, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("enum serial ports failed: %w", err)
	}

	var result []SerialPortInfo
	for _, p := range ports {
		pLower := strings.ToLower(p)
		if strings.Contains(pLower, "bluetooth") {
			continue
		}

		name := p
		if idx := strings.LastIndex(p, "/"); idx >= 0 {
			name = p[idx+1:]
		} else if idx := strings.LastIndex(p, "\\"); idx >= 0 {
			name = p[idx+1:]
		}

		result = append(result, SerialPortInfo{
			PortName:    p,
			FriendlyName: name,
		})
	}

	m.log(fmt.Sprintf("Found %d available serial ports", len(result)))
	return result, nil
}

// Connect opens a serial port connection.
func (m *Manager) Connect(portName string, baudRate int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port != nil {
		m.log("Serial port already connected")
		return nil
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("cannot open serial port %s: %w", portName, err)
	}

	m.port = port
	m.portName = portName
	m.baudRate = baudRate

	m.log(fmt.Sprintf("Serial port %s connected (%d bps)", portName, baudRate))
	return nil
}

// Disconnect closes the serial port connection.
func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port != nil {
		m.port.Close()
		m.log(fmt.Sprintf("Serial port %s disconnected", m.portName))
		m.port = nil
	}
}

// GetFirmwareVersion queries the firmware version via UART.
func (m *Manager) GetFirmwareVersion() (uint32, error) {
	m.mu.Lock()
	if m.port == nil {
		m.mu.Unlock()
		return 0, fmt.Errorf("serial port not connected")
	}
	m.mu.Unlock()

	version, err := upgrade.GetVersion(m)
	if err != nil {
		return 0, err
	}
	m.log(fmt.Sprintf("Firmware version: %s", upgrade.FormatVersion(version)))
	return version, nil
}

// BoardReboot sends a reboot command via UART.
func (m *Manager) BoardReboot() error {
	m.mu.Lock()
	if m.port == nil {
		m.mu.Unlock()
		return fmt.Errorf("serial port not connected")
	}
	m.mu.Unlock()

	return upgrade.Reboot(m)
}

// FirmwareUpgrade performs a complete firmware upgrade over UART.
func (m *Manager) FirmwareUpgrade(filePath string, testMode bool) error {
	m.mu.Lock()
	if m.port == nil {
		m.mu.Unlock()
		return fmt.Errorf("serial port not connected")
	}
	m.mu.Unlock()

	return upgrade.RunUpgrade(m, filePath, testMode, 0xFF, &upgrade.Callbacks{
		OnLog:      m.onLog,
		OnProgress: m.onProgress,
	})
}

// OpenFirmwareFile opens a native file dialog to select a firmware file.
// Deprecated: use CANUpgradeService.OpenFirmwareFile instead.
func OpenFirmwareFile() (string, error) {
	return "", fmt.Errorf("not implemented: use CANUpgradeService.OpenFirmwareFile")
}

// DecodeResponse parses a UART response frame's data payload into code+val.
// Kept for backward compatibility with any external callers.
func decodeResponseLocal(data []byte) (code uint32, val uint32) {
	return DecodeResponse(data)
}

// ensurePort checks if the port is open and returns an error if not.
func (m *Manager) ensurePort() error {
	if m.port == nil {
		return fmt.Errorf("serial port not connected")
	}
	return nil
}

// Read is a helper for external callers that need raw serial read (e.g., diagnostics).
func (m *Manager) Read(buf []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.port == nil {
		return 0, io.ErrClosedPipe
	}
	return m.port.Read(buf)
}
