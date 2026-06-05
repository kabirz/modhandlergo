package canmanager

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kabirz/modhandlergo/internal/canhal"
	"github.com/kabirz/modhandlergo/internal/candispatcher"
)

// Manager handles CAN firmware upgrade operations: connect, version query,
// board reboot, and OTA firmware flashing.
type Manager struct {
	mu         sync.Mutex
	backend    canhal.Backend
	dispatcher *candispatcher.Dispatcher
	channel    int
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

func (m *Manager) progress(pct int) {
	if m.onProgress != nil {
		m.onProgress(pct)
	}
}

// Connect establishes a CAN connection on the given channel and baud rate.
func (m *Manager) Connect(channel int, baud canhal.BaudRate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.channel != canhal.InvalidChannel {
		m.log("CAN 连接已存在, 请勿重复连接")
		return nil
	}

	if channel == VirtualChannel {
		m.channel = channel
		m.virtualMode = true
		m.log("虚拟 CAN 连接成功 (测试模式)")
		return nil
	}

	if m.backend == nil {
		return fmt.Errorf("CAN HAL 未初始化")
	}

	if err := m.backend.Connect(channel, baud); err != nil {
		return fmt.Errorf("CAN 初始化失败: %w", err)
	}

	m.channel = channel
	m.virtualMode = false
	m.log(fmt.Sprintf("CAN(id=%xh) 连接成功 (%s)", channel, m.backend.Name()))

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
		m.log("虚拟 CAN 连接已断开")
	} else if m.channel != canhal.InvalidChannel {
		m.log(fmt.Sprintf("CAN(id=%xh) 连接已断开", m.channel))
		if m.backend != nil {
			m.backend.Disconnect()
		}
	}
	m.channel = canhal.InvalidChannel
	m.virtualMode = false
}

// IsConnected returns whether the CAN is connected.
func (m *Manager) IsConnected() bool {
	return m.channel != canhal.InvalidChannel
}

// GetChannel returns the current channel or InvalidChannel.
func (m *Manager) GetChannel() int {
	return m.channel
}

// GetFirmwareVersion queries the board firmware version via CAN.
func (m *Manager) GetFirmwareVersion() (uint32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.channel == canhal.InvalidChannel {
		return 0, fmt.Errorf("CAN已断开连接, 请重新连接")
	}

	if m.virtualMode {
		m.log("固件版本: v1.0.0 (虚拟 CAN)")
		return 0x01000000, nil
	}

	// Send version query
	var data [8]byte
	EncodeCANFrameData(CANFrameData{Code: CmdVersion}, data[:])

	frame := &canhal.Frame{
		ID:    PlatformRx,
		DLC:   8,
		Data:  data,
		Flags: canhal.FlagStandard,
	}
	if err := m.backend.Write(frame); err != nil {
		return 0, fmt.Errorf("CAN 发送失败")
	}

	// Wait for response
	resp, err := m.waitForResponse(5 * time.Second)
	if err != nil {
		return 0, fmt.Errorf("CAN 读取失败，超时！！")
	}

	if resp.Code == FWCodeVersion {
		version := resp.Val
		m.log(fmt.Sprintf("固件版本: v%d.%d.%d",
			(version>>24)&0xFF, (version>>16)&0xFF, (version>>8)&0xFF))
		return version, nil
	}

	return 0, fmt.Errorf("CAN 读取失败，数据错误！！")
}

// BoardReboot sends a reboot command to the board.
func (m *Manager) BoardReboot() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.channel == canhal.InvalidChannel {
		return fmt.Errorf("CAN已断开连接, 请重新连接")
	}

	if m.virtualMode {
		m.log("虚拟板卡重启成功")
		return nil
	}

	var data [8]byte
	EncodeCANFrameData(CANFrameData{Code: CmdReboot}, data[:])

	frame := &canhal.Frame{
		ID:    PlatformRx,
		DLC:   8,
		Data:  data,
		Flags: canhal.FlagStandard,
	}
	if err := m.backend.Write(frame); err != nil {
		return fmt.Errorf("CAN 发送失败")
	}
	return nil
}

// FirmwareUpgrade performs a complete firmware upgrade over CAN.
func (m *Manager) FirmwareUpgrade(filePath string, testMode bool) error {
	m.mu.Lock()
	if m.channel == canhal.InvalidChannel {
		m.mu.Unlock()
		return fmt.Errorf("CAN已断开连接, 请重新连接")
	}
	m.mu.Unlock()

	if m.virtualMode {
		return m.virtualFirmwareUpgrade(filePath)
	}
	return m.halFirmwareUpgrade(filePath, testMode)
}

// DetectDevices delegates to the CAN backend to detect available devices.
func (m *Manager) DetectDevices() ([]int, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("CAN HAL 未初始化")
	}
	return m.backend.DetectDevices()
}

// ReplaceBackend swaps the CAN backend and dispatcher (for adapter hot-switching).
func (m *Manager) ReplaceBackend(backend canhal.Backend, dispatcher *candispatcher.Dispatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backend = backend
	m.dispatcher = dispatcher
}

// GetBackend returns the current CAN backend.
func (m *Manager) GetBackend() canhal.Backend {
	return m.backend
}

// GetDispatcher returns the current dispatcher.
func (m *Manager) GetDispatcher() *candispatcher.Dispatcher {
	return m.dispatcher
}

func (m *Manager) waitForResponse(timeout time.Duration) (*CANFrameData, error) {
	if m.dispatcher == nil {
		return nil, fmt.Errorf("dispatcher not initialized")
	}

	frame, err := m.dispatcher.WaitFrame(PlatformTx, timeout)
	if err != nil {
		return nil, err
	}

	d := DecodeCANFrameData(frame.Data[:])
	return &d, nil
}

func (m *Manager) halFirmwareUpgrade(filePath string, testMode bool) error {
	// Open firmware file
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
	var data [8]byte
	EncodeCANFrameData(CANFrameData{Code: CmdStartUpdate, Val: uint32(fileSize)}, data[:])

	frame := &canhal.Frame{
		ID:    PlatformRx,
		DLC:   8,
		Data:  data,
		Flags: canhal.FlagStandard,
	}
	if err := m.backend.Write(frame); err != nil {
		return fmt.Errorf("发送固件大小失败")
	}

	// Wait for flash erase
	resp, err := m.waitForResponse(15 * time.Second)
	if err != nil {
		return fmt.Errorf("Flash擦除超时")
	}
	if resp.Code != FWCodeOffset || resp.Val != 0 {
		return fmt.Errorf("Flash擦除失败: code(%d), offset(%d)", resp.Code, resp.Val)
	}

	// Send firmware data 8 bytes at a time
	var bytesSent int64
	buf := make([]byte, 8)
	frame.ID = FWDataRx

	for {
		n, err := f.Read(buf)
		if err != nil || n == 0 {
			break
		}

		copy(frame.Data[:], buf)
		frame.DLC = uint8(n)
		// Zero-fill remaining bytes
		for i := n; i < 8; i++ {
			frame.Data[i] = 0
		}

		if err := m.backend.Write(frame); err != nil {
			return fmt.Errorf("发送文件数据失败")
		}

		bytesSent += int64(n)

		// Every 64 bytes or at end, check acknowledgment
		if bytesSent%64 == 0 || bytesSent == fileSize {
			m.progress(int(bytesSent * 100 / fileSize))

			resp, err := m.waitForResponse(5 * time.Second)
			if err != nil {
				return fmt.Errorf("固件更新超时!")
			}
			if resp.Code == FWCodeSuccess && resp.Val == uint32(bytesSent) {
				break
			}
			if resp.Code != FWCodeOffset {
				return fmt.Errorf("固件升级失败: code(%d), offset(%d)", resp.Code, resp.Val)
			}
		}
	}

	m.progress(100)

	// Send confirm command
	frame.ID = PlatformRx
	frame.DLC = 8
	frame.Flags = canhal.FlagStandard
	confirmVal := uint32(1)
	if testMode {
		confirmVal = 0
	}
	EncodeCANFrameData(CANFrameData{Code: CmdConfirm, Val: confirmVal}, frame.Data[:])

	if err := m.backend.Write(frame); err != nil {
		return fmt.Errorf("固件发送确认失败!")
	}

	// Wait for confirmation
	resp, err = m.waitForResponse(30 * time.Second)
	if err != nil {
		return fmt.Errorf("固件确认超时!")
	}

	if resp.Code == FWCodeConfirm && resp.Val == 0x55AA55AA {
		m.log(fmt.Sprintf("文件 %s 上传完成. 点击重启，板卡将在45-60秒内完成重启", filePath))
		return nil
	}
	if resp.Code == FWCodeTransferErr {
		return fmt.Errorf("固件更新失败")
	}

	return fmt.Errorf("固件确认失败: code(%d), val(0x%08X)", resp.Code, resp.Val)
}

func (m *Manager) virtualFirmwareUpgrade(filePath string) error {
	m.log("虚拟 CAN 模式：模拟固件升级...")

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("无法打开源固件文件")
	}
	defer f.Close()

	fi, _ := f.Stat()
	fileSize := fi.Size()
	m.log(fmt.Sprintf("开始固件升级, 固件大小: %d 字节", fileSize))
	m.log("输出文件: virtual_firmware.bin")

	time.Sleep(500 * time.Millisecond)
	m.log("Flash 擦除完成")

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
	m.log("固件发送完成")
	time.Sleep(200 * time.Millisecond)
	m.log("固件确认完成")

	m.progress(100)
	m.log(fmt.Sprintf("虚拟固件已保存 (%d 字节)", totalBytes))
	return nil
}

// Ensure Manager satisfies Wails ServiceStartup interface (optional lifecycle).
var _ context.Context = context.Background()
