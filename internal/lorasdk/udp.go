package lorasdk

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	udpPort    = 1566
	udpTimeout = 5 * time.Second
)

// UDPClient handles UDP device discovery and AT command transport.
// Uses USR1566 JSON protocol matching the USR-LG210-L gateway.
type UDPClient struct {
	mu       sync.Mutex
	cb       Callbacks
	devMAC   string
	devGWID  string
	localIP  string // known local interface IP — re-use after first success
}

func NewUDPClient(cb Callbacks) *UDPClient {
	return &UDPClient{cb: cb}
}

func (u *UDPClient) SetLocalIP(ip string) { u.localIP = ip }

// wrapJSON wraps a JSON object in USR1566 protocol: "USR1566" + JSON + "USR1566"
func wrapJSON(root map[string]interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	return []byte("USR1566" + string(jsonBytes) + "USR1566"), nil
}

// unwrapJSON extracts JSON substring from a USR1566 wrapped response.
func unwrapJSON(data string) string {
	start := strings.Index(data, "{")
	end := strings.LastIndex(data, "}")
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	return data[start : end+1]
}

// udpSendCore is the core UDP send+receive matching C's udp_worker.
// Creates sockets bound to each local interface, broadcasts payload,
// collects responses, closes all sockets.
func (u *UDPClient) udpSendCore(payload []byte) []string {
	// Collect local IPv4 addresses (matching C's GetAdaptersAddresses)
	var localIPs []net.IP

	if u.localIP != "" {
		// Known interface — only use that one (C code: if (sdk->local_if_ip[0]))
		ip := net.ParseIP(u.localIP)
		if ip != nil && ip.To4() != nil {
			localIPs = append(localIPs, ip.To4())
		}
	}

	if len(localIPs) == 0 {
		// Enumerate all active IPv4 interfaces
		ifaces, err := net.Interfaces()
		if err != nil {
			u.cb.OnError(fmt.Sprintf("enum network interfaces failed: %v", err), LogUDP)
			return nil
		}
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok || ipNet.IP.To4() == nil {
					continue
				}
				localIPs = append(localIPs, ipNet.IP.To4())
			}
		}
	}

	if len(localIPs) == 0 {
		u.cb.OnError("no available network interfaces", LogUDP)
		return nil
	}

	// Create a socket per interface, send broadcast, collect into socks slice
	type sockInfo struct {
		conn *net.UDPConn
		ip   string
	}
	var socks []sockInfo
	bcastAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: udpPort}

	for _, ip := range localIPs {
		localAddr := &net.UDPAddr{IP: ip}
		conn, err := net.ListenUDP("udp4", localAddr)
		if err != nil {
			continue
		}

		_, err = conn.WriteToUDP(payload, bcastAddr)
		if err != nil {
			conn.Close()
			continue
		}

		socks = append(socks, sockInfo{conn: conn, ip: ip.String()})
	}

	if len(socks) == 0 {
		u.cb.OnError("send failed", LogUDP)
		return nil
	}

	// Collect responses — stop after first valid response (matches C: got_response break)
	var responses []string
	buf := make([]byte, 2048)
	deadline := time.Now().Add(udpTimeout)

	gotResponse := false
	for time.Now().Before(deadline) && !gotResponse {
		for _, s := range socks {
			s.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			n, remoteAddr, err := s.conn.ReadFromUDP(buf)
			if err != nil || n == 0 {
				continue
			}

			content := string(buf[:n])
			responses = append(responses, content)

			// Record the local interface IP for future use (C code: sdk->local_if_ip)
			if u.localIP == "" {
				u.localIP = s.ip
				u.cb.OnLog(fmt.Sprintf("recorded local interface: %s", s.ip), LogUDP)
			}

			// Record the remote device IP
			if remoteAddr != nil {
				u.cb.OnLog(fmt.Sprintf("response from: %s", remoteAddr.IP.String()), LogUDP)
			}

			gotResponse = true
			break
		}
	}

	// Close all sockets
	for _, s := range socks {
		s.conn.Close()
	}

	return responses
}

// SearchDevices broadcasts a discovery packet and collects responses.
func (u *UDPClient) SearchDevices(ctx interface{}) {
	u.cb.OnLog("Starting LoRa device search...", LogUDP)

	// Reset local IP to force re-enumeration (matches C: sdk->local_if_ip[0] = '\0')
	u.localIP = ""

	payload := map[string]interface{}{
		"VER":  "1.0",
		"MSG":  "SEARCH",
		"TYPE": "LORA",
	}

	data, err := wrapJSON(payload)
	if err != nil {
		u.cb.OnError(fmt.Sprintf("build search packet failed: %v", err), LogUDP)
		return
	}

	responses := u.udpSendCore(data)
	for _, resp := range responses {
		u.processResponse(resp, "")
	}

	if len(responses) == 0 {
		u.cb.OnLog("No devices found (timeout 5s)", LogUDP)
	}

	u.cb.OnLog("Device search complete", LogUDP)
}

// SendAT sends an AT command via UDP wrapped in USR1566 JSON protocol.
func (u *UDPClient) SendAT(atCmd string, gatewayIP string) {
	if !strings.HasSuffix(atCmd, "\r\n") {
		atCmd += "\r\n"
	}

	msgType := "SETPARA"
	if strings.Contains(atCmd, "?") {
		msgType = "GETPARA"
	}

	mac := u.devMAC
	if mac == "" {
		mac = "D4AD20ED63C4"
	}

	payload := map[string]interface{}{
		"VER":  "1.0",
		"MSG":  msgType,
		"TYPE": "AT",
		"CMD":  atCmd,
		"USER": "admin",
		"PSW":  "admin",
		"MAC":  mac,
	}

	data, err := wrapJSON(payload)
	if err != nil {
		u.cb.OnError(fmt.Sprintf("build AT command failed: %v", err), LogUDP)
		return
	}

	u.cb.OnLog(fmt.Sprintf("UDP send: %s", strings.TrimSpace(atCmd)), LogUDP)

	responses := u.udpSendCore(data)
	for _, resp := range responses {
		u.processResponse(resp, gatewayIP)
	}
}

// GetNetParams queries network parameters from a gateway.
func (u *UDPClient) GetNetParams(gatewayIP string) {
	if u.devMAC == "" {
		u.cb.OnError("search devices first", LogUDP)
		return
	}

	payload := map[string]interface{}{
		"VER":  "1.0",
		"MSG":  "GETPARA",
		"TYPE": "JSON",
		"CMD":  "NETDEV",
		"USER": "admin",
		"PSW":  "admin",
		"MAC":  u.devMAC,
	}

	data, err := wrapJSON(payload)
	if err != nil {
		u.cb.OnError(fmt.Sprintf("build network query failed: %v", err), LogUDP)
		return
	}

	u.cb.OnLog("Querying network parameters...", LogUDP)

	responses := u.udpSendCore(data)
	for _, resp := range responses {
		u.processResponse(resp, gatewayIP)
	}
}

// QueryRSSI queries gateway RSSI and sends the response via TCP.
func (u *UDPClient) QueryRSSI(gatewayIP string, tcpClient *TCPClient, nid uint32) {
	if !tcpClient.IsConnected() {
		u.cb.OnError("TCP not connected, cannot query RSSI", LogUDP)
		return
	}

	payload := map[string]interface{}{
		"VER":  "1.0",
		"MSG":  "GETPARA",
		"TYPE": "AT",
		"CMD":  "AT+NINFO?\r\n",
		"USER": "admin",
		"PSW":  "admin",
		"MAC":  u.devMAC,
	}

	data, err := wrapJSON(payload)
	if err != nil {
		return
	}

	responses := u.udpSendCore(data)
	for _, resp := range responses {
		u.processResponse(resp, gatewayIP)
	}
}

// processResponse parses a USR1566 wrapped JSON response and dispatches callbacks.
func (u *UDPClient) processResponse(raw string, fromIP string) {
	jsonStr := unwrapJSON(raw)
	if jsonStr == "" {
		u.cb.OnLog(raw, LogUDP)
		return
	}

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		u.cb.OnLog(fmt.Sprintf("RX <- (JSON parse error): %s", jsonStr), LogUDP)
		return
	}

	msg, _ := root["MSG"].(string)

	switch msg {
	case "ACK-SEARCH":
		mac, _ := root["MAC"].(string)
		dev, _ := root["DEV"].(string)
		sver, _ := root["SVER"].(string)
		if mac != "" {
			u.devMAC = mac
		}
		u.cb.OnDeviceFound(mac, dev, sver, fromIP)
		u.cb.OnLog("Device found!", LogUDP)

	case "ACK-GETPARA", "ACK-SETPARA":
		cmdObj, ok := root["CMD"]
		if !ok {
			return
		}
		switch cmd := cmdObj.(type) {
		case map[string]interface{}:
			ip, _ := cmd["IP"].(string)
			sm, _ := cmd["SM"].(string)
			gw, _ := cmd["GW"].(string)
			if ip != "" {
				u.cb.OnNetParams(ip, sm, gw)
				u.cb.OnLog("Network parameters received", LogUDP)
			}
		case string:
			u.cb.OnATResponse(cmd)
			u.cb.OnLog(fmt.Sprintf("RX <- CMD: %s", cmd), LogUDP)

			// Parse GWID from response
			if idx := strings.Index(cmd, "+GWID:"); idx >= 0 {
				val := strings.TrimSpace(cmd[idx+6:])
				// Remove trailing "OK" or other status
				if spIdx := strings.IndexAny(val, " \r\n"); spIdx >= 0 {
					val = val[:spIdx]
				}
				if val != "" {
					u.devGWID = val
					u.cb.OnLog(fmt.Sprintf("GWID: %s", val), LogUDP)
				}
			}
		}
	}
}
