package service

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ── Frame protocol constants ──

const (
	dataHandler = 0x01
	dataTest    = 0x02
	dataRSSI    = 0x03
)

// ── Gateway simulator config ──

type gatewayConfig struct {
	NID       uint32
	GWID      uint32
	MAC       string
	DevName   string
	SWVer     string
	IP        string
	Mask      string
	GW        string
	DHCP      string
	Option    int
	NWMode    int
	TTMode    int
	WMode     int
	UPWID     string
	SockEN    string
	SockA     string
	CH        map[int]int
	SPD       map[int]int
	PWR       map[int]int
	RSSISNR   int
	RSSIVal   int
}

func defaultConfig() *gatewayConfig {
	return &gatewayConfig{
		NID:     0x00000001,
		GWID:    0x00000005,
		MAC:     "D4AD20ED63C4",
		DevName: "USR-LG210-L",
		SWVer:   "V4.1.7",
		IP:      "127.0.0.1",
		Mask:    "255.255.0.0",
		GW:      "127.0.0.1",
		DHCP:    "ON",
		UPWID:   "OFF",
		SockEN:  "ON,OFF",
		SockA:   "TCPC,192.168.1.100,1883,1234",
		CH:      map[int]int{1: 4700, 2: 4800},
		SPD:     map[int]int{1: 7, 2: 7},
		PWR:     map[int]int{1: 30, 2: 30},
		RSSISNR: 12,
		RSSIVal: -65,
	}
}

// ── CRC16-CCITT ──

func crc16CCITT(seed uint16, data []byte) uint16 {
	crc := seed
	for _, b := range data {
		e := (crc ^ uint16(b)) & 0xFF
		f := (e ^ (e << 4)) & 0xFF
		crc = (crc >> 8) ^ (f << 8) ^ (f << 3) ^ (f >> 4)
	}
	return crc & 0xFFFF
}

// ── Frame build/parse ──

func buildFrame(nid uint32, payload []byte) []byte {
	buf := make([]byte, 6+len(payload)+2)
	binary.BigEndian.PutUint32(buf[0:4], nid)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(payload)))
	copy(buf[6:], payload)
	crc := crc16CCITT(0, buf[:6+len(payload)])
	binary.BigEndian.PutUint16(buf[6+len(payload):], crc)
	return buf
}

func buildRXPacket(nid uint32, payload []byte) []byte {
	frame := buildFrame(nid, payload)
	pkt := make([]byte, 2+len(frame)+2)
	pkt[0] = 0xAA
	pkt[1] = 0x55
	copy(pkt[2:], frame)
	pkt[2+len(frame)] = '\r'
	pkt[2+len(frame)+1] = '\n'
	return pkt
}

type parsedFrame struct {
	NID     uint32
	Payload []byte
	DataLen int
}

func parseToolFrames(buf []byte) ([]parsedFrame, []byte) {
	var frames []parsedFrame
	pos := 0
	for pos < len(buf) {
		idx := bytesIndexOf(buf[pos:], 0xAA, 0x55)
		if idx < 0 {
			break
		}
		idx += pos
		if idx < 4 {
			break
		}
		tail := bytesIndexOf(buf[idx+2:], '\r', '\n')
		if tail < 0 {
			break
		}
		tail += idx + 2
		content := buf[idx+2 : tail]
		frameEnd := tail + 2

		if len(content) >= 8 {
			nid := binary.BigEndian.Uint32(content[0:4])
			dataLen := int(binary.BigEndian.Uint16(content[4:6]))
			total := 6 + dataLen + 2
			if total <= 2048 && len(content) >= total {
				calcCRC := crc16CCITT(0, content[:6+dataLen])
				rxCRC := binary.BigEndian.Uint16(content[6+dataLen : total])
				if calcCRC == rxCRC {
					frames = append(frames, parsedFrame{
						NID:     nid,
						Payload: content[6 : 6+dataLen],
						DataLen: dataLen,
					})
				}
			}
		}
		pos = frameEnd
	}
	return frames, buf[pos:]
}

func bytesIndexOf(data []byte, b1, b2 byte) int {
	for i := 0; i < len(data)-1; i++ {
		if data[i] == b1 && data[i+1] == b2 {
			return i
		}
	}
	return -1
}

// ── GatewaySimService ──

type GatewaySimConfig struct {
	TCPPort int    `json:"tcpPort"`
	UDPPort int    `json:"udpPort"`
	NID     string `json:"nid"`
	GWID    string `json:"gwid"`
}

type GatewaySimService struct {
	app     *application.App
	mu      sync.Mutex
	cfg     *gatewayConfig
	running bool
	cancel  context.CancelFunc
	tcpLn   net.Listener
	udpConn *net.UDPConn
	client  net.Conn
	clientMu sync.Mutex
	clientConnected bool
	stats   struct{ rx, tx, err int }
	autoTele    bool
	autoInterval time.Duration
}

func NewGatewaySimService() *GatewaySimService {
	return &GatewaySimService{}
}

func (s *GatewaySimService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	s.app = application.Get()
	return nil
}

func (s *GatewaySimService) ServiceShutdown() error {
	s.Stop()
	return nil
}

func (s *GatewaySimService) Start(config GatewaySimConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("gateway simulator is already running")
	}

	cfg := defaultConfig()
	if config.NID != "" {
		fmt.Sscanf(config.NID, "%X", &cfg.NID)
	}
	if config.GWID != "" {
		fmt.Sscanf(config.GWID, "%X", &cfg.GWID)
	}

	tcpPort := config.TCPPort
	if tcpPort <= 0 {
		tcpPort = 1234
	}
	udpPort := config.UDPPort
	if udpPort <= 0 {
		udpPort = 1566
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.cfg = cfg
	s.running = true
	s.autoTele = false
	s.autoInterval = 2 * time.Second

	// TCP server
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", tcpPort))
	if err != nil {
		cancel()
		s.running = false
		return fmt.Errorf("TCP listen failed: %w", err)
	}
	s.tcpLn = ln
	s.logf("Gateway simulator started (NID=%08X GWID=%08X)", cfg.NID, cfg.GWID)
	s.logf("[TCP] Listening on port %d", tcpPort)
	go s.tcpLoop(ctx)

	// UDP server
	udpAddr := &net.UDPAddr{Port: udpPort}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		ln.Close()
		cancel()
		s.running = false
		return fmt.Errorf("UDP listen failed: %w", err)
	}
	s.udpConn = udpConn
	s.logf("[UDP] Listening on port %d", udpPort)
	go s.udpLoop(ctx)

	s.emitStatus(true)
	return nil
}

func (s *GatewaySimService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	if s.tcpLn != nil {
		s.tcpLn.Close()
	}
	s.clientMu.Lock()
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
	s.clientMu.Unlock()
	if s.udpConn != nil {
		s.udpConn.Close()
	}
	s.emitStatus(false)
	s.logf("Gateway simulator stopped")
	return nil
}

func (s *GatewaySimService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// IsClientConnected returns whether a TCP client is connected.
func (s *GatewaySimService) IsClientConnected() bool {
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	return s.clientConnected
}

// ── TCP ──

func (s *GatewaySimService) tcpLoop(ctx context.Context) {
	ln := s.tcpLn
	for {
		conn, err := ln.Accept()
		if err != nil {
			if !s.running {
				return
			}
			continue
		}
		s.clientMu.Lock()
		s.client = conn
		s.clientConnected = true
		s.stats = struct{ rx, tx, err int }{}
		s.clientMu.Unlock()
		s.logf("[TCP] Client connected: %s", conn.RemoteAddr())
		s.emitClientState(true)

		go s.autoTeleLoop(ctx)
		s.handleTCPClient(ctx, conn)

		s.clientMu.Lock()
		if s.client == conn {
			s.client = nil
		}
		s.clientConnected = false
		s.clientMu.Unlock()
		s.autoTele = false
		conn.Close()
		s.logf("[TCP] Client disconnected")
		s.emitClientState(false)
	}
}

func (s *GatewaySimService) handleTCPClient(ctx context.Context, conn net.Conn) {
	rxBuf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	conn.SetReadDeadline(time.Time{})

	for s.running {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, err := conn.Read(tmp)
		if n > 0 {
			rxBuf = append(rxBuf, tmp[:n]...)
			frames, remaining := parseToolFrames(rxBuf)
			rxBuf = remaining
			for _, f := range frames {
				s.processFrame(f)
			}
		}
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
	}
}

func (s *GatewaySimService) processFrame(f parsedFrame) {
	s.mu.Lock()
	s.stats.rx++
	s.mu.Unlock()

	if len(f.Payload) < 1 {
		return
	}
	dtype := f.Payload[0]
	body := f.Payload[1:]
	ts := time.Now().Format("15:04:05")

	switch dtype {
	case dataHandler:
		if len(body) == 8 {
			// Joystick data (8 bytes)
			x := float64(int16(binary.BigEndian.Uint16(body[0:2]))) / 10.0
			y := float64(int16(binary.BigEndian.Uint16(body[2:4]))) / 10.0
			btn := "Released"
			if body[4]&0x01 == 0 {
				btn = "Pressed"
			}
			s.logf("[%s] RX Telemetry [%08X] X=%.1f Y=%.1f Btn=%s", ts, f.NID, x, y, btn)
		} else if len(body) >= 19 {
			// Scanner data (20 bytes total including type byte, so body=19)
			flags := body[0]
			overbreak := int16(binary.BigEndian.Uint16(body[1:3]))
			laser := binary.BigEndian.Uint32(body[3:7])
			coordX := int32(binary.BigEndian.Uint32(body[7:11]))
			coordY := int32(binary.BigEndian.Uint32(body[11:15]))
			coordZ := int32(binary.BigEndian.Uint32(body[15:19]))
			s.logf("[%s] RX Scanner [%08X] flags=0x%02X overbreak=%d laser=%d X=%d Y=%d Z=%d",
				ts, f.NID, flags, overbreak, laser, coordX, coordY, coordZ)
		} else {
			s.logf("[%s] RX HANDLER [%08X] %dB: %s", ts, f.NID, len(body),
				strings.ToUpper(hex.EncodeToString(body)))
		}
	case dataRSSI:
		if len(body) == 0 {
			s.logf("[%s] RX RSSI Request [%08X]", ts, f.NID)
			s.sendToClient(f.NID, []byte{dataRSSI, byte(int8(s.cfg.RSSIVal))})
			s.logf("[%s] TX RSSI Response [%08X] %d dBm", ts, f.NID, s.cfg.RSSIVal)
		} else {
			s.logf("[%s] RX RSSI [%08X] %dB: %s", ts, f.NID, len(body),
				strings.ToUpper(hex.EncodeToString(body)))
		}
	default:
		typeName := fmt.Sprintf("0x%02X", dtype)
		if dtype == dataTest {
			typeName = "TEST"
		}
		s.logf("[%s] RX %s [%08X] %dB: %s", ts, typeName, f.NID, len(body),
			strings.ToUpper(hex.EncodeToString(body)))
	}
}

func (s *GatewaySimService) sendToClient(nid uint32, payload []byte) {
	s.clientMu.Lock()
	conn := s.client
	s.clientMu.Unlock()
	if conn == nil {
		return
	}
	pkt := buildRXPacket(nid, payload)
	if _, err := conn.Write(pkt); err != nil {
		s.mu.Lock()
		s.stats.err++
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.stats.tx++
	s.mu.Unlock()
}

func (s *GatewaySimService) autoTeleLoop(ctx context.Context) {
	ticker := time.NewTicker(s.autoInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.autoTele {
				s.sendTelemetry(rand.Intn(501)-250, rand.Intn(501)-250, rand.Intn(2))
			}
		}
	}
}

func (s *GatewaySimService) sendTelemetry(x, y, btn int) {
	data := make([]byte, 9)
	data[0] = dataHandler
	binary.BigEndian.PutUint16(data[1:3], uint16(int16(x)))
	binary.BigEndian.PutUint16(data[3:5], uint16(int16(y)))
	if btn != 0 {
		data[5] = 0x00
	} else {
		data[5] = 0x01
	}
	data[6] = 0xFF
	data[7] = 0xFF
	data[8] = 0xFF
	s.sendToClient(s.cfg.NID, data)
	s.logf("[%s] TX Telemetry [%08X] X=%.1f Y=%.1f",
		time.Now().Format("15:04:05"), s.cfg.NID, float64(x)/10, float64(y)/10)
}

// ── UDP ──

func (s *GatewaySimService) udpLoop(ctx context.Context) {
	buf := make([]byte, 4096)
	for s.running {
		select {
		case <-ctx.Done():
			return
		default:
		}
		s.udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		raw := string(buf[:n])
		s.logf("[%s] [UDP] RX from %s: %s",
			time.Now().Format("15:04:05"), addr, truncate(raw, 80))

		resp := s.handleUDP(raw)
		if resp != nil {
			if _, err := s.udpConn.WriteToUDP(resp, addr); err != nil {
				s.logf("[%s] [UDP] TX to %s failed: %v", time.Now().Format("15:04:05"), addr, err)
			} else {
				s.logf("[%s] [UDP] TX -> %s: response sent",
					time.Now().Format("15:04:05"), addr)
			}
		}
	}
}

func (s *GatewaySimService) handleUDP(raw string) []byte {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(raw[start:end+1]), &root); err != nil {
		return nil
	}
	msg, _ := root["MSG"].(string)

	switch msg {
	case "SEARCH":
		return s.handleSearch()
	case "GETPARA", "SETPARA":
		ackMsg := "ACK-GETPARA"
		if msg == "SETPARA" {
			ackMsg = "ACK-SETPARA"
		}
		cmdType, _ := root["TYPE"].(string)
		switch cmdType {
		case "JSON":
			cmdVal, _ := root["CMD"].(string)
			if cmdVal == "NETDEV" {
				return s.handleGetNet(ackMsg)
			}
		case "AT":
			atCmd, _ := root["CMD"].(string)
			return s.handleAT(atCmd, ackMsg)
		}
	}
	return nil
}

func (s *GatewaySimService) handleSearch() []byte {
	return udpWrap(map[string]any{
		"VER": "1.0", "MSG": "ACK-SEARCH",
		"MAC": s.cfg.MAC, "DEV": s.cfg.DevName,
		"SVER": s.cfg.SWVer, "TYPE": "LORA",
	})
}

func (s *GatewaySimService) handleGetNet(ackMsg string) []byte {
	return udpWrap(map[string]any{
		"VER": "1.0", "MSG": ackMsg,
		"CMD": map[string]string{
			"IP": s.cfg.IP, "SM": s.cfg.Mask, "GW": s.cfg.GW,
		},
	})
}

func (s *GatewaySimService) handleAT(atCmd, ackMsg string) []byte {
	resp := s.simulateAT(strings.TrimSpace(atCmd))
	return udpWrap(map[string]any{
		"VER": "1.0", "MSG": ackMsg, "CMD": resp,
	})
}

func (s *GatewaySimService) simulateAT(cmd string) string {
	c := strings.ToUpper(strings.TrimRight(cmd, "\r\n"))

	switch {
	case c == "AT+VER?":
		return fmt.Sprintf("\r\n+VER:%s\r\n\r\nOK\r\n", s.cfg.SWVer)
	case c == "AT+GWID?":
		return fmt.Sprintf("\r\n+GWID:%08X\r\n\r\nOK\r\n", s.cfg.GWID)
	case c == "AT+CSQ?":
		return "\r\n+CSQ:4,18\r\n\r\nOK\r\n"
	case c == "AT+DHCP?":
		return fmt.Sprintf("\r\n+DHCP:%s\r\n\r\nOK\r\n", s.cfg.DHCP)
	case strings.HasPrefix(c, "AT+DHCP="):
		s.cfg.DHCP = c[len("AT+DHCP="):]
		return "\r\nOK\r\n"
	case c == "AT+GWIP?":
		return fmt.Sprintf("\r\n+GWIP:%s\r\n\r\nOK\r\n", s.cfg.IP)
	case strings.HasPrefix(c, "AT+GWIP="):
		s.cfg.IP = c[len("AT+GWIP="):]
		return "\r\nOK\r\n"
	case c == "AT+MASK?":
		return fmt.Sprintf("\r\n+MASK:%s\r\n\r\nOK\r\n", s.cfg.Mask)
	case strings.HasPrefix(c, "AT+MASK="):
		s.cfg.Mask = c[len("AT+MASK="):]
		return "\r\nOK\r\n"
	case c == "AT+GW?":
		return fmt.Sprintf("\r\n+GW:%s\r\n\r\nOK\r\n", s.cfg.GW)
	case strings.HasPrefix(c, "AT+GW="):
		s.cfg.GW = c[len("AT+GW="):]
		return "\r\nOK\r\n"
	case c == "AT+OPTION?":
		return fmt.Sprintf("\r\n+OPTION:%d\r\n\r\nOK\r\n", s.cfg.Option)
	case strings.HasPrefix(c, "AT+OPTION="):
		fmt.Sscanf(c[len("AT+OPTION="):], "%d", &s.cfg.Option)
		return "\r\nOK\r\n"
	case c == "AT+NWMODE?":
		return fmt.Sprintf("\r\n+NWMODE:%d\r\n\r\nOK\r\n", s.cfg.NWMode)
	case strings.HasPrefix(c, "AT+NWMODE="):
		fmt.Sscanf(c[len("AT+NWMODE="):], "%d", &s.cfg.NWMode)
		return "\r\n+NWMODE:OK\r\n"
	case c == "AT+TTMODE?":
		return fmt.Sprintf("\r\n+TTMODE:%d\r\n\r\nOK\r\n", s.cfg.TTMode)
	case strings.HasPrefix(c, "AT+TTMODE="):
		fmt.Sscanf(c[len("AT+TTMODE="):], "%d", &s.cfg.TTMode)
		return "\r\n+TTMODE:OK\r\n"
	case c == "AT+WMODE?":
		return fmt.Sprintf("\r\n+WMODE:%d\r\n\r\nOK\r\n", s.cfg.WMode)
	case strings.HasPrefix(c, "AT+WMODE="):
		fmt.Sscanf(c[len("AT+WMODE="):], "%d", &s.cfg.WMode)
		return "\r\n+WMODE:OK\r\n"
	case c == "AT+UPWID?":
		return fmt.Sprintf("\r\n+UPWID:%s\r\n", s.cfg.UPWID)
	case strings.HasPrefix(c, "AT+UPWID="):
		val := c[len("AT+UPWID="):]
		if val == "ON" || val == "OFF" {
			s.cfg.UPWID = val
		}
		return fmt.Sprintf("\r\n+UPWID:%s\r\n", s.cfg.UPWID)
	case c == "AT+SOCKEN?":
		return fmt.Sprintf("\r\n+SOCKEN:%s\r\n\r\nOK\r\n", s.cfg.SockEN)
	case strings.HasPrefix(c, "AT+SOCKEN="):
		s.cfg.SockEN = c[len("AT+SOCKEN="):]
		return "\r\n+SOCKEN:OK\r\n"
	case c == "AT+SOCKA?":
		return fmt.Sprintf("\r\n+SOCKA:%s\r\n\r\nOK\r\n", s.cfg.SockA)
	case strings.HasPrefix(c, "AT+SOCKA="):
		s.cfg.SockA = c[len("AT+SOCKA="):]
		return "\r\n+SOCKA:OK\r\n"
	case strings.HasPrefix(c, "AT+CH") && len(c) > 5:
		return s.handleATCh(c[5:])
	case strings.HasPrefix(c, "AT+SPD") && len(c) > 6:
		return s.handleATSpd(c[6:])
	case strings.HasPrefix(c, "AT+PWR") && len(c) > 6:
		return s.handleATPwr(c[6:])
	case c == "AT+NINFO?":
		nidHigh := s.cfg.NID >> 16
		nidLow := s.cfg.NID & 0xFFFF
		return fmt.Sprintf("\r\n+NINFO:%03X,%04X,1,+%03d,+%03d,%08X,00000000,1,2026/04/21-12:00:00,0000000000,000\r\n\r\nOK\r\n",
			nidHigh, nidLow, s.cfg.RSSISNR, abs(s.cfg.RSSIVal), s.cfg.GWID)
	case c == "AT+NID?":
		return fmt.Sprintf("\r\n+NID:%08X\r\n\r\nOK\r\n", s.cfg.NID)
	case strings.HasPrefix(c, "AT+GWID="):
		fmt.Sscanf(c[len("AT+GWID="):], "%X", &s.cfg.GWID)
		return "\r\nOK\r\n"
	case strings.HasPrefix(c, "AT+NID="):
		fmt.Sscanf(c[len("AT+NID="):], "%X", &s.cfg.NID)
		return "\r\nOK\r\n"
	}
	return "\r\nOK\r\n"
}

func (s *GatewaySimService) handleATCh(rest string) string {
	if before, after, ok := strings.Cut(rest, "="); ok {
		var n, val int
		fmt.Sscanf(before, "%d", &n)
		fmt.Sscanf(after, "%d", &val)
		s.cfg.CH[n] = val
		return fmt.Sprintf("\r\n+CH%d:OK\r\n", n)
	}
	if strings.HasSuffix(rest, "?") {
		var n int
		fmt.Sscanf(rest[:len(rest)-1], "%d", &n)
		val := s.cfg.CH[n]
		return fmt.Sprintf("\r\n+CH%d:%d\r\n\r\nOK\r\n", n, val)
	}
	return "\r\nOK\r\n"
}

func (s *GatewaySimService) handleATSpd(rest string) string {
	if before, after, ok := strings.Cut(rest, "="); ok {
		var n, val int
		fmt.Sscanf(before, "%d", &n)
		fmt.Sscanf(after, "%d", &val)
		s.cfg.SPD[n] = val
		return fmt.Sprintf("\r\n+SPD%d:OK\r\n", n)
	}
	if strings.HasSuffix(rest, "?") {
		var n int
		fmt.Sscanf(rest[:len(rest)-1], "%d", &n)
		val := s.cfg.SPD[n]
		return fmt.Sprintf("\r\n+SPD%d:%d\r\n\r\nOK\r\n", n, val)
	}
	return "\r\nOK\r\n"
}

func (s *GatewaySimService) handleATPwr(rest string) string {
	if before, after, ok := strings.Cut(rest, "="); ok {
		var n, val int
		fmt.Sscanf(before, "%d", &n)
		fmt.Sscanf(after, "%d", &val)
		s.cfg.PWR[n] = val
		return fmt.Sprintf("\r\n+PWR%d:OK\r\n", n)
	}
	if strings.HasSuffix(rest, "?") {
		var n int
		fmt.Sscanf(rest[:len(rest)-1], "%d", &n)
		val := s.cfg.PWR[n]
		return fmt.Sprintf("\r\n+PWR%d:%d\r\n\r\nOK\r\n", n, val)
	}
	return "\r\nOK\r\n"
}

// ── Commands from frontend ──

func (s *GatewaySimService) SendCommand(cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	// Check running state under lock, then release before I/O
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("gateway simulator is not running")
	}
	action := parts[0]
	autoTele := s.autoTele
	s.mu.Unlock()

	switch action {
	case "telemetry":
		x := rand.Intn(501) - 250
		y := rand.Intn(501) - 250
		btn := rand.Intn(2)
		s.sendTelemetry(x, y, btn)
	case "rssi":
		s.sendToClient(s.cfg.NID, []byte{dataRSSI})
		s.logf("[%s] TX RSSI Request [%08X]", time.Now().Format("15:04:05"), s.cfg.NID)
	case "auto":
		s.mu.Lock()
		if len(parts) > 1 && parts[1] == "off" {
			s.autoTele = false
		} else if len(parts) > 1 && parts[1] == "on" {
			s.autoTele = true
			if len(parts) > 2 {
				var interval float64
				fmt.Sscanf(parts[2], "%f", &interval)
				if interval > 0 {
					s.autoInterval = time.Duration(interval * float64(time.Second))
				}
			}
		} else {
			s.autoTele = !autoTele
		}
		state := "OFF"
		if s.autoTele {
			state = "ON"
		}
		interval := s.autoInterval
		s.mu.Unlock()
		s.logf("Auto telemetry: %s (interval=%s)", state, interval)
	case "stats":
		s.mu.Lock()
		rx, tx, errCnt := s.stats.rx, s.stats.tx, s.stats.err
		auto := s.autoTele
		s.mu.Unlock()
		s.logf("TCP: RX=%d TX=%d ERR=%d", rx, tx, errCnt)
		s.logf("Auto telemetry: %v", auto)
	}
	return nil
}

// ── Helpers ──

func (s *GatewaySimService) logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if s.app != nil {
		s.app.Event.Emit("gateway:sim:log", msg)
	}
}

func (s *GatewaySimService) emitStatus(running bool) {
	if s.app != nil {
		s.app.Event.Emit("gateway:sim:status", running)
	}
}

func (s *GatewaySimService) emitClientState(connected bool) {
	if s.app != nil {
		s.app.Event.Emit("gateway:sim:client", connected)
	}
}

func udpWrap(data map[string]any) []byte {
	j, _ := json.Marshal(data)
	return []byte("USR1566" + string(j) + "USR1566")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

