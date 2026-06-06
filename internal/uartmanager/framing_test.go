package uartmanager

import (
	"testing"

	"github.com/kabirz/modhandlergo/internal/upgrade"
)

func TestCalcCRC16(t *testing.T) {
	// Known test vector: CRC16-Modbus of "123456789" = 0x4B37
	data := []byte("123456789")
	crc := CalcCRC16(data)
	// Modbus CRC16 uses polynomial 0xA001 (reversed 0x8005)
	// The standard test vector for Modbus CRC16 of "123456789" is 0x4B37
	if crc != 0x4B37 {
		t.Errorf("CalcCRC16(\"123456789\") = 0x%04X, want 0x4B37", crc)
	}
}

func TestCalcCRC16_Empty(t *testing.T) {
	crc := CalcCRC16(nil)
	if crc != 0xFFFF {
		t.Errorf("CalcCRC16(nil) = 0x%04X, want 0xFFFF", crc)
	}
}

func TestBuildFrame(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	out := make([]byte, 32)

	n, err := BuildFrame(FrameTypeCmd, data, out)
	if err != nil {
		t.Fatalf("BuildFrame error: %v", err)
	}

	// HEAD(1) + TYPE(1) + LEN(2) + DATA(8) + CRC(2) + TAIL(1) = 15
	if n != 15 {
		t.Fatalf("BuildFrame length = %d, want 15", n)
	}

	// Verify head
	if out[0] != FrameHead {
		t.Errorf("head = 0x%02X, want 0x%02X", out[0], FrameHead)
	}
	// Verify type
	if out[1] != FrameTypeCmd {
		t.Errorf("type = 0x%02X, want 0x%02X", out[1], FrameTypeCmd)
	}
	// Verify length (big-endian)
	dataLen := int(out[2])<<8 | int(out[3])
	if dataLen != 8 {
		t.Errorf("data length = %d, want 8", dataLen)
	}
	// Verify tail
	if out[n-1] != FrameTail {
		t.Errorf("tail = 0x%02X, want 0x%02X", out[n-1], FrameTail)
	}
}

func TestBuildFrame_EmptyData(t *testing.T) {
	out := make([]byte, 32)
	n, err := BuildFrame(FrameTypeCmd, nil, out)
	if err != nil {
		t.Fatalf("BuildFrame error: %v", err)
	}
	// HEAD(1) + TYPE(1) + LEN(2) + CRC(2) + TAIL(1) = 7
	if n != 7 {
		t.Errorf("BuildFrame(empty) length = %d, want 7", n)
	}
}

func TestBuildFrame_TooLong(t *testing.T) {
	out := make([]byte, 32)
	data := make([]byte, MaxDataLen+1)
	_, err := BuildFrame(FrameTypeCmd, data, out)
	if err == nil {
		t.Error("expected error for oversized data")
	}
}

func TestParseFrame_Valid(t *testing.T) {
	// Build a valid frame
	data := []byte{0x01, 0x02, 0x03, 0x04}
	out := make([]byte, 32)
	n, err := BuildFrame(FrameTypeData, data, out)
	if err != nil {
		t.Fatalf("BuildFrame error: %v", err)
	}

	fType, frameData, consumed, err := ParseFrame(out[:n])
	if err != nil {
		t.Fatalf("ParseFrame error: %v", err)
	}
	if consumed != n {
		t.Errorf("consumed = %d, want %d", consumed, n)
	}
	if fType != FrameTypeData {
		t.Errorf("frame type = 0x%02X, want 0x%02X", fType, FrameTypeData)
	}
	if len(frameData) != len(data) {
		t.Fatalf("data length = %d, want %d", len(frameData), len(data))
	}
	for i, b := range frameData {
		if b != data[i] {
			t.Errorf("data[%d] = 0x%02X, want 0x%02X", i, b, data[i])
		}
	}
}

func TestParseFrame_Incomplete(t *testing.T) {
	// Only 3 bytes — too short
	_, _, consumed, err := ParseFrame([]byte{0xAA, 0x01, 0x00})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumed != 0 {
		t.Errorf("consumed = %d, want 0 (incomplete)", consumed)
	}
}

func TestParseFrame_NoHead(t *testing.T) {
	// Buffer with no frame head
	buf := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, _, consumed, err := ParseFrame(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumed != 0 {
		t.Errorf("consumed = %d, want 0", consumed)
	}
}

func TestDecodeResponse(t *testing.T) {
	data := []byte{0x01, 0x00, 0x00, 0x00, 0xAA, 0x55, 0xAA, 0x55}
	code, val := DecodeResponse(data)
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if val != 0x55AA55AA {
		t.Errorf("val = 0x%08X, want 0x55AA55AA", val)
	}
}

func TestDecodeResponse_Short(t *testing.T) {
	code, val := DecodeResponse([]byte{0x01, 0x02})
	if code != 0 || val != 0 {
		t.Error("expected zero values for short data")
	}
}

func TestEncodeCommand(t *testing.T) {
	// Test encoding a version query command
	data := upgrade.EncodeCommand(upgrade.CmdVersion, 0)
	if data[0] != byte(upgrade.CmdVersion) || data[1] != 0 || data[2] != 0 || data[3] != 0 {
		t.Errorf("cmd bytes = %v, expected version command", data[:4])
	}
	if data[4] != 0 || data[5] != 0 || data[6] != 0 || data[7] != 0 {
		t.Errorf("param bytes = %v, expected zero", data[4:8])
	}

	// Test encoding a start update with a value
	data = upgrade.EncodeCommand(upgrade.CmdStartUpdate, 0x12345678)
	if data[4] != 0x78 || data[5] != 0x56 || data[6] != 0x34 || data[7] != 0x12 {
		t.Errorf("param bytes = %v, expected little-endian 0x12345678", data[4:8])
	}
}
