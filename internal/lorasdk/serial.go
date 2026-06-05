package lorasdk

import (
	"fmt"
	"strings"
	"sync"
	"time"

	serial "go.bug.st/serial"
)

// SerialClient manages serial port AT command transport.
type SerialClient struct {
	mu      sync.Mutex
	port    serial.Port
	isOpen  bool
	atMode  bool
	cb      Callbacks
}

// NewSerialClient creates a new serial AT client.
func NewSerialClient(cb Callbacks) *SerialClient {
	return &SerialClient{cb: cb}
}

// Open opens a serial port for AT commands.
func (s *SerialClient) Open(portName string, baudRate int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isOpen {
		return fmt.Errorf("serial port already open")
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

	s.port = port
	s.isOpen = true
	s.atMode = false

	s.cb.OnLog(fmt.Sprintf("serial port %s opened (%d bps)", portName, baudRate), LogSerial)

	// Enter AT mode
	s.enterATMode()

	return nil
}

// Close closes the serial port.
func (s *SerialClient) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isOpen {
		return
	}

	if s.atMode {
		s.exitATMode()
	}

	s.port.Close()
	s.port = nil
	s.isOpen = false
	s.cb.OnLog("serial port closed", LogSerial)
}

// IsOpen returns whether the serial port is open.
func (s *SerialClient) IsOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isOpen
}

// SendAT sends an AT command via serial port and returns the response.
func (s *SerialClient) SendAT(cmd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isOpen || s.port == nil {
		return fmt.Errorf("serial port not open")
	}

	if !strings.HasSuffix(cmd, "\r\n") {
		cmd += "\r\n"
	}

	s.cb.OnLog(fmt.Sprintf("serial send: %s", strings.TrimSpace(cmd)), LogSerial)

	if _, err := s.port.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("serial write failed: %w", err)
	}

	// Read response
	s.port.SetReadTimeout(3 * time.Second)
	buf := make([]byte, 4096)
	var response []byte

	for {
		n, err := s.port.Read(buf)
		if n > 0 {
			response = append(response, buf[:n]...)
		}
		if err != nil {
			break
		}
		// Check if response is complete (ends with OK or ERROR)
		respStr := string(response)
		if strings.Contains(respStr, "OK") || strings.Contains(respStr, "ERROR") {
			break
		}
	}

	if len(response) > 0 {
		respStr := string(response)
		s.cb.OnLog(fmt.Sprintf("serial response: %s", strings.TrimSpace(respStr)), LogSerial)
		s.cb.OnATResponse(respStr)
	}

	return nil
}

// QueryDeviceInfo queries device information via serial AT.
func (s *SerialClient) QueryDeviceInfo() error {
	return s.SendAT("AT+INFO?")
}

// QueryNetParams queries network parameters via serial AT.
func (s *SerialClient) QueryNetParams() error {
	return s.SendAT("AT+NETDEV?")
}

func (s *SerialClient) enterATMode() {
	if s.port == nil {
		return
	}

	// Send AT command to enter AT mode
	s.port.Write([]byte("AT\r\n"))
	time.Sleep(200 * time.Millisecond)

	// Drain any response
	buf := make([]byte, 256)
	s.port.SetReadTimeout(500 * time.Millisecond)
	for {
		n, err := s.port.Read(buf)
		if n == 0 || err != nil {
			break
		}
	}

	s.atMode = true
	s.cb.OnLog("entered AT mode", LogSerial)
}

func (s *SerialClient) exitATMode() {
	if s.port == nil {
		return
	}

	s.port.Write([]byte("AT+EXIT\r\n"))
	time.Sleep(100 * time.Millisecond)
	s.atMode = false
}
