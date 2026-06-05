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

// parseFrame parses frames from rxbuf matching the C lora_sdk_tcp.c logic:
// 1. Find 0xAA 0x55 header
// 2. Find \r\n as frame terminator
// 3. Content = NID(4) + LEN(2) + DATA + CRC(2)
// 4. Verify CRC over NID+LEN+DATA
func (t *TCPClient) parseFrame(buf []byte) (*parsedFrame, int) {
	// Find header 0xAA 0x55
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

	// Discard bytes before header
	if start > 0 {
		buf = buf[start:]
		start = 0
	}

	// Find \r\n terminator (C code: sdk->tcp_rx_buf[i] == 0x0D && sdk->tcp_rx_buf[i+1] == 0x0A)
	tailPos := -1
	for i := 2; i+1 < len(buf); i++ {
		if buf[i] == 0x0D && buf[i+1] == 0x0A {
			tailPos = i
			break
		}
	}
	if tailPos < 0 {
		// If buffer is too large without finding \r\n, discard header to re-sync
		if len(buf) > 2048 {
			return nil, 2
		}
		return nil, 0 // incomplete, wait for more data
	}

	// Content is between header(2) and \r\n
	// C code: content_len = tail_pos - 2
	contentLen := tailPos - 2
	totalLen := tailPos + 2 // includes \r\n

	// C code: if (content_len >= SDK_FRAME_OVERHEAD)
	// SDK_FRAME_OVERHEAD = NID(4) + LEN(2) + CRC(2) = 8
	if contentLen < 8 {
		return nil, totalLen
	}

	// Parse content: NID(4) + LEN(2) + DATA + CRC(2)
	content := buf[2 : 2+contentLen]
	nid := binary.BigEndian.Uint32(content[0:4])
	dataLen := int(binary.BigEndian.Uint16(content[4:6]))

	// C code: total = SDK_FRAME_HEADER_SIZE + data_len + SDK_FRAME_CRC_SIZE
	// SDK_FRAME_HEADER_SIZE = NID(4) + LEN(2) = 6
	// SDK_FRAME_CRC_SIZE = 2
	expectedContentLen := 6 + dataLen + 2
	if contentLen < expectedContentLen {
		return nil, totalLen
	}
	if expectedContentLen > 2048 {
		return nil, 2 // abnormal frame, re-sync
	}

	// CRC check: CRC-CCITT over NID(4) + LEN(2) + DATA, seed=0
	// C code: crc16_ccitt(0, data, SDK_FRAME_HEADER_SIZE + data_len)
	calcCRC := crc16CCITT(0, content[0:6+dataLen])
	rxCRC := binary.BigEndian.Uint16(content[6+dataLen : 6+dataLen+2])

	if calcCRC != rxCRC {
		t.cb.OnError(fmt.Sprintf("CRC error! calc=%04X rx=%04X", calcCRC, rxCRC), LogTCP)
		return nil, totalLen
	}

	// Extract payload (DATA part, after NID+LEN)
	var payload []byte
	if dataLen > 0 {
		payload = make([]byte, dataLen)
		copy(payload, content[6:6+dataLen])
	}

	return &parsedFrame{NID: nid, Payload: payload}, totalLen
}
