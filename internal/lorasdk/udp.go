package lorasdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	udpSearchPort  = 5200
	udpSearchTimeout = 3 * time.Second
)

// UDPClient handles UDP device discovery and AT command transport.
type UDPClient struct {
	mu      sync.Mutex
	cb      Callbacks
	conn    *net.UDPConn
	localIP string
}

// NewUDPClient creates a new UDP client.
func NewUDPClient(cb Callbacks) *UDPClient {
	return &UDPClient{cb: cb}
}

// SetLocalIP sets the local network interface IP for broadcasting.
func (u *UDPClient) SetLocalIP(ip string) {
	u.localIP = ip
}

// SearchDevices broadcasts a discovery packet and collects responses.
func (u *UDPClient) SearchDevices(ctx context.Context) {
	u.cb.OnLog("开始搜索 LoRa 设备...", LogUDP)

	// Get broadcast address
	broadcastAddr := "255.255.255.255"
	if u.localIP != "" {
		// Derive broadcast from local IP (simple: replace last octet with 255)
		parts := strings.Split(u.localIP, ".")
		if len(parts) == 4 {
			parts[3] = "255"
			broadcastAddr = strings.Join(parts, ".")
		}
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastAddr, udpSearchPort))
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP 地址解析失败: %v", err), LogUDP)
		return
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP 本地地址解析失败: %v", err), LogUDP)
		return
	}

	conn, err := net.DialUDP("udp", localAddr, addr)
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP 连接失败: %v", err), LogUDP)
		return
	}
	defer conn.Close()

	// Send discovery packet
	searchCmd := "AT+SEARCH\r\n"
	_, err = conn.Write([]byte(searchCmd))
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP 搜索包发送失败: %v", err), LogUDP)
		return
	}

	u.cb.OnLog("搜索包已发送，等待响应...", LogUDP)

	// Collect responses
	deadline := time.Now().Add(udpSearchTimeout)
	conn.SetReadDeadline(deadline)

	buf := make([]byte, 4096)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			continue
		}

		if n > 0 {
			response := string(buf[:n])
			u.parseDeviceResponse(response, remoteAddr.IP.String())
		}
	}

	u.cb.OnLog("设备搜索完成", LogUDP)
}

// SendAT sends an AT command via UDP and waits for a response.
func (u *UDPClient) SendAT(atCmd string, gatewayIP string) {
	if !strings.HasSuffix(atCmd, "\r\n") {
		atCmd += "\r\n"
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", gatewayIP, udpSearchPort))
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP 地址解析失败: %v", err), LogUDP)
		return
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP 本地地址解析失败: %v", err), LogUDP)
		return
	}

	conn, err := net.DialUDP("udp", localAddr, addr)
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP 连接失败: %v", err), LogUDP)
		return
	}
	defer conn.Close()

	u.cb.OnLog(fmt.Sprintf("UDP 发送: %s", strings.TrimSpace(atCmd)), LogUDP)

	_, err = conn.Write([]byte(atCmd))
	if err != nil {
		u.cb.OnError(fmt.Sprintf("UDP AT 命令发送失败: %v", err), LogUDP)
		return
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			u.cb.OnError("AT 命令响应超时", LogUDP)
		} else {
			u.cb.OnError(fmt.Sprintf("UDP 读取失败: %v", err), LogUDP)
		}
		return
	}

	if n > 0 {
		response := string(buf[:n])
		u.cb.OnLog(fmt.Sprintf("UDP 响应: %s", strings.TrimSpace(response)), LogUDP)

		// Try to parse as JSON (NETDEV response)
		if strings.Contains(response, "NETDEV") {
			u.parseNetParams(response)
		} else {
			u.cb.OnATResponse(response)
		}
	}
}

// GetNetParams queries network parameters from a gateway.
func (u *UDPClient) GetNetParams(gatewayIP string) {
	u.SendAT("AT+NETDEV?\r\n", gatewayIP)
}

// QueryRSSI queries gateway RSSI and sends the response via TCP.
func (u *UDPClient) QueryRSSI(gatewayIP string, tcpClient *TCPClient, nid uint32) {
	if !tcpClient.IsConnected() {
		u.cb.OnError("TCP 未连接，无法查询 RSSI", LogUDP)
		return
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", gatewayIP, udpSearchPort))
	if err != nil {
		return
	}

	localAddr, _ := net.ResolveUDPAddr("udp", ":0")
	conn, err := net.DialUDP("udp", localAddr, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.Write([]byte("AT+RSSI?\r\n"))

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 256)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return
	}

	// Parse RSSI response
	response := string(buf[:n])
	u.cb.OnLog(fmt.Sprintf("RSSI 响应: %s", strings.TrimSpace(response)), LogUDP)

	// Simple parsing: extract SNR and RSSI values
	// Expected format varies; send as raw response
	// TODO: implement proper RSSI parsing per embedded protocol
	_ = response
}

func (u *UDPClient) parseDeviceResponse(response string, fromIP string) {
	// Try JSON format first
	if strings.HasPrefix(response, "{") {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(response), &result); err == nil {
			mac, _ := result["mac"].(string)
			name, _ := result["name"].(string)
			version, _ := result["sw"].(string)
			u.cb.OnDeviceFound(mac, name, version, fromIP)
			return
		}
	}

	// Try plain text format (AT+SEARCH response)
	// Format: mac,name,version
	parts := strings.Split(strings.TrimSpace(response), ",")
	if len(parts) >= 2 {
		mac := parts[0]
		name := ""
		version := ""
		if len(parts) >= 2 {
			name = parts[1]
		}
		if len(parts) >= 3 {
			version = parts[2]
		}
		u.cb.OnDeviceFound(mac, name, version, fromIP)
	}
}

func (u *UDPClient) parseNetParams(response string) {
	// Try JSON format
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		ip, _ := result["ip"].(string)
		mask, _ := result["mask"].(string)
		gateway, _ := result["gateway"].(string)
		if ip != "" {
			u.cb.OnNetParams(ip, mask, gateway)
			return
		}
	}

	// Try NETDEV text format
	if strings.Contains(response, "NETDEV") {
		// Parse NETDEV response
		lines := strings.Split(response, "\n")
		var ip, mask, gateway string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "+NETDEV:") {
				parts := strings.Split(strings.TrimPrefix(line, "+NETDEV:"), ",")
				if len(parts) >= 3 {
					ip = strings.TrimSpace(parts[0])
					mask = strings.TrimSpace(parts[1])
					gateway = strings.TrimSpace(parts[2])
				}
			}
		}
		if ip != "" {
			u.cb.OnNetParams(ip, mask, gateway)
		}
	}
}
