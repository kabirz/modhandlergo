package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.bug.st/serial"
)

// ── Telnet protocol constants ──

const (
	telnetIAC  byte = 0xFF // Interpret As Command
	telnetWILL byte = 0xFB
	telnetWONT byte = 0xFC
	telnetDO   byte = 0xFD
	telnetDONT byte = 0xFE
	telnetSB   byte = 0xFA // Sub-negotiation Begin
	telnetSE   byte = 0xF0 // Sub-negotiation End
	telnetNOP  byte = 0xF1

	optEcho byte = 0x01 // ECHO
	optSGA  byte = 0x03 // Suppress Go Ahead
)

// ── Public types (exported for TypeScript binding) ──

type TransportType string

const (
	TransportTCP    TransportType = "tcp"
	TransportTelnet TransportType = "telnet"
	TransportUART   TransportType = "uart"
)

// TerminalService provides a raw terminal over TCP, Telnet, or UART,
// similar to a serial terminal or Telnet client.
type TerminalService struct {
	app *application.App

	mu      sync.Mutex
	conn    io.ReadWriteCloser
	cancel  context.CancelFunc
	trans   TransportType
	address string
	running bool

	// Telnet mode
	telnetMode bool

	// UART-specific
	portName string
	baudRate int
}

func NewTerminalService() *TerminalService {
	return &TerminalService{}
}

func (s *TerminalService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	s.app = application.Get()
	return nil
}

// ConnectTCP connects to a TCP host:port.
func (s *TerminalService) ConnectTCP(host string, port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("already connected")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP connect failed: %w", err)
	}

	s.conn = conn
	s.trans = TransportTCP
	s.address = addr
	s.running = true
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.readLoop(ctx)

	s.emitStatus(true)
	return nil
}

// ConnectTelnet connects to a TCP host:port with Telnet IAC negotiation.
func (s *TerminalService) ConnectTelnet(host string, port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("already connected")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("Telnet connect failed: %w", err)
	}

	s.conn = conn
	s.trans = TransportTelnet
	s.address = addr
	s.running = true
	s.telnetMode = true
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.readLoop(ctx)

	s.emitStatus(true)
	return nil
}

// ConnectUART connects to a serial port.
func (s *TerminalService) ConnectUART(portName string, baudRate int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("already connected")
	}

	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("UART connect failed: %w", err)
	}

	// Set read/write timeouts so readLoop doesn't block forever
	port.SetReadTimeout(100 * time.Millisecond)

	s.conn = port
	s.trans = TransportUART
	s.portName = portName
	s.baudRate = baudRate
	s.address = fmt.Sprintf("%s@%d", portName, baudRate)
	s.running = true
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.readLoop(ctx)

	s.emitStatus(true)
	return nil
}

// Disconnect closes the connection.
func (s *TerminalService) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}
	s.running = false
	s.telnetMode = false
	if s.cancel != nil {
		s.cancel()
	}
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.emitStatus(false)
	return nil
}

// Send writes data to the connection.
func (s *TerminalService) Send(data string) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err := conn.Write([]byte(data))
	if err != nil {
		s.pushError("write failed: %v", err)
	}
	return err
}

// EnumPorts lists available serial ports.
func (s *TerminalService) EnumPorts() []map[string]string {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil
	}
	result := make([]map[string]string, 0, len(ports))
	for _, p := range ports {
		result = append(result, map[string]string{"name": p})
	}
	return result
}

// IsConnected returns whether the terminal is connected.
func (s *TerminalService) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *TerminalService) ServiceShutdown() error {
	s.Disconnect()
	return nil
}

// ── Internal ──

func (s *TerminalService) readLoop(ctx context.Context) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		s.mu.Lock()
		conn := s.conn
		telnet := s.telnetMode
		s.mu.Unlock()
		if conn == nil {
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			// Check intentional disconnect BEFORE cleanup,
			// because cleanup calls cancel() which makes ctx.Done() always true.
			select {
			case <-ctx.Done():
				// intentional — Disconnect() already emitted status
				return
			default:
			}

			// Unexpected disconnect — clean up and notify
			s.mu.Lock()
			s.running = false
			s.telnetMode = false
			if s.cancel != nil {
				s.cancel()
			}
			if s.conn != nil {
				s.conn.Close()
				s.conn = nil
			}
			s.mu.Unlock()

			if err != io.EOF {
				s.pushError("read error: %v", err)
			}
			s.pushEvent("terminal:status", false)
			return
		}
		if n > 0 {
			if telnet {
				s.handleTelnetData(buf[:n])
			} else {
				s.pushEvent("terminal:data", string(buf[:n]))
			}
		}
	}
}

// ── Telnet IAC handling ──

// handleTelnetData parses IAC sequences from raw Telnet data,
// responds to negotiations, and emits only clean text to the frontend.
func (s *TerminalService) handleTelnetData(data []byte) {
	var clean []byte
	i := 0
	for i < len(data) {
		if data[i] != telnetIAC {
			clean = append(clean, data[i])
			i++
			continue
		}
		// IAC found — need at least 2 bytes
		if i+1 >= len(data) {
			// Trailing IAC at buffer boundary, keep for next read
			clean = append(clean, data[i])
			break
		}
		cmd := data[i+1]

		// IAC IAC → escaped 0xFF literal
		if cmd == telnetIAC {
			clean = append(clean, 0xFF)
			i += 2
			continue
		}

		// 3-byte commands: WILL/WONT/DO/DONT
		if cmd == telnetWILL || cmd == telnetWONT || cmd == telnetDO || cmd == telnetDONT {
			if i+2 >= len(data) {
				break // incomplete, wait for more data
			}
			opt := data[i+2]
			s.handleTelnetCommand(cmd, opt)
			i += 3
			continue
		}

		// Sub-negotiation: IAC SB ... IAC SE
		if cmd == telnetSB {
			end := bytesIndexIACSE(data[i+2:])
			if end < 0 {
				// Incomplete sub-negotiation — skip for now
				i = len(data)
				continue
			}
			// Skip entire sub-negotiation (i+2 + end + 2 for IAC SE)
			i += 2 + end + 2
			continue
		}

		// 2-byte commands (NOP, BRK, etc.) — skip
		i += 2
	}

	if len(clean) > 0 {
		s.pushEvent("terminal:data", string(clean))
	}
}

// handleTelnetCommand responds to a single WILL/WONT/DO/DONT negotiation.
func (s *TerminalService) handleTelnetCommand(cmd, opt byte) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return
	}

	var resp []byte
	switch cmd {
	case telnetDO:
		if opt == optSGA {
			// We will suppress go-ahead
			resp = []byte{telnetIAC, telnetWILL, opt}
		} else {
			resp = []byte{telnetIAC, telnetWONT, opt}
		}
	case telnetDONT:
		// No response needed
	case telnetWILL:
		if opt == optEcho || opt == optSGA {
			resp = []byte{telnetIAC, telnetDO, opt}
		} else {
			resp = []byte{telnetIAC, telnetDONT, opt}
		}
	case telnetWONT:
		// No response needed
	}

	if len(resp) > 0 {
		// Best-effort write; ignore errors (connection may close)
		_, _ = conn.Write(resp)
	}
}

// bytesIndexIACSE searches for IAC SE pattern in data, returns the offset
// of IAC relative to the start of data (not including the IAC itself).
func bytesIndexIACSE(data []byte) int {
	for i := 0; i < len(data)-1; i++ {
		if data[i] == telnetIAC && data[i+1] == telnetSE {
			return i
		}
	}
	return -1
}

func (s *TerminalService) emitStatus(running bool) {
	s.pushEvent("terminal:status", running)
}

func (s *TerminalService) pushEvent(event string, data interface{}) {
	if s.app != nil {
		s.app.Event.Emit(event, data)
	}
}

func (s *TerminalService) pushError(format string, args ...interface{}) {
	s.pushEvent("terminal:error", fmt.Sprintf(format, args...))
}
