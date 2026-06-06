// Package upgrade provides shared firmware upgrade logic for both CAN and UART transports.
// It extracts the common state machine from canmanager and uartmanager.
package upgrade

// Command codes shared between CAN and UART protocols.
const (
	CmdStartUpdate = 0
	CmdConfirm     = 1
	CmdVersion     = 2
	CmdReboot      = 3
)

// EncodeCommand builds an 8-byte command payload (little-endian).
func EncodeCommand(cmd uint32, param uint32) [8]byte {
	var data [8]byte
	data[0] = byte(cmd)
	data[1] = byte(cmd >> 8)
	data[2] = byte(cmd >> 16)
	data[3] = byte(cmd >> 24)
	data[4] = byte(param)
	data[5] = byte(param >> 8)
	data[6] = byte(param >> 16)
	data[7] = byte(param >> 24)
	return data
}

// Response codes shared between CAN and UART protocols.
const (
	FWCodeOffset      = 0
	FWCodeSuccess     = 1
	FWCodeVersion     = 2
	FWCodeConfirm     = 3
	FWCodeFlashError  = 4
	FWCodeTransferErr = 5
)
