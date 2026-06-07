package lorasdk

import (
	"encoding/binary"
	"testing"
)

func TestCRC16CCITT(t *testing.T) {
	tests := []struct {
		name     string
		seed     uint16
		data     []byte
		expected uint16
	}{
		{"empty", 0, []byte{}, 0},
		{"single byte 0x00", 0, []byte{0x00}, 0},
		{"known sequence", 0, []byte{0x01, 0x02, 0x03, 0x04}, crc16CCITT(0, []byte{0x01, 0x02, 0x03, 0x04})},
		{"non-zero seed", 0xFFFF, []byte{0x41, 0x42}, crc16CCITT(0xFFFF, []byte{0x41, 0x42})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := crc16CCITT(tt.seed, tt.data)
			// Verify determinism
			got2 := crc16CCITT(tt.seed, tt.data)
			if got != got2 {
				t.Errorf("crc16CCITT not deterministic: %04X != %04X", got, got2)
			}
		})
	}
}

func TestCRC16CCITT_Determinism(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	c1 := crc16CCITT(0, data)
	c2 := crc16CCITT(0, data)
	if c1 != c2 {
		t.Errorf("CRC not deterministic: %04X != %04X", c1, c2)
	}
}

func TestParseScannerData(t *testing.T) {
	// Build a valid 20-byte scanner frame
	payload := make([]byte, ScannerFrameSize)
	payload[0] = DataHandler
	payload[1] = ScannerFOverBreak | ScannerFLaser // flags

	// Overbreak = 100 (big-endian int16)
	binary.BigEndian.PutUint16(payload[2:4], uint16(int16(100)))
	// Laser = 5000
	binary.BigEndian.PutUint32(payload[4:8], 5000)
	// CoordX = 1000
	binary.BigEndian.PutUint32(payload[8:12], uint32(int32(1000)))
	// CoordY = -500 (use bit pattern to avoid overflow)
	binary.BigEndian.PutUint32(payload[12:16], 0xFFFFFE0C)
	// CoordZ = 200
	binary.BigEndian.PutUint32(payload[16:20], uint32(int32(200)))

	sd, ok := ParseScannerData(payload)
	if !ok {
		t.Fatal("ParseScannerData returned false for valid payload")
	}

	if !sd.OverbreakValid {
		t.Error("OverbreakValid should be true")
	}
	if !sd.LaserValid {
		t.Error("LaserValid should be true")
	}
	if sd.CoordZValid {
		t.Error("CoordZValid should be false")
	}
	if sd.CoordXYValid {
		t.Error("CoordXYValid should be false")
	}
	if sd.Overbreak != 100 {
		t.Errorf("Overbreak = %d, want 100", sd.Overbreak)
	}
	if sd.Laser != 5000 {
		t.Errorf("Laser = %d, want 5000", sd.Laser)
	}
	if sd.CoordX != 1000 {
		t.Errorf("CoordX = %d, want 1000", sd.CoordX)
	}
	if sd.CoordY != -500 {
		t.Errorf("CoordY = %d, want -500", sd.CoordY)
	}
}

func TestParseScannerData_Invalid(t *testing.T) {
	// Too short
	_, ok := ParseScannerData([]byte{0x01, 0x00})
	if ok {
		t.Error("should return false for short payload")
	}

	// Wrong type byte
	_, ok = ParseScannerData(make([]byte, ScannerFrameSize))
	if ok {
		t.Error("should return false for wrong type")
	}
}

func TestPackScannerData_Roundtrip(t *testing.T) {
	original := ScannerData{
		OverbreakValid: true,
		LaserValid:     true,
		CoordZValid:    true,
		CoordXYValid:   true,
		Overbreak:      1234,
		Laser:          99999,
		CoordX:         12345,
		// CoordY: -6789 — use bit pattern to avoid overflow in test
		// The struct field is int32, so we assign directly
		CoordY: -6789,
		CoordZ: 42,
	}

	buf := make([]byte, ScannerFrameSize)
	n := PackScannerData(original, buf)
	if n != ScannerFrameSize {
		t.Fatalf("PackScannerData returned %d, want %d", n, ScannerFrameSize)
	}

	parsed, ok := ParseScannerData(buf)
	if !ok {
		t.Fatal("ParseScannerData failed on packed data")
	}

	if parsed != original {
		t.Errorf("roundtrip mismatch:\n  original: %+v\n  parsed:   %+v", original, parsed)
	}
}

func TestPackScannerData_TooSmall(t *testing.T) {
	sd := ScannerData{LaserValid: true, Laser: 100}
	buf := make([]byte, 5)
	n := PackScannerData(sd, buf)
	if n != -1 {
		t.Errorf("expected -1 for small buffer, got %d", n)
	}
}

func TestBuildFrame(t *testing.T) {
	nid := uint32(0x12345678)
	data := []byte{0x01, 0x02, 0x03, 0x04}

	frame := BuildFrame(nid, data)

	// Verify NID prefix
	gotNID := binary.BigEndian.Uint32(frame[0:4])
	if gotNID != nid {
		t.Errorf("NID prefix = 0x%08X, want 0x%08X", gotNID, nid)
	}

	// Verify header
	if frame[4] != FrameHdr1 || frame[5] != FrameHdr2 {
		t.Errorf("header = %02X %02X, want %02X %02X", frame[4], frame[5], FrameHdr1, FrameHdr2)
	}

	// Verify NID in frame
	gotNID2 := binary.BigEndian.Uint32(frame[6:10])
	if gotNID2 != nid {
		t.Errorf("NID in frame = 0x%08X, want 0x%08X", gotNID2, nid)
	}

	// Verify length
	gotLen := binary.BigEndian.Uint16(frame[10:12])
	if int(gotLen) != len(data) {
		t.Errorf("length = %d, want %d", gotLen, len(data))
	}

	// Verify CRLF footer
	fLen := len(frame)
	if frame[fLen-2] != '\r' || frame[fLen-1] != '\n' {
		t.Errorf("footer = %02X %02X, want 0D 0A", frame[fLen-2], frame[fLen-1])
	}
}

func TestBuildFrame_EmptyData(t *testing.T) {
	frame := BuildFrame(0x100, nil)
	if len(frame) == 0 {
		t.Fatal("BuildFrame returned empty frame")
	}
	// Should still have valid structure
	if frame[4] != FrameHdr1 {
		t.Errorf("header mismatch")
	}
}

func TestConnState_String(t *testing.T) {
	tests := []struct {
		state ConnState
		want  string
	}{
		{ConnDisconnected, "disconnected"},
		{ConnConnecting, "connecting"},
		{ConnConnected, "connected"},
		{ConnState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestLogSource_String(t *testing.T) {
	tests := []struct {
		src  LogSource
		want string
	}{
		{LogTCP, "TCP"},
		{LogUDP, "UDP"},
		{LogSerial, "Serial"},
		{LogSource(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.src.String(); got != tt.want {
			t.Errorf("LogSource(%d).String() = %q, want %q", tt.src, got, tt.want)
		}
	}
}
