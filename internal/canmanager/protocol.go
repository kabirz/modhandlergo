// Package canmanager implements the CAN firmware upgrade protocol.
package canmanager

// Frame ID constants matching embedded mod-can.h
const (
	PlatformRx = 0x101 // Platform → Controller (control commands)
	PlatformTx = 0x102 // Controller → Platform (response)
	FWDataRx   = 0x103 // Platform → Controller (firmware data)

	Heartbeat       = 0x763 // Controller → Platform (heartbeat)
	ControllerState = 0x1E3 // Controller → Platform (X/Y BE + buttons)
	LaserRanging    = 0x263 // Platform → Controller (over-excavation + laser)
	CoordXY         = 0x363 // Platform → Controller (X/Y coordinates)
	CoordZ          = 0x463 // Platform → Controller (Z coordinate)

	LoraConfigCmd  = 0x105 // Platform → Controller (LoRa param config command)
	LoraConfigResp = 0x106 // Controller → Platform (LoRa param config response)
)

// VirtualChannel is a special channel value for test mode without hardware.
const VirtualChannel = 0xFFFF

// CANFrameData represents the 8-byte payload of a control frame.
// Maps to the C can_frame_t struct.
type CANFrameData struct {
	Code uint32
	Val  uint32
}

// PutBE16 writes a uint16 in big-endian order.
func PutBE16(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

// PutBE32 writes a uint32 in big-endian order.
func PutBE32(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

// GetBE16 reads a uint16 in big-endian order.
func GetBE16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}

// GetBE32 reads a uint32 in big-endian order.
func GetBE32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// DecodeCANFrameData parses 8 bytes into CANFrameData.
func DecodeCANFrameData(data []byte) CANFrameData {
	if len(data) < 8 {
		return CANFrameData{}
	}
	return CANFrameData{
		Code: GetBE32(data[0:4]),
		Val:  GetBE32(data[4:8]),
	}
}

// EncodeCANFrameData writes CANFrameData into 8 bytes.
func EncodeCANFrameData(d CANFrameData, data []byte) {
	PutBE32(data[0:4], d.Code)
	PutBE32(data[4:8], d.Val)
}
