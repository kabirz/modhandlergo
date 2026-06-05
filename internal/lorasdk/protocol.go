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
	LogTCP    LogSource = iota
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
	FrameHdr1       = 0xAA
	FrameHdr2       = 0x55
	FrameOverhead   = 10 // HDR(2) + NID(4) + LEN(2) + CRC(2)
	FrameWrapper    = 4  // 2 header + 2 footer (\r\n)
	CRLF            = "\r\n"

	DataHandler = 0x01
	DataTest    = 0x02
	DataRSSI    = 0x03

	ScannerFrameSize = 20
	ScannerFOverBreak = 0x01
	ScannerFLaser     = 0x02
	ScannerFCoordZ    = 0x04
	ScannerFCoordXY   = 0x08
)

// ScannerData represents parsed scanner telemetry data.
type ScannerData struct {
	OverbreakValid bool    `json:"overbreakValid"`
	LaserValid     bool    `json:"laserValid"`
	CoordZValid    bool    `json:"coordZValid"`
	CoordXYValid   bool    `json:"coordXYValid"`
	Overbreak      int16   `json:"overbreak"`
	Laser          uint32  `json:"laser"`
	CoordX         int32   `json:"coordX"`
	CoordY         int32   `json:"coordY"`
	CoordZ         int32   `json:"coordZ"`
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

// BuildFrame constructs a LoRa wire-protocol frame:
// [0xAA][0x55][NID_BE(4)][LEN_BE(2)][DATA][CRC16_BE(2)][\r\n]
func BuildFrame(nid uint32, data []byte) []byte {
	dataLen := len(data)
	totalLen := 2 + 4 + 2 + dataLen + 2 + 2 // hdr + nid + len + data + crc + crlf
	buf := make([]byte, totalLen)

	idx := 0
	buf[idx] = FrameHdr1
	idx++
	buf[idx] = FrameHdr2
	idx++
	binary.BigEndian.PutUint32(buf[idx:], nid)
	idx += 4
	binary.BigEndian.PutUint16(buf[idx:], uint16(dataLen))
	idx += 2
	if dataLen > 0 {
		copy(buf[idx:], data)
		idx += dataLen
	}

	// CRC16 over NID + LEN + DATA
	crc := calcCRC16(buf[6 : 6+4+2+dataLen]) // NID+LEN+DATA starts at offset 6
	// Actually CRC is over the entire content between headers and CRC
	// Re-examine: CRC covers NID+LEN+DATA
	crcData := buf[6 : 6+dataLen] // simplified
	_ = crcData
	crc = calcCRC16(buf[2 : 2+4+2+dataLen]) // from NID to end of DATA
	binary.BigEndian.PutUint16(buf[idx:], crc)
	idx += 2
	buf[idx] = '\r'
	idx++
	buf[idx] = '\n'

	return buf
}

// calcCRC16 computes CRC16-CCITT (init 0xFFFF, polynomial 0xA001).
func calcCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// DeviceInfo represents a discovered LoRa gateway device.
type DeviceInfo struct {
	MAC        string `json:"mac"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	IP         string `json:"ip"`
}

// NetParams represents network parameters of a LoRa gateway.
type NetParams struct {
	IP      string `json:"ip"`
	Mask    string `json:"mask"`
	Gateway string `json:"gateway"`
}
