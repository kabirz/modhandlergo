package loraservice

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/kabirz/modhandlergo/internal/lorasdk"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Bridge wraps LoRa SDK callbacks and emits parsed Wails events.
type Bridge struct {
	app            *application.App
	sdk            *lorasdk.SDK
	pendingRSSiNID uint32
}

// NewBridge creates a new LoRa callback bridge.
func NewBridge() *Bridge {
	return &Bridge{}
}

// SetApp sets the Wails application reference for event emission.
func (b *Bridge) SetApp(app *application.App) {
	b.app = app
}

// SetSDK sets the SDK reference for sending frames back (e.g. scanner echo).
func (b *Bridge) SetSDK(sdk *lorasdk.SDK) {
	b.sdk = sdk
}

func (b *Bridge) emit(event string, data interface{}) {
	if b.app != nil {
		b.app.Event.Emit(event, data)
	}
}

// OnConnState implements lorasdk.Callbacks.
func (b *Bridge) OnConnState(state lorasdk.ConnState) {
	b.emit("lora:connstate", int(state))
}

// OnFrame implements lorasdk.Callbacks — parses frame and emits typed events.
func (b *Bridge) OnFrame(nid uint32, payload []byte) {
	if len(payload) == 0 {
		return
	}

	// Emit raw frame for history display
	b.emit("lora:frame", map[string]interface{}{
		"nid":     nid,
		"payload": payload,
	})

	frameType := payload[0]

	switch frameType {
	case lorasdk.DataHandler:
		// Parse joystick telemetry (8 bytes: X(2B BE) + Y(2B BE) + btn(1B) + 0xFF*3)
		if len(payload) >= 9 {
			body := payload[1:9]
			if body[5] == 0xFF && body[6] == 0xFF && body[7] == 0xFF {
				xSigned := int16(uint16(body[0])<<8 | uint16(body[1]))
				ySigned := int16(uint16(body[2])<<8 | uint16(body[3]))
				btn := body[4] & 0x01
				btnStr := "Pressed"
				if btn != 0 {
					btnStr = "Released"
				}
				b.emit("lora:log", map[string]interface{}{
					"msg": fmt.Sprintf("[%s] RX Telemetry NID=0x%08X: X=%.1f° Y=%.1f° Btn=%s",
						time.Now().Format("15:04:05.000"), nid,
						float32(xSigned)/10.0, float32(ySigned)/10.0, btnStr),
					"src": 0,
				})
			}
		}
		// Parse merged scanner frame (20 bytes)
		if sd, ok := lorasdk.ParseScannerData(payload); ok {
			b.emit("lora:scanner", map[string]interface{}{
				"nid":            nid,
					"overbreakValid": sd.OverbreakValid,
					"laserValid":     sd.LaserValid,
					"coordZValid":    sd.CoordZValid,
					"coordXYValid":   sd.CoordXYValid,
					"overbreak":      sd.Overbreak,
					"laser":          sd.Laser,
					"coordX":         sd.CoordX,
					"coordY":         sd.CoordY,
					"coordZ":         sd.CoordZ,
			})
		}
		// Echo: send scanner merged frame back (matches C code behavior)
		b.sendScannerEcho(nid)

	case lorasdk.DataTest:
		// Parse TEST body: [index 2B BE][timestamp 4B BE]
		testDesc := ""
		if len(payload) > 1 {
			body := payload[1:]
			if len(body) >= 6 {
				idx := uint16(body[0])<<8 | uint16(body[1])
				ts := uint32(body[2])<<24 | uint32(body[3])<<16 | uint32(body[4])<<8 | uint32(body[5])
				testDesc = fmt.Sprintf("idx=%d ts=%d ms -> echo", idx, ts)
			} else if len(body) >= 2 {
				idx := uint16(body[0])<<8 | uint16(body[1])
				testDesc = fmt.Sprintf("idx=%d -> echo", idx)
			} else {
				testDesc = "TEST (short)"
			}
		}
		b.emit("lora:log", map[string]interface{}{
			"msg": fmt.Sprintf("[%s] RX Test NID=0x%08X: %s",
				time.Now().Format("15:04:05.000"), nid, testDesc),
			"src": 0,
		})
		b.emit("lora:test", map[string]interface{}{
			"nid":     nid,
			"payload": payload,
		})
		// Echo: send the same payload back to the NID
		b.sendTestEcho(nid, payload)

	case lorasdk.DataRSSI:
		b.emit("lora:log", map[string]interface{}{
			"msg": fmt.Sprintf("[%s] RX RSSI Request NID=0x%08X",
				time.Now().Format("15:04:05.000"), nid),
			"src": 0,
		})
		b.emit("lora:rssi", map[string]interface{}{
			"nid":     nid,
			"payload": payload,
		})
		// Query gateway RSSI and send response back to this NID
		b.queryAndSendRSSI(nid)
	}
}

// OnDeviceFound implements lorasdk.Callbacks.
func (b *Bridge) OnDeviceFound(mac, deviceName, swVersion, fromIP string) {
	b.emit("lora:device", map[string]interface{}{
		"mac":     mac,
		"name":    deviceName,
		"version": swVersion,
		"ip":      fromIP,
	})
}

// OnATResponse implements lorasdk.Callbacks.
// Parses known AT responses and emits typed events (matching C ParseAtResponse).
func (b *Bridge) OnATResponse(response string) {
	b.emit("lora:atresponse", response)

	s := strings.TrimSpace(response)

	// Parse helper: "+PREFIX: value" -> value string
	extract := func(prefix string) string {
		idx := strings.Index(s, prefix)
		if idx < 0 {
			return ""
		}
		val := strings.TrimSpace(s[idx+len(prefix):])
		if spIdx := strings.IndexAny(val, " \r\n"); spIdx >= 0 {
			val = val[:spIdx]
		}
		return val
	}

	if v := extract("+NWMODE:"); v != "" {
		b.emit("lora:nwmode", v)
	}
	if v := extract("+TTMODE:"); v != "" {
		b.emit("lora:ttmode", v)
	}
	if v := extract("+WMODE:"); v != "" {
		b.emit("lora:wmode", v)
	}
	if v := extract("+DHCP:"); v != "" {
		b.emit("lora:dhcp", v)
	}
	if v := extract("+OPTION:"); v != "" {
		b.emit("lora:option", v)
	}
	if v := extract("UPWID:"); v != "" {
		b.emit("lora:upwid", "UPWID: "+v)
	}
	if v := extract("+GWIP:"); v != "" {
		b.emit("lora:netip", v)
	}
	if v := extract("+MASK:"); v != "" {
		b.emit("lora:netmask", v)
	}
	if v := extract("+GW:"); v != "" {
		b.emit("lora:netgw", v)
	}
	if v := extract("+CSQ:"); v != "" {
		b.emit("lora:csq", v)
	}
	if v := extract("+GWID:"); v != "" {
		b.emit("lora:gwid", v)
	}

	// +CH<n>:<freq>, +SPD<n>:<spd>, +PWR<n>:<pwr> — same pattern
	emitTagValue := func(tag string, minLen int, event string) {
		if idx := strings.Index(s, tag); idx >= 0 && len(s) > idx+minLen {
			if colon := strings.Index(s[idx:], ":"); colon >= 0 {
				v := strings.TrimSpace(s[idx+colon+1:])
				if spIdx := strings.IndexAny(v, " \r\n"); spIdx >= 0 {
					v = v[:spIdx]
				}
				if v != "" && v != "OK" {
					b.emit(event, v)
				}
			}
		}
	}
	emitTagValue("+CH", 4, "lora:chfreq")
	emitTagValue("+SPD", 5, "lora:spd")
	emitTagValue("+PWR", 5, "lora:pwr")

	// +SOCKA:<mode>,<ip>,<remote_port>,<local_port>
	if idx := strings.Index(s, "+SOCKA:"); idx >= 0 {
		val := strings.TrimSpace(s[idx+7:])
		b.emit("lora:socka", val)
	}

	// +SOCKEN:<status>,<status>
	if idx := strings.Index(s, "+SOCKEN:"); idx >= 0 {
		val := strings.TrimSpace(s[idx+8:])
		b.emit("lora:socken", val)
	}

	// +NINFO: parse SNR/RSSI and send RSSI response back to pending NID
	if idx := strings.Index(s, "+NINFO:"); idx >= 0 {
		b.handleNINFO(s[idx+7:])
	}
}

// OnNetParams implements lorasdk.Callbacks.
func (b *Bridge) OnNetParams(ip, mask, gateway string) {
	b.emit("lora:netparams", map[string]string{
		"ip":      ip,
		"mask":    mask,
		"gateway": gateway,
	})
}

// OnLog implements lorasdk.Callbacks.
func (b *Bridge) OnLog(message string, source lorasdk.LogSource) {
	b.emit("lora:log", map[string]interface{}{"msg": fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05.000"), message), "src": int(source)})
}

// OnHexDump implements lorasdk.Callbacks.
func (b *Bridge) OnHexDump(prefix string, data []byte) {
	b.emit("lora:hexdump", map[string]interface{}{
		"prefix": prefix,
		"data":   data,
	})
}

// OnError implements lorasdk.Callbacks.
func (b *Bridge) OnError(message string, source lorasdk.LogSource) {
	b.emit("lora:log", map[string]interface{}{"msg": fmt.Sprintf("[%s] [ERROR] %s", time.Now().Format("15:04:05.000"), message), "src": int(source)})
}

// sendScannerEcho sends a simulated scanner merged frame back to the NID.
// Matches C code: random overbreak/laser/coord values, packed with lora_scanner_pack.
// Runs in a goroutine to avoid blocking the receive loop.
func (b *Bridge) sendScannerEcho(nid uint32) {
	if b.sdk == nil {
		return
	}

	go func() {
		sd := lorasdk.ScannerData{
			OverbreakValid: true,
			LaserValid:     true,
			CoordZValid:    true,
			CoordXYValid:   true,
			Overbreak:      int16(rand.IntN(200) - 100),
			Laser:          uint32(rand.IntN(49000) + 1000),
			CoordX:         int32(rand.IntN(10000) - 5000),
			CoordY:         int32(rand.IntN(10000) - 5000),
			CoordZ:         int32(rand.IntN(5000)),
		}
		buf := make([]byte, lorasdk.ScannerFrameSize)
		lorasdk.PackScannerData(sd, buf)
		b.sdk.SendFrame(nid, buf)
	}()
}

// sendTestEcho sends the test payload back to the NID (echo).
func (b *Bridge) sendTestEcho(nid uint32, payload []byte) {
	if b.sdk == nil {
		return
	}
	go func() {
		b.sdk.SendFrame(nid, payload)
	}()
}

// queryAndSendRSSI queries gateway RSSI via UDP and sends the response back to the NID via TCP.
func (b *Bridge) queryAndSendRSSI(nid uint32) {
	if b.sdk == nil {
		return
	}
	b.pendingRSSiNID = nid
	go func() {
		b.sdk.QueryRSSI("", nid)
	}()
}

// handleNINFO parses +NINFO response fields and sends RSSI response back to pending NID.
// Format: field1,field2,field3,snr,rssi (comma-separated, snr=field4, rssi=field5)
func (b *Bridge) handleNINFO(info string) {
	info = strings.TrimSpace(info)
	if info == "" {
		return
	}

	nid := b.pendingRSSiNID
	if nid == 0 || b.sdk == nil {
		return
	}

	snrVal := 0
	rssiVal := -120
	fields := strings.Split(info, ",")
	if len(fields) >= 5 {
		if v, err := strconv.Atoi(strings.TrimSpace(fields[3])); err == nil {
			snrVal = v
		}
		if v, err := strconv.Atoi(strings.TrimSpace(fields[4])); err == nil {
			rssiVal = v
		}
	}

	snrRaw := byte(int8(snrVal))
	rssiRaw := byte(int8(rssiVal))
	testFlag := byte(b.sdk.GetTestFlag())

	b.sdk.SendRSSIResponse(nid, snrRaw, rssiRaw, testFlag)
	b.pendingRSSiNID = 0
	b.emit("lora:log", map[string]interface{}{"msg": fmt.Sprintf("[%s] RSSI response sent: SNR=%d, RSSI=%d", time.Now().Format("15:04:05.000"), snrVal, rssiVal), "src": 0})
}

// Ensure Bridge satisfies lorasdk.Callbacks interface.
var _ lorasdk.Callbacks = (*Bridge)(nil)
