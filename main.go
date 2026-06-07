package main

import (
	"embed"
	"log"

	"github.com/kabirz/modhandlergo/internal/lorasdk"
	"github.com/kabirz/modhandlergo/internal/loraservice"
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
	application.RegisterEvent[map[string]interface{}]("lora:log")
	application.RegisterEvent[int]("lora:connstate")
	application.RegisterEvent[map[string]interface{}]("lora:frame")
	application.RegisterEvent[string]("lora:gwid")
	application.RegisterEvent[string]("lora:nwmode")
	application.RegisterEvent[string]("lora:ttmode")
	application.RegisterEvent[string]("lora:dhcp")
	application.RegisterEvent[string]("lora:option")
	application.RegisterEvent[string]("lora:upwid")
	application.RegisterEvent[string]("lora:netip")
	application.RegisterEvent[string]("lora:netmask")
	application.RegisterEvent[string]("lora:netgw")
	application.RegisterEvent[string]("lora:csq")
	application.RegisterEvent[string]("lora:chfreq")
	application.RegisterEvent[string]("lora:spd")
	application.RegisterEvent[string]("lora:pwr")
	application.RegisterEvent[string]("lora:socka")
	application.RegisterEvent[string]("lora:socken")
	application.RegisterEvent[map[string]interface{}]("lora:device")
	application.RegisterEvent[string]("lora:atresponse")
	application.RegisterEvent[map[string]string]("lora:netparams")
	application.RegisterEvent[map[string]interface{}]("can:frame")
	application.RegisterEvent[int]("can:connected")
	application.RegisterEvent[any]("can:disconnected")
}

func main() {
	// Create shared services
	commonSvc := service.NewCommonService()

	// Create LoRa SDK with event-routing callbacks
	loraBridge := loraservice.NewBridge()
	loraSDK := lorasdk.NewSDK(loraBridge)
	loraBridge.SetSDK(loraSDK)

	loraDataSvc := service.NewLoRaDataService(loraSDK)
	loraConfigSvc := service.NewLoRaConfigService(loraSDK)
	canUpgradeSvc := service.NewCANUpgradeService(commonSvc)
	canCommandSvc := service.NewCANCommandService(commonSvc)

	app := application.New(application.Options{
		Name:        "激光测距工具",
		Description: "激光测距系统配套工具",
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

	// Pass app to callback bridge for event emission
	loraBridge.SetApp(app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "激光测距工具",
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
