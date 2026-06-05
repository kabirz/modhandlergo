package loraservice

import (
	"math/rand"
	"strings"

	"github.com/kabirz/modhandlergo/internal/lorasdk"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Bridge wraps LoRa SDK callbacks and emits parsed Wails events.
type Bridge struct {
	app *application.App
	sdk *lorasdk.SDK
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
		b.emit("lora:test", map[string]interface{}{
			"nid":     nid,
			"payload": payload,
		})

	case lorasdk.DataRSSI:
		b.emit("lora:rssi", map[string]interface{}{
			"nid":     nid,
			"payload": payload,
		})
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

	// +CH<n>:<freq>
	if idx := strings.Index(s, "+CH"); idx >= 0 && len(s) > idx+4 {
		if colon := strings.Index(s[idx:], ":"); colon >= 0 {
			v := strings.TrimSpace(s[idx+colon+1:])
			if spIdx := strings.IndexAny(v, " \r\n"); spIdx >= 0 {
				v = v[:spIdx]
			}
			if v != "" && v != "OK" {
				b.emit("lora:chfreq", v)
			}
		}
	}

	// +SPD<n>:<spd>
	if idx := strings.Index(s, "+SPD"); idx >= 0 && len(s) > idx+5 {
		if colon := strings.Index(s[idx:], ":"); colon >= 0 {
			v := strings.TrimSpace(s[idx+colon+1:])
			if spIdx := strings.IndexAny(v, " \r\n"); spIdx >= 0 {
				v = v[:spIdx]
			}
			if v != "" && v != "OK" {
				b.emit("lora:spd", v)
			}
		}
	}

	// +PWR<n>:<pwr>
	if idx := strings.Index(s, "+PWR"); idx >= 0 && len(s) > idx+5 {
		if colon := strings.Index(s[idx:], ":"); colon >= 0 {
			v := strings.TrimSpace(s[idx+colon+1:])
			if spIdx := strings.IndexAny(v, " \r\n"); spIdx >= 0 {
				v = v[:spIdx]
			}
			if v != "" && v != "OK" {
				b.emit("lora:pwr", v)
			}
		}
	}

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
	b.emit("lora:log", message)
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
	b.emit("lora:log", "[ERROR] "+message)
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
			Overbreak:      int16(rand.Intn(200) - 100),
			Laser:          uint32(rand.Intn(49000) + 1000),
			CoordX:         int32(rand.Intn(10000) - 5000),
			CoordY:         int32(rand.Intn(10000) - 5000),
			CoordZ:         int32(rand.Intn(5000)),
		}
		buf := make([]byte, lorasdk.ScannerFrameSize)
		lorasdk.PackScannerData(sd, buf)
		b.sdk.SendFrame(nid, buf)
	}()
}

// Ensure Bridge satisfies lorasdk.Callbacks interface.
var _ lorasdk.Callbacks = (*Bridge)(nil)
