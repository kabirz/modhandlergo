package lorasdk

import (
	"context"
	"sync"
)

// SDK is the main LoRa Gateway SDK struct.
// It manages TCP data streaming, UDP device discovery, and serial AT commands.
type SDK struct {
	mu          sync.Mutex
	cb          Callbacks
	tcp         *TCPClient
	udp         *UDPClient
	serial      *SerialClient
	atTransport ATTransport
	testFlag    int
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewSDK creates a new LoRa SDK instance.
func NewSDK(cb Callbacks) *SDK {
	ctx, cancel := context.WithCancel(context.Background())

	s := &SDK{
		cb:          cb,
		tcp:         NewTCPClient(cb),
		udp:         NewUDPClient(cb),
		serial:      NewSerialClient(cb),
		atTransport: ATTransportUDP,
		ctx:         ctx,
		cancel:      cancel,
	}

	return s
}

// Close cleans up all resources held by the SDK.
func (s *SDK) Close() {
	s.cancel()
	s.tcp.Disconnect()
	s.serial.Close()
}

// --- TCP Operations ---

// Connect establishes a TCP connection to the LoRa gateway.
func (s *SDK) Connect(ip string, port int) {
	s.tcp.Connect(ip, port)
}

// Disconnect closes the TCP connection.
func (s *SDK) Disconnect() {
	s.tcp.Disconnect()
}

// ConnState returns the current TCP connection state.
func (s *SDK) ConnState() ConnState {
	return s.tcp.State()
}

// SendFrame sends a LoRa frame to a node via TCP.
func (s *SDK) SendFrame(nid uint32, data []byte) error {
	return s.tcp.SendFrame(nid, data)
}

// SendRSSIResponse sends an RSSI response frame.
func (s *SDK) SendRSSIResponse(nid uint32, snrRaw, rssiRaw, testFlag byte) error {
	return s.tcp.SendRSSIResponse(nid, snrRaw, rssiRaw, testFlag)
}

// --- UDP Operations ---

// SearchDevices broadcasts a discovery packet.
func (s *SDK) SearchDevices() {
	s.udp.SearchDevices(s.ctx)
}

// GetNetParams queries network parameters from the gateway.
func (s *SDK) GetNetParams(gatewayIP string) {
	s.udp.GetNetParams(gatewayIP)
}

// SendAT sends an AT command via the configured transport (UDP or serial).
func (s *SDK) SendAT(atCmd string, gatewayIP string) {
	s.mu.Lock()
	transport := s.atTransport
	s.mu.Unlock()

	if transport == ATTransportSerial && s.serial.IsOpen() {
		s.serial.SendAT(atCmd)
	} else {
		s.udp.SendAT(atCmd, gatewayIP)
	}
}

// QueryRSSI queries gateway RSSI via UDP and sends response via TCP.
func (s *SDK) QueryRSSI(gatewayIP string, nid uint32) {
	s.udp.QueryRSSI(gatewayIP, s.tcp, nid)
}

// Reboot sends AT+Z to reboot the gateway.
func (s *SDK) Reboot(gatewayIP string) {
	s.SendAT("AT+Z", gatewayIP)
}

// --- Serial Operations ---

// SerialOpen opens a serial port for AT commands.
func (s *SDK) SerialOpen(portName string, baudRate int) error {
	return s.serial.Open(portName, baudRate)
}

// SerialClose closes the serial port.
func (s *SDK) SerialClose() {
	s.serial.Close()
}

// SerialIsOpen returns whether the serial port is open.
func (s *SDK) SerialIsOpen() bool {
	return s.serial.IsOpen()
}

// --- Transport Selection ---

// SetATTransport selects the transport for AT commands.
func (s *SDK) SetATTransport(transport ATTransport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.atTransport = transport
}

// GetATTransport returns the current AT transport.
func (s *SDK) GetATTransport() ATTransport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.atTransport
}

// --- Test Mode ---

// SetTestFlag sets the test flag for RSSI responses.
func (s *SDK) SetTestFlag(flag int) {
	s.testFlag = flag
}

// IsTCPConnected returns whether TCP is connected.
func (s *SDK) IsTCPConnected() bool {
	return s.tcp.IsConnected()
}
