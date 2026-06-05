package main

import (
	"embed"
	"log"

	"github.com/kabirz/modhandlergo/internal/lorasdk"
	"github.com/kabirz/modhandlergo/service"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register typed events for the binding generator.
	application.RegisterEvent[string]("can:log")
	application.RegisterEvent[int]("can:progress")
	application.RegisterEvent[string]("uart:log")
	application.RegisterEvent[int]("uart:progress")
	application.RegisterEvent[string]("lora:log")
	application.RegisterEvent[lorasdk.ConnState]("lora:connstate")
	application.RegisterEvent[lorasdk.ScannerData]("lora:scanner")
	application.RegisterEvent[map[string]interface{}]("lora:device")
	application.RegisterEvent[string]("lora:atresponse")
	application.RegisterEvent[map[string]string]("lora:netparams")
	application.RegisterEvent[map[string]interface{}]("can:frame")
}

func main() {
	// Create shared services
	commonSvc := service.NewCommonService()

	// Create LoRa SDK with event-routing callbacks
	loraCallbacks := &loraCallbackBridge{}
	loraSDK := lorasdk.NewSDK(loraCallbacks)

	loraDataSvc := service.NewLoRaDataService(loraSDK)
	loraConfigSvc := service.NewLoRaConfigService(loraSDK)
	canUpgradeSvc := service.NewCANUpgradeService(commonSvc)
	canCommandSvc := service.NewCANCommandService(commonSvc)

	// Store app reference in callbacks for event emission
	loraCallbacks.appReady = func(app *application.App) {
		loraCallbacks.app = app
	}

	app := application.New(application.Options{
		Name:        "ModHandlerGo",
		Description: "ModHandler PC Tool — 激光测距系统配套工具",
		Services: []application.Service{
			application.NewService(commonSvc),
			application.NewService(loraDataSvc),
			application.NewService(loraConfigSvc),
			application.NewService(canUpgradeSvc),
			application.NewService(canCommandSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Pass app to callback bridge
	loraCallbacks.app = app

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "ModHandlerGo",
		Width:  1280,
		Height: 800,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(24, 24, 27),
		URL:               "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// loraCallbackBridge routes LoRa SDK callbacks to Wails events.
type loraCallbackBridge struct {
	app     *application.App
	appReady func(*application.App)
}

func (b *loraCallbackBridge) OnConnState(state lorasdk.ConnState) {
	if b.app != nil {
		b.app.Event.Emit("lora:connstate", int(state))
	}
}

func (b *loraCallbackBridge) OnFrame(nid uint32, payload []byte) {
	if b.app != nil {
		data := map[string]interface{}{
			"nid":     nid,
			"payload": payload,
		}
		b.app.Event.Emit("lora:frame", data)
	}
}

func (b *loraCallbackBridge) OnDeviceFound(mac, deviceName, swVersion, fromIP string) {
	if b.app != nil {
		data := map[string]interface{}{
			"mac":     mac,
			"name":    deviceName,
			"version": swVersion,
			"ip":      fromIP,
		}
		b.app.Event.Emit("lora:device", data)
	}
}

func (b *loraCallbackBridge) OnATResponse(response string) {
	if b.app != nil {
		b.app.Event.Emit("lora:atresponse", response)
	}
}

func (b *loraCallbackBridge) OnNetParams(ip, mask, gateway string) {
	if b.app != nil {
		data := map[string]string{
			"ip":      ip,
			"mask":    mask,
			"gateway": gateway,
		}
		b.app.Event.Emit("lora:netparams", data)
	}
}

func (b *loraCallbackBridge) OnLog(message string, source lorasdk.LogSource) {
	if b.app != nil {
		b.app.Event.Emit("lora:log", message)
	}
}

func (b *loraCallbackBridge) OnHexDump(prefix string, data []byte) {
	if b.app != nil {
		b.app.Event.Emit("lora:hexdump", map[string]interface{}{
			"prefix": prefix,
			"data":   data,
		})
	}
}

func (b *loraCallbackBridge) OnError(message string, source lorasdk.LogSource) {
	if b.app != nil {
		b.app.Event.Emit("lora:log", "[ERROR] "+message)
	}
}
