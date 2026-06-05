package uartmanager

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	serial "go.bug.st/serial"
)

// Manager handles UART serial port firmware upgrade operations.
type Manager struct {
	mu       sync.Mutex
	port     serial.Port
	portName string
	baudRate int

	onLog      func(string)
	onProgress func(int)
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

// EnumPorts returns a list of available serial ports.
func (m *Manager) EnumPorts() ([]SerialPortInfo, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("枚举串口失败: %w", err)
	}

	var result []SerialPortInfo
	for _, p := range ports {
		// Filter out Bluetooth ports
		pLower := strings.ToLower(p)
		if strings.Contains(pLower, "bluetooth") {
			continue
		}

		name := p
		// Extract short name for display
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

	m.log(fmt.Sprintf("查询到 %d 个可用串口", len(result)))
	return result, nil
}

// Connect opens a serial port connection.
func (m *Manager) Connect(portName string, baudRate int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port != nil {
		m.log("串口已连接, 请勿重复连接")
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
		return fmt.Errorf("无法打开串口 %s: %w", portName, err)
	}

	m.port = port
	m.portName = portName
	m.baudRate = baudRate

	m.log(fmt.Sprintf("串口 %s 连接成功 (%d bps)", portName, baudRate))
	return nil
}

// Disconnect closes the serial port connection.
func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port != nil {
		m.port.Close()
		m.log(fmt.Sprintf("串口 %s 连接已断开", m.portName))
		m.port = nil
	}
}

// IsConnected returns whether the serial port is open.
func (m *Manager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.port != nil
}

// GetFirmwareVersion queries the firmware version via UART.
func (m *Manager) GetFirmwareVersion() (uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port == nil {
		return 0, fmt.Errorf("串口未连接")
	}

	// Clear receive buffer
	m.port.ResetInputBuffer()

	cmdData := EncodeCommand(CmdVersion, 0)
	frame := make([]byte, 32)
	frameLen, err := BuildFrame(FrameTypeCmd, cmdData[:], frame)
	if err != nil {
		return 0, fmt.Errorf("构建命令帧失败: %w", err)
	}

	if _, err := m.port.Write(frame[:frameLen]); err != nil {
		return 0, fmt.Errorf("发送版本查询命令失败: %w", err)
	}

	m.log("等待版本响应...")

	code, version, err := m.waitForResponse(5 * time.Second)
	if err != nil {
		return 0, fmt.Errorf("读取版本响应超时: %w", err)
	}

	if code == FWCodeVersion {
		m.log(fmt.Sprintf("固件版本: v%d.%d.%d",
			(version>>24)&0xFF, (version>>16)&0xFF, (version>>8)&0xFF))
		return version, nil
	}

	return 0, fmt.Errorf("读取版本响应数据错误")
}

// BoardReboot sends a reboot command via UART.
func (m *Manager) BoardReboot() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.port == nil {
		return fmt.Errorf("串口未连接")
	}

	cmdData := EncodeCommand(CmdReboot, 0)
	frame := make([]byte, 32)
	frameLen, err := BuildFrame(FrameTypeCmd, cmdData[:], frame)
	if err != nil {
		return fmt.Errorf("构建命令帧失败: %w", err)
	}

	if _, err := m.port.Write(frame[:frameLen]); err != nil {
		return fmt.Errorf("发送重启命令失败: %w", err)
	}

	m.log("重启命令已发送")
	return nil
}

// FirmwareUpgrade performs a complete firmware upgrade over UART.
func (m *Manager) FirmwareUpgrade(filePath string, testMode bool) error {
	m.mu.Lock()
	if m.port == nil {
		m.mu.Unlock()
		return fmt.Errorf("串口未连接")
	}
	m.mu.Unlock()

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开文件: %s", filePath)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("无法获取文件信息")
	}
	fileSize := fi.Size()
	m.log(fmt.Sprintf("开始固件升级, 固件大小: %d 字节", fileSize))

	// Send start update command with file size
	m.mu.Lock()
	cmdData := EncodeCommand(CmdStartUpdate, uint32(fileSize))
	frame := make([]byte, 32)
	frameLen, _ := BuildFrame(FrameTypeCmd, cmdData[:], frame)

	if _, err := m.port.Write(frame[:frameLen]); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("发送固件大小失败")
	}
	m.mu.Unlock()

	// Wait for flash erase
	code, offset, err := m.waitForResponse(15 * time.Second)
	if err != nil {
		return fmt.Errorf("Flash擦除超时")
	}
	if code != FWCodeOffset || offset != 0 {
		return fmt.Errorf("Flash擦除失败: code(%d), offset(%d)", code, offset)
	}

	m.log("Flash擦除完成")

	// Send firmware data 8 bytes at a time
	var bytesSent int64
	dataBuf := make([]byte, 8)
	transferComplete := false

	for {
		n, err := f.Read(dataBuf)
		if n == 0 || err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取固件文件失败: %w", err)
		}

		// Pad to 8 bytes
		for i := n; i < 8; i++ {
			dataBuf[i] = 0xFF
		}

		m.mu.Lock()
		frameLen, _ = BuildFrame(FrameTypeData, dataBuf, frame)
		_, writeErr := m.port.Write(frame[:frameLen])
		m.mu.Unlock()

		if writeErr != nil {
			return fmt.Errorf("发送文件数据失败")
		}

		bytesSent += 8

		// Every 64 bytes or at end
		if bytesSent%64 == 0 || bytesSent >= fileSize {
			m.progress(int(bytesSent * 100 / fileSize))

			code, offset, err := m.waitForResponse(5 * time.Second)
			if err != nil {
				return fmt.Errorf("固件更新超时")
			}
			if code == FWCodeSuccess {
				transferComplete = true
				break
			}
			if code != FWCodeOffset {
				return fmt.Errorf("固件升级失败: code(%d), offset(%d)", code, offset)
			}
		}
	}

	m.progress(100)

	// Check transfer completion
	if !transferComplete && bytesSent > 0 {
		code, _, err := m.waitForResponse(5 * time.Second)
		if err != nil {
			return fmt.Errorf("等待固件传输完成超时")
		}
		if code != FWCodeSuccess {
			return fmt.Errorf("固件传输未成功完成: code(%d)", code)
		}
	}

	// Send confirm command
	confirmVal := uint32(1)
	if testMode {
		confirmVal = 0
	}

	m.mu.Lock()
	cmdData = EncodeCommand(CmdConfirm, confirmVal)
	frameLen, _ = BuildFrame(FrameTypeCmd, cmdData[:], frame)
	_, writeErr := m.port.Write(frame[:frameLen])
	m.mu.Unlock()

	if writeErr != nil {
		return fmt.Errorf("发送确认命令失败")
	}

	code, respVal, err := m.waitForResponse(30 * time.Second)
	if err != nil {
		return fmt.Errorf("固件确认超时")
	}

	if code == FWCodeConfirm && respVal == 0x55AA55AA {
		m.log(fmt.Sprintf("文件 %s 上传完成", filePath))
		return nil
	}
	if code == FWCodeTransferErr {
		return fmt.Errorf("固件更新失败")
	}

	return fmt.Errorf("固件确认失败: code(%d)", code)
}

func (m *Manager) waitForResponse(timeout time.Duration) (code uint32, val uint32, err error) {
	buf := make([]byte, 256)
	var bufPos int
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		m.mu.Lock()
		if m.port == nil {
			m.mu.Unlock()
			return 0, 0, fmt.Errorf("串口未连接")
		}
		// Set read deadline to avoid blocking forever
		m.port.SetReadTimeout(100 * time.Millisecond)
		n, readErr := m.port.Read(buf[bufPos:])
		m.mu.Unlock()

		if readErr != nil {
			continue
		}
		if n > 0 {
			bufPos += n

			// Try to parse frame
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
				// Successfully parsed
				_ = fType
				if len(data) == 8 {
					code, val = DecodeResponse(data)
					return code, val, nil
				}
				// Remove consumed bytes
				copy(buf, buf[consumed:bufPos])
				bufPos -= consumed
			}
		}
	}

	return 0, 0, fmt.Errorf("timeout")
}
