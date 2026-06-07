// Package lorasdk provides a pure Go implementation of the LoRa Gateway SDK.
// It supports TCP data streaming, UDP device discovery, and serial AT commands.
package lorasdk

import "encoding/binary"

// ConnState represents the TCP connection state.
type ConnState int

const (
	ConnDisconnected ConnState = iota
	ConnConnecting
	ConnConnected
)

func (s ConnState) String() string {
	switch s {
	case ConnDisconnected:
		return "disconnected"
	case ConnConnecting:
		return "connecting"
	case ConnConnected:
		return "connected"
	default:
		return "unknown"
	}
}

// LogSource identifies which transport generated a log message.
type LogSource int

const (
	LogTCP LogSource = iota
	LogUDP
	LogSerial
)

func (s LogSource) String() string {
	switch s {
	case LogTCP:
		return "TCP"
	case LogUDP:
		return "UDP"
	case LogSerial:
		return "Serial"
	default:
		return "unknown"
	}
}

// ATTransport selects the transport for AT commands.
type ATTransport int

const (
	ATTransportUDP    ATTransport = 0
	ATTransportSerial ATTransport = 1
)

// Wire protocol constants
const (
	FrameHdr1     = 0xAA
	FrameHdr2     = 0x55
	FrameOverhead = 10 // HDR(2) + NID(4) + LEN(2) + CRC(2)
	FrameWrapper  = 4  // 2 header + 2 footer (\r\n)
	CRLF          = "\r\n"

	DataHandler = 0x01
	DataTest    = 0x02
	DataRSSI    = 0x03

	ScannerFrameSize  = 20
	ScannerFOverBreak = 0x01
	ScannerFLaser     = 0x02
	ScannerFCoordZ    = 0x04
	ScannerFCoordXY   = 0x08
)

// ScannerData represents parsed scanner telemetry data.
type ScannerData struct {
	OverbreakValid bool   `json:"overbreakValid"`
	LaserValid     bool   `json:"laserValid"`
	CoordZValid    bool   `json:"coordZValid"`
	CoordXYValid   bool   `json:"coordXYValid"`
	Overbreak      int16  `json:"overbreak"`
	Laser          uint32 `json:"laser"`
	CoordX         int32  `json:"coordX"`
	CoordY         int32  `json:"coordY"`
	CoordZ         int32  `json:"coordZ"`
}

// ParseScannerData parses a merged scanner frame payload into ScannerData.
// 'payload' includes the type byte. 'len' is the data field length.
func ParseScannerData(payload []byte) (ScannerData, bool) {
	if len(payload) < ScannerFrameSize || payload[0] != DataHandler {
		return ScannerData{}, false
	}
	flags := payload[1]
	return ScannerData{
		OverbreakValid: flags&ScannerFOverBreak != 0,
		LaserValid:     flags&ScannerFLaser != 0,
		CoordZValid:    flags&ScannerFCoordZ != 0,
		CoordXYValid:   flags&ScannerFCoordXY != 0,
		Overbreak:      int16(binary.BigEndian.Uint16(payload[2:4])),
		Laser:          binary.BigEndian.Uint32(payload[4:8]),
		CoordX:         int32(binary.BigEndian.Uint32(payload[8:12])),
		CoordY:         int32(binary.BigEndian.Uint32(payload[12:16])),
		CoordZ:         int32(binary.BigEndian.Uint32(payload[16:20])),
	}, true
}

// PackScannerData packs ScannerData into a buffer.
func PackScannerData(s ScannerData, buf []byte) int {
	if len(buf) < ScannerFrameSize {
		return -1
	}
	buf[0] = DataHandler
	buf[1] = 0
	if s.OverbreakValid {
		buf[1] |= ScannerFOverBreak
	}
	if s.LaserValid {
		buf[1] |= ScannerFLaser
	}
	if s.CoordZValid {
		buf[1] |= ScannerFCoordZ
	}
	if s.CoordXYValid {
		buf[1] |= ScannerFCoordXY
	}
	binary.BigEndian.PutUint16(buf[2:], uint16(s.Overbreak))
	binary.BigEndian.PutUint32(buf[4:], s.Laser)
	binary.BigEndian.PutUint32(buf[8:], uint32(s.CoordX))
	binary.BigEndian.PutUint32(buf[12:], uint32(s.CoordY))
	binary.BigEndian.PutUint32(buf[16:], uint32(s.CoordZ))
	return ScannerFrameSize
}

// BuildFrame constructs a LoRa wire-protocol frame for sending.
// Wire format: [NID_BE(4)][0xAA][0x55][NID_BE(4)][LEN_BE(2)][DATA][CRC16_BE(2)][\r\n]
// The first 4 bytes are the gateway prefix (NID).
func BuildFrame(nid uint32, data []byte) []byte {
	dataLen := len(data)
	// total: prefix(4) + hdr(2) + nid(4) + len(2) + data + crc(2) + crlf(2)
	totalLen := 4 + 2 + 4 + 2 + dataLen + 2 + 2
	buf := make([]byte, totalLen)

	idx := 0
	// Gateway NID prefix
	binary.BigEndian.PutUint32(buf[idx:], nid)
	idx += 4
	// Frame header
	buf[idx] = FrameHdr1
	idx++
	buf[idx] = FrameHdr2
	idx++
	// NID
	binary.BigEndian.PutUint32(buf[idx:], nid)
	idx += 4
	// LEN
	binary.BigEndian.PutUint16(buf[idx:], uint16(dataLen))
	idx += 2
	// DATA
	if dataLen > 0 {
		copy(buf[idx:], data)
		idx += dataLen
	}
	// CRC16-CCITT over NID(4) + LEN(2) + DATA, seed=0
	crc := crc16CCITT(0, buf[6:6+4+2+dataLen])
	binary.BigEndian.PutUint16(buf[idx:], crc)
	idx += 2
	buf[idx] = '\r'
	idx++
	buf[idx] = '\n'

	return buf
}

// crc16CCITT computes CRC16-CCITT matching the C implementation in loralib/crc16.c.
// e and f are uint8_t (truncated to 8 bits), seed is uint16_t.
func crc16CCITT(seed uint16, data []byte) uint16 {
	for _, b := range data {
		e := uint8(seed ^ uint16(b))
		f := uint8(e ^ (e << 4))
		seed = (seed >> 8) ^ (uint16(f) << 8) ^ (uint16(f) << 3) ^ (uint16(f) >> 4)
	}
	return seed
}

// DeviceInfo represents a discovered LoRa gateway device.
type DeviceInfo struct {
	MAC     string `json:"mac"`
	Name    string `json:"name"`
	Version string `json:"version"`
	IP      string `json:"ip"`
}

// NetParams represents network parameters of a LoRa gateway.
type NetParams struct {
	IP      string `json:"ip"`
	Mask    string `json:"mask"`
	Gateway string `json:"gateway"`
}
