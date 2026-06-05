package lorasdk

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// TCPClient manages the TCP connection to the LoRa gateway for data streaming.
type TCPClient struct {
	mu       sync.Mutex
	conn     net.Conn
	cb       Callbacks
	cancel   context.CancelFunc
	running  bool
	connState ConnState
}

// NewTCPClient creates a new TCP client.
func NewTCPClient(cb Callbacks) *TCPClient {
	return &TCPClient{
		cb: cb,
	}
}

// Connect establishes a TCP connection and starts the receive loop.
func (t *TCPClient) Connect(ip string, port int) {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	t.cb.OnConnState(ConnConnecting)

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	go func() {
		addr := fmt.Sprintf("%s:%d", ip, port)
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			t.cb.OnError(fmt.Sprintf("TCP 连接失败: %v", err), LogTCP)
			t.cb.OnConnState(ConnDisconnected)
			return
		}

		t.mu.Lock()
		t.conn = conn
		t.running = true
		t.connState = ConnConnected
		t.mu.Unlock()

		t.cb.OnConnState(ConnConnected)
		t.cb.OnLog(fmt.Sprintf("TCP 已连接到 %s", addr), LogTCP)

		t.receiveLoop(ctx)

		t.mu.Lock()
		t.running = false
		t.connState = ConnDisconnected
		t.conn = nil
		t.mu.Unlock()

		t.cb.OnConnState(ConnDisconnected)
	}()
}

// Disconnect closes the TCP connection.
func (t *TCPClient) Disconnect() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
	t.running = false
	t.connState = ConnDisconnected
}

// SendFrame sends a LoRa frame to a node via the TCP connection.
func (t *TCPClient) SendFrame(nid uint32, data []byte) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("TCP 未连接")
	}

	frame := BuildFrame(nid, data)
	_, err := conn.Write(frame)
	return err
}

// SendRSSIResponse sends an RSSI response frame.
func (t *TCPClient) SendRSSIResponse(nid uint32, snrRaw, rssiRaw, testFlag byte) error {
	data := []byte{DataRSSI, testFlag, snrRaw, rssiRaw}
	return t.SendFrame(nid, data)
}

// State returns the current connection state.
func (t *TCPClient) State() ConnState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connState
}

// IsConnected returns whether TCP is connected.
func (t *TCPClient) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

func (t *TCPClient) receiveLoop(ctx context.Context) {
	buf := make([]byte, 4096)
	var rxbuf []byte

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		t.mu.Lock()
		conn := t.conn
		t.mu.Unlock()

		if conn == nil {
			return
		}

		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return // intentional disconnect
			}
			t.cb.OnLog("TCP 连接断开", LogTCP)
			return
		}

		if n > 0 {
			rxbuf = append(rxbuf, buf[:n]...)

			// Parse frames from buffer
			for len(rxbuf) >= FrameOverhead {
				frame, consumed := t.parseFrame(rxbuf)
				if consumed == 0 {
					break // incomplete frame
				}
				if frame != nil {
					t.cb.OnFrame(frame.NID, frame.Payload)
				}
				rxbuf = rxbuf[consumed:]
			}

			// Prevent buffer from growing indefinitely
			if len(rxbuf) > 8192 {
				rxbuf = rxbuf[len(rxbuf)-4096:]
			}
		}
	}
}

type parsedFrame struct {
	NID     uint32
	Payload []byte
}

// parseFrame attempts to parse a single LoRa frame from the buffer.
// Returns the parsed frame and the number of bytes consumed.
func (t *TCPClient) parseFrame(buf []byte) (*parsedFrame, int) {
	// Find frame header: 0xAA 0x55
	start := -1
	for i := 0; i <= len(buf)-2; i++ {
		if buf[i] == FrameHdr1 && buf[i+1] == FrameHdr2 {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, len(buf) // discard everything
	}

	// Discard any bytes before header
	if start > 0 {
		// Keep the header and everything after
		buf = buf[start:]
		start = 0
	}

	// Need at least: HDR(2) + NID(4) + LEN(2) = 8 bytes for header
	if len(buf) < 8 {
		return nil, 0
	}

	nid := binary.BigEndian.Uint32(buf[2:6])
	dataLen := int(binary.BigEndian.Uint16(buf[6:8]))

	// Total frame: HDR(2) + NID(4) + LEN(2) + DATA(dataLen) + CRC(2) + CRLF(2)
	totalLen := 2 + 4 + 2 + dataLen + 2 + 2

	if len(buf) < totalLen {
		return nil, 0 // incomplete
	}

	// Verify CRC
	crcData := buf[6 : 8+dataLen] // NID + LEN + DATA (from after header to before CRC)
	expectedCRC := calcCRC16(buf[2 : 6+dataLen]) // from NID to end of DATA
	actualCRC := binary.BigEndian.Uint16(buf[8+dataLen : 10+dataLen])
	_ = crcData

	if actualCRC != expectedCRC {
		t.cb.OnError("LoRa 帧CRC校验失败", LogTCP)
		return nil, 2 // skip header, try next
	}

	// Verify CRLF footer
	if buf[totalLen-2] != '\r' || buf[totalLen-1] != '\n' {
		return nil, 2 // skip header
	}

	// Extract payload
	var payload []byte
	if dataLen > 0 {
		payload = make([]byte, dataLen)
		copy(payload, buf[8:8+dataLen])
	}

	return &parsedFrame{NID: nid, Payload: payload}, totalLen
}
