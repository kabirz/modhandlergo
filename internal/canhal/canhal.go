// Package canhal provides a hardware abstraction layer for CAN bus adapters.
package canhal

import "time"

// BaudRate enumerates standard CAN bus speeds.
type BaudRate int

const (
	Baud10K  BaudRate = iota
	Baud20K
	Baud50K
	Baud100K
	Baud125K
	Baud250K
	Baud500K
	Baud1M
	BaudCount
)

// BaudValues maps BaudRate enum to actual bit rates.
var BaudValues = [BaudCount]int{
	10000, 20000, 50000, 100000,
	125000, 250000, 500000, 1000000,
}

// BaudRateNames maps BaudRate enum to display names.
var BaudRateNames = [BaudCount]string{
	"10K", "20K", "50K", "100K",
	"125K", "250K", "500K", "1M",
}

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

// Adapter enumerates supported CAN adapter types.
type Adapter int

const (
	AdapterPCAN      Adapter = iota
	AdapterSocketCAN
	AdapterCount
)

// InvalidChannel indicates no valid CAN channel.
const InvalidChannel = -1

// Backend is the interface each CAN hardware driver implements.
type Backend interface {
	Name() string
	DetectDevices() ([]int, error)
	Connect(channel int, baud BaudRate) error
	Disconnect() error
	Write(frame *Frame) error
	Read(timeout time.Duration) (*Frame, error)
	SetFilter(fromID, toID uint32) error
	IsConnected() bool
	Channel() int
	Close() error
}
