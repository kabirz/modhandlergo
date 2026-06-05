//go:build linux

package socketcan

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

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

const (
	canRaw      = 1
	canFrameLen = 16
)

type canFrameLayout struct {
	ID   uint32
	DLC  uint8
	Pad  [3]byte
	Data [8]byte
}

// Backend implements CAN backend for Linux SocketCAN.
type Backend struct {
	mu        sync.Mutex
	iface     string
	ifaceIdx  int
	fd        int
	connected bool
}

// New creates a new SocketCAN backend.
func New() *Backend {
	return &Backend{fd: -1}
}

func (b *Backend) Name() string { return "SocketCAN" }

func (b *Backend) DetectDevices() ([]int, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("socketcan: failed to enumerate interfaces: %w", err)
	}
	var channels []int
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "can") || strings.HasPrefix(iface.Name, "vcan") {
			channels = append(channels, iface.Index)
		}
	}
	return channels, nil
}

func (b *Backend) Connect(channel int, baud BaudRate) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	iface, err := net.InterfaceByIndex(channel)
	if err != nil {
		return fmt.Errorf("socketcan: interface with index %d not found: %w", channel, err)
	}

	fd, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW, canRaw)
	if err != nil {
		return fmt.Errorf("socketcan: failed to create socket: %w", err)
	}

	addr := &unix.SockaddrCAN{Ifindex: channel}
	if err := unix.Bind(fd, addr); err != nil {
		unix.Close(fd)
		return fmt.Errorf("socketcan: failed to bind to %s: %w", iface.Name, err)
	}

	b.fd = fd
	b.iface = iface.Name
	b.ifaceIdx = channel
	b.connected = true
	return nil
}

func (b *Backend) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fd >= 0 {
		unix.Close(b.fd)
		b.fd = -1
	}
	b.iface = ""
	b.ifaceIdx = 0
	b.connected = false
	return nil
}

func (b *Backend) Write(frame *Frame) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.connected || b.fd < 0 {
		return fmt.Errorf("socketcan: not connected")
	}

	cf := canFrameLayout{ID: frame.ID, DLC: frame.DLC, Data: frame.Data}
	if frame.Flags&FlagExtended != 0 {
		cf.ID |= 0x80000000
	}
	if frame.Flags&FlagRemote != 0 {
		cf.ID |= 0x40000000
	}

	buf := (*[canFrameLen]byte)(unsafe.Pointer(&cf))[:]
	_, err := unix.Write(b.fd, buf)
	if err != nil {
		return fmt.Errorf("socketcan: write failed: %w", err)
	}
	return nil
}

func (b *Backend) Read(timeout time.Duration) (*Frame, error) {
	if !b.connected || b.fd < 0 {
		return nil, fmt.Errorf("socketcan: not connected")
	}

	fdSet := &unix.FdSet{}
	fdSet.Set(b.fd)
	tv := &unix.Timeval{
		Sec:  int64(timeout / time.Second),
		Usec: int64((timeout % time.Second) / time.Microsecond),
	}
	n, err := unix.Select(b.fd+1, fdSet, nil, nil, tv)
	if err != nil || n == 0 {
		return nil, nil
	}

	var buf [canFrameLen]byte
	_, err = unix.Read(b.fd, buf[:])
	if err != nil {
		return nil, err
	}

	cf := (*canFrameLayout)(unsafe.Pointer(&buf[0]))
	var flags FrameFlags
	if cf.ID&0x80000000 != 0 {
		flags |= FlagExtended
	}
	if cf.ID&0x40000000 != 0 {
		flags |= FlagRemote
	}

	return &Frame{
		ID:    cf.ID & 0x1FFFFFFF,
		DLC:   cf.DLC,
		Data:  cf.Data,
		Flags: flags,
	}, nil
}

func (b *Backend) SetFilter(fromID, toID uint32) error { return nil }
func (b *Backend) IsConnected() bool                   { return b.connected }
func (b *Backend) Channel() int {
	if b.connected {
		return b.ifaceIdx
	}
	return InvalidChannel
}
func (b *Backend) Close() error {
	if b.connected {
		b.Disconnect()
	}
	return nil
}

// EnumerateCANInterfaceNames returns names of all CAN interfaces.
func EnumerateCANInterfaceNames() []string {
	entries, err := os.ReadDir("/sys/class/net/")
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "can") || strings.HasPrefix(name, "vcan") {
			if _, err := net.InterfaceByName(name); err == nil {
				names = append(names, name)
			}
		}
	}
	return names
}
