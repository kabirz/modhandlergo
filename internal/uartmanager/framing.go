// Package uartmanager provides serial port firmware upgrade functionality.
package uartmanager

import (
	"fmt"
)

// UART frame format constants
const (
	FrameHead     = 0xAA
	FrameTail     = 0x55
	FrameTypeCmd  = 0x01
	FrameTypeData = 0x02
)

// MaxDataLen is the maximum data payload in a UART frame.
const MaxDataLen = 8

// SerialPortInfo describes a detected serial port.
type SerialPortInfo struct {
	PortName     string `json:"portName"`
	FriendlyName string `json:"friendlyName"`
}

// CalcCRC16 computes CRC16 with init=0xFFFF, polynomial=0xA001.
// Matches the C version's CalcCRC16 exactly.
func CalcCRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for i := range data {
		crc ^= uint16(data[i])
		for j := 0; j < 8; j++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// BuildFrame assembles a UART frame: HEAD + TYPE + LEN_BE(2) + DATA + CRC16_BE(2) + TAIL.
func BuildFrame(frameType byte, data []byte, out []byte) (int, error) {
	dataLen := len(data)
	if dataLen > MaxDataLen {
		return 0, fmt.Errorf("data length %d exceeds max %d", dataLen, MaxDataLen)
	}

	idx := 0
	out[idx] = FrameHead
	idx++
	out[idx] = frameType
	idx++
	// Length in big-endian
	out[idx] = byte(dataLen >> 8)
	idx++
	out[idx] = byte(dataLen)
	idx++
	// Data
	if dataLen > 0 {
		copy(out[idx:], data)
		idx += dataLen
	}
	// CRC16 in big-endian
	crc := CalcCRC16(data[:dataLen])
	out[idx] = byte(crc >> 8)
	idx++
	out[idx] = byte(crc)
	idx++
	out[idx] = FrameTail
	idx++

	return idx, nil
}

// ParseFrame attempts to parse a UART frame from the buffer.
// Returns the frame type, data, number of bytes consumed, and any error.
// If the frame is incomplete, returns (0, nil, 0, nil).
// If the frame is invalid, returns a negative consumed count indicating bytes to discard.
func ParseFrame(buf []byte) (frameType byte, data []byte, consumed int, err error) {
	bufLen := len(buf)
	if bufLen < 7 { // minimum: HEAD(1)+TYPE(1)+LEN(2)+CRC(2)+TAIL(1)
		return 0, nil, 0, nil
	}

	// Find frame head
	idx := 0
	for idx < bufLen && buf[idx] != FrameHead {
		idx++
	}
	if idx >= bufLen-6 {
		return 0, nil, 0, nil
	}

	frameStart := idx
	idx++ // skip HEAD

	fType := buf[idx]
	idx++

	dataLen := int(buf[idx])<<8 | int(buf[idx+1])
	idx += 2

	if dataLen > MaxDataLen {
		return 0, nil, -(frameStart + 1), fmt.Errorf("invalid data length %d", dataLen)
	}

	totalLen := 1 + 1 + 2 + dataLen + 2 + 1
	if frameStart+totalLen > bufLen {
		return 0, nil, 0, nil // incomplete
	}

	// Extract data
	frameData := make([]byte, dataLen)
	if dataLen > 0 {
		copy(frameData, buf[idx:idx+dataLen])
	}
	idx += dataLen

	// Verify CRC
	recvCRC := uint16(buf[idx])<<8 | uint16(buf[idx+1])
	idx += 2
	calcCRC := CalcCRC16(frameData[:dataLen])
	if recvCRC != calcCRC {
		return 0, nil, -(frameStart + 1), fmt.Errorf("CRC mismatch")
	}

	// Verify tail
	if buf[idx] != FrameTail {
		return 0, nil, -(frameStart + 1), fmt.Errorf("invalid tail byte")
	}

	return fType, frameData, totalLen, nil
}

// DecodeResponse parses a UART response frame's data payload into code+val.
func DecodeResponse(data []byte) (code uint32, val uint32) {
	if len(data) < 8 {
		return 0, 0
	}
	code = uint32(data[3])<<24 | uint32(data[2])<<16 | uint32(data[1])<<8 | uint32(data[0])
	val = uint32(data[7])<<24 | uint32(data[6])<<16 | uint32(data[5])<<8 | uint32(data[4])
	return
}

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
