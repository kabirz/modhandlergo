//go:build windows

package pcan

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// BaudRate enumerates standard CAN bus speeds.
type BaudRate int

const (
	Baud10K BaudRate = iota
	Baud20K
	Baud50K
	Baud100K
	Baud125K
	Baud250K
	Baud500K
	Baud1M
	BaudCount
)

// FrameFlags represents CAN frame type flags.
type FrameFlags uint8

const (
	FlagStandard FrameFlags = 0x00
	FlagExtended FrameFlags = 0x01
	FlagRemote   FrameFlags = 0x02
)

// Frame represents a single CAN frame.
type Frame struct {
	ID    uint32
	Data  [8]byte
	DLC   uint8
	Flags FrameFlags
}

// InvalidChannel indicates no valid CAN channel.
const InvalidChannel = -1

var (
	dll = syscall.NewLazyDLL("PCANBasic.dll")

	procInitialize     = dll.NewProc("CAN_Initialize")
	procUninitialize   = dll.NewProc("CAN_Uninitialize")
	procWrite          = dll.NewProc("CAN_Write")
	procRead           = dll.NewProc("CAN_Read")
	procFilterMessages = dll.NewProc("CAN_FilterMessages")
	procLookUpChannel  = dll.NewProc("CAN_LookUpChannel")
)

const (
	pcanErrorOK         = 0x00000
	pcanErrorQRCVEmpty  = 0x00020
	pcanMessageStandard = 0x00
	pcanMessageRTR      = 0x01
	pcanMessageExtended = 0x02
	pcanNonebus         = 0x00
)

var pcanBaudTable = [BaudCount]uint16{
	0x672F, 0x532F, 0x472F, 0x432F,
	0x031C, 0x011C, 0x001C, 0x0014,
}

type pcanMsg struct {
	ID      uint32
	MsgType uint8
	Len     uint8
	Data    [8]byte
}

type pcanTimestamp struct {
	Millis         uint32
	MillisOverflow uint16
	Micros         uint16
}

// Backend implements CAN backend for PCAN USB adapters.
type Backend struct {
	mu        sync.Mutex
	channel   int
	connected bool
	loaded    bool
}

// New creates a new PCAN backend.
func New() *Backend {
	b := &Backend{channel: InvalidChannel}
	if err := dll.Load(); err == nil {
		b.loaded = true
	}
	return b
}

func (b *Backend) Name() string { return "PCAN" }

func (b *Backend) DetectDevices() ([]int, error) {
	if !b.loaded {
		return nil, fmt.Errorf("PCAN: PCANBasic.dll not loaded")
	}
	var channels []int
	for i := 0; i < 16; i++ {
		param := fmt.Sprintf("devicetype=pcan_usb,controllernumber=%d", i)
		paramPtr, _ := syscall.BytePtrFromString(param)
		var ch uint16
		ret, _, _ := procLookUpChannel.Call(
			uintptr(unsafe.Pointer(paramPtr)),
			uintptr(unsafe.Pointer(&ch)),
		)
		if ret == pcanErrorOK && ch != pcanNonebus {
			channels = append(channels, int(ch))
		}
	}
	return channels, nil
}

func (b *Backend) Connect(channel int, baud BaudRate) error {
	if !b.loaded {
		return fmt.Errorf("PCAN: PCANBasic.dll not loaded")
	}
	if baud < 0 || baud >= BaudCount {
		return fmt.Errorf("PCAN: invalid baud rate index %d", baud)
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	ret, _, _ := procInitialize.Call(uintptr(channel), uintptr(pcanBaudTable[baud]), 0, 0, 0)
	if ret != pcanErrorOK {
		return fmt.Errorf("PCAN: initialization failed (channel=0x%x)", channel)
	}
	b.channel = channel
	b.connected = true
	return nil
}

func (b *Backend) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.channel != InvalidChannel {
		procUninitialize.Call(uintptr(b.channel))
	}
	b.channel = InvalidChannel
	b.connected = false
	return nil
}

func (b *Backend) Write(frame *Frame) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.connected {
		return fmt.Errorf("PCAN: not connected")
	}

	msg := pcanMsg{ID: frame.ID, Len: frame.DLC, MsgType: pcanMessageStandard}
	if frame.DLC > 8 {
		msg.Len = 8
	}
	if frame.Flags&FlagExtended != 0 {
		msg.MsgType = pcanMessageExtended
	}
	if frame.Flags&FlagRemote != 0 {
		msg.MsgType |= pcanMessageRTR
	}
	copy(msg.Data[:], frame.Data[:msg.Len])

	ret, _, _ := procWrite.Call(uintptr(b.channel), uintptr(unsafe.Pointer(&msg)))
	if ret != pcanErrorOK {
		return fmt.Errorf("PCAN: write failed")
	}
	return nil
}

func (b *Backend) Read(timeout time.Duration) (*Frame, error) {
	if !b.connected {
		return nil, fmt.Errorf("PCAN: not connected")
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var msg pcanMsg
		var ts pcanTimestamp
		ret, _, _ := procRead.Call(
			uintptr(b.channel),
			uintptr(unsafe.Pointer(&msg)),
			uintptr(unsafe.Pointer(&ts)),
		)
		if ret == pcanErrorOK {
			frame := &Frame{ID: msg.ID, DLC: msg.Len}
			copy(frame.Data[:], msg.Data[:])
			if msg.MsgType&pcanMessageExtended != 0 {
				frame.Flags |= FlagExtended
			}
			if msg.MsgType&pcanMessageRTR != 0 {
				frame.Flags |= FlagRemote
			}
			return frame, nil
		}
		if ret == pcanErrorQRCVEmpty {
			time.Sleep(time.Millisecond)
			continue
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, nil
}

func (b *Backend) SetFilter(fromID, toID uint32) error {
	if !b.connected {
		return fmt.Errorf("PCAN: not connected")
	}
	ret, _, _ := procFilterMessages.Call(uintptr(b.channel), uintptr(fromID), uintptr(toID), uintptr(pcanMessageStandard))
	if ret != pcanErrorOK {
		return fmt.Errorf("PCAN: set filter failed")
	}
	return nil
}

func (b *Backend) IsConnected() bool { return b.connected }
func (b *Backend) Channel() int      { return b.channel }

func (b *Backend) Close() error {
	if b.connected {
		b.Disconnect()
	}
	return nil
}
