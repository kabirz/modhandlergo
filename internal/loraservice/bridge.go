package loraservice

import (
	"math/rand"

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
func (b *Bridge) OnATResponse(response string) {
	b.emit("lora:atresponse", response)
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
