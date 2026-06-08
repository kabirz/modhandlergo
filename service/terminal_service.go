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

// ── Public types (exported for TypeScript binding) ──

type TransportType string

const (
	TransportTCP  TransportType = "tcp"
	TransportUART TransportType = "uart"
)

// TerminalService provides a raw terminal over TCP or UART,
// similar to a serial terminal or Telnet client.
type TerminalService struct {
	app *application.App

	mu      sync.Mutex
	conn    io.ReadWriteCloser
	cancel  context.CancelFunc
	trans   TransportType
	address string
	running bool

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
		s.mu.Unlock()
		if conn == nil {
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			// Connection closed — clean up
			s.mu.Lock()
			s.running = false
			if s.cancel != nil {
				s.cancel()
			}
			if s.conn != nil {
				s.conn.Close()
				s.conn = nil
			}
			s.mu.Unlock()

			// Report error only if not an intentional disconnect
			select {
			case <-ctx.Done():
				// intentional — skip error
			default:
				if err != io.EOF {
					s.pushError("read error: %v", err)
				}
			}
			s.pushEvent("terminal:status", false)
			return
		}
		if n > 0 {
			s.pushEvent("terminal:data", string(buf[:n]))
		}
	}
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
