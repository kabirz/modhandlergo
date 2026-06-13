package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
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

	optEcho  byte = 0x01 // ECHO
	optSGA   byte = 0x03 // Suppress Go Ahead
	optTType byte = 0x18 // TERMINAL-TYPE
	optNAWS  byte = 0x1F // Negotiate About Window Size

	telnetSubIS   byte = 0 // Sub-negotiation: IS (client sends term type)
	telnetSubSend byte = 1 // Sub-negotiation: SEND (server requests term type)
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
	termType   string
	cols       uint16
	rows       uint16

	// UART-specific
	portName string
	baudRate int
}

func NewTerminalService() *TerminalService {
	return &TerminalService{
		termType: "xterm-256color",
		cols:     80,
		rows:     24,
	}
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

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP connect failed: %w", err)
	}

	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetKeepAlive(true)
		tc.SetKeepAlivePeriod(30 * time.Second)
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
func (s *TerminalService) ConnectTelnet(host string, port, cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("already connected")
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("Telnet connect failed: %w", err)
	}

	if tc, ok := conn.(*net.TCPConn); ok {
		tc.SetKeepAlive(true)
		tc.SetKeepAlivePeriod(30 * time.Second)
	}

	s.conn = conn
	s.trans = TransportTelnet
	s.address = addr
	s.running = true
	s.telnetMode = true
	s.cols = uint16(cols)
	s.rows = uint16(rows)
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
			// Extract sub-negotiation payload (between SB and IAC SE)
			s.handleTelnetSubneg(data[i+2 : i+2+end])
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
	var resp []byte
	var nawsConn io.Writer

	s.mu.Lock()
	conn := s.conn
	switch cmd {
	case telnetDO:
		switch opt {
		case optSGA, optTType, optNAWS:
			resp = []byte{telnetIAC, telnetWILL, opt}
		default:
			resp = []byte{telnetIAC, telnetWONT, opt}
		}
		if opt == optNAWS {
			nawsConn = conn
		}
	case telnetDONT:
	case telnetWILL:
		if opt == optEcho || opt == optSGA {
			resp = []byte{telnetIAC, telnetDO, opt}
		} else {
			resp = []byte{telnetIAC, telnetDONT, opt}
		}
	case telnetWONT:
	}
	s.mu.Unlock()

	if len(resp) > 0 && conn != nil {
		_, _ = conn.Write(resp)
	}
	if nawsConn != nil {
		s.mu.Lock()
		cols := s.cols
		rows := s.rows
		s.mu.Unlock()
		writeNAWS(nawsConn, cols, rows)
	}
}

// iacSE is the Telnet sub-negotiation terminator sequence (IAC SE).
var iacSE = []byte{telnetIAC, telnetSE}

// bytesIndexIACSE searches for the IAC SE terminator in data and returns the
// offset of IAC relative to the start of data (excluding IAC itself).
func bytesIndexIACSE(data []byte) int {
	return bytes.Index(data, iacSE)
}

// handleTelnetSubneg processes a sub-negotiation payload (content between SB and IAC SE).
func (s *TerminalService) handleTelnetSubneg(payload []byte) {
	if len(payload) < 1 {
		return
	}
	opt := payload[0]

	switch opt {
	case optTType:
		// TERMINAL-TYPE: server sends SEND → client responds IS "xterm-256color"
		if len(payload) >= 2 && payload[1] == telnetSubSend {
			s.mu.Lock()
			conn := s.conn
			tt := s.termType
			s.mu.Unlock()
			if conn == nil {
				return
			}
			resp := append([]byte{telnetIAC, telnetSB, optTType, telnetSubIS}, []byte(tt)...)
			resp = append(resp, telnetIAC, telnetSE)
			_, _ = conn.Write(resp)
		}
	}
}

// sendNAWS sends the current window size to the server via NAWS sub-negotiation.
// Caller must NOT hold s.mu.
func (s *TerminalService) sendNAWS(conn io.Writer) {
	s.mu.Lock()
	cols := s.cols
	rows := s.rows
	s.mu.Unlock()

	writeNAWS(conn, cols, rows)
}

// writeNAWS writes a NAWS sub-negotiation frame. Does not acquire any locks.
func writeNAWS(conn io.Writer, cols, rows uint16) {
	resp := []byte{
		telnetIAC, telnetSB, optNAWS,
		byte(cols >> 8), byte(cols),
		byte(rows >> 8), byte(rows),
		telnetIAC, telnetSE,
	}
	_, _ = conn.Write(resp)
}

// Resize updates the terminal dimensions and sends NAWS if in Telnet mode.
func (s *TerminalService) Resize(cols, rows int) error {
	s.mu.Lock()
	s.cols = uint16(cols)
	s.rows = uint16(rows)
	telnet := s.telnetMode
	conn := s.conn
	s.mu.Unlock()

	if telnet && conn != nil {
		writeNAWS(conn, uint16(cols), uint16(rows))
	}
	return nil
}

func (s *TerminalService) emitStatus(running bool) {
	s.pushEvent("terminal:status", running)
}

func (s *TerminalService) pushEvent(event string, data any) {
	if s.app != nil {
		s.app.Event.Emit(event, data)
	}
}

func (s *TerminalService) pushError(format string, args ...any) {
	s.pushEvent("terminal:error", fmt.Sprintf(format, args...))
}
