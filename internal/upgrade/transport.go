package upgrade

import "time"

// Transport abstracts the transport layer for firmware upgrade operations.
// CAN and UART each implement this interface with their own framing.
type Transport interface {
	// SendCommand sends a command frame (e.g., start update, confirm, version query, reboot).
	SendCommand(cmd uint32, param uint32) error

	// SendData sends an 8-byte firmware data frame.
	SendData(data []byte) error

	// WaitForResponse blocks until a response frame arrives or timeout.
	WaitForResponse(timeout time.Duration) (code uint32, val uint32, err error)

	// IsConnected returns whether the transport is connected.
	IsConnected() bool
}
