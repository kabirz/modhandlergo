package service

import (
	"context"

	"github.com/kabirz/modhandlergo/internal/lorasdk"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// LoRaConfigService provides LoRa device configuration operations for the frontend.
type LoRaConfigService struct {
	app *application.App
	sdk *lorasdk.SDK
}

// NewLoRaConfigService creates a new LoRa config service.
func NewLoRaConfigService(sdk *lorasdk.SDK) *LoRaConfigService {
	return &LoRaConfigService{sdk: sdk}
}

// ServiceStartup is called when the Wails app starts.
func (s *LoRaConfigService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	s.app = application.Get()
	return nil
}

// SearchDevices broadcasts a discovery packet.
func (s *LoRaConfigService) SearchDevices() {
	s.sdk.SearchDevices()
}

// GetNetParams queries network parameters.
func (s *LoRaConfigService) GetNetParams(gatewayIP string) {
	s.sdk.GetNetParams(gatewayIP)
}

// SendAT sends an AT command via the configured transport.
func (s *LoRaConfigService) SendAT(atCmd string, gatewayIP string) {
	s.sdk.SendAT(atCmd, gatewayIP)
}

// Reboot sends a reboot command to the gateway.
func (s *LoRaConfigService) Reboot(gatewayIP string) {
	s.sdk.Reboot(gatewayIP)
}

// SerialOpen opens a serial port for AT commands.
func (s *LoRaConfigService) SerialOpen(portName string, baudRate int) error {
	return s.sdk.SerialOpen(portName, baudRate)
}

// SerialClose closes the serial port.
func (s *LoRaConfigService) SerialClose() {
	s.sdk.SerialClose()
}

// SerialIsOpen returns whether the serial port is open.
func (s *LoRaConfigService) SerialIsOpen() bool {
	return s.sdk.SerialIsOpen()
}

// SetATTransport selects the AT transport (0=UDP, 1=Serial).
func (s *LoRaConfigService) SetATTransport(transport int) {
	s.sdk.SetATTransport(lorasdk.ATTransport(transport))
}

// GetATTransport returns the current AT transport.
func (s *LoRaConfigService) GetATTransport() int {
	return int(s.sdk.GetATTransport())
}
