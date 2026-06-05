package service

import (
	"context"

	"github.com/kabirz/modhandlergo/internal/lorasdk"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// LoRaDataService provides LoRa data streaming operations for the frontend.
type LoRaDataService struct {
	app *application.App
	sdk *lorasdk.SDK
}

// NewLoRaDataService creates a new LoRa data service.
func NewLoRaDataService(sdk *lorasdk.SDK) *LoRaDataService {
	return &LoRaDataService{sdk: sdk}
}

// ServiceStartup is called when the Wails app starts.
func (s *LoRaDataService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	s.app = application.Get()
	return nil
}

// Connect establishes a TCP connection to the LoRa gateway.
func (s *LoRaDataService) Connect(ip string, port int) {
	s.sdk.Connect(ip, port)
}

// Disconnect closes the TCP connection.
func (s *LoRaDataService) Disconnect() {
	s.sdk.Disconnect()
}

// ConnState returns the current connection state.
func (s *LoRaDataService) ConnState() string {
	return s.sdk.ConnState().String()
}

// IsConnected returns whether TCP is connected.
func (s *LoRaDataService) IsConnected() bool {
	return s.sdk.IsTCPConnected()
}

// SendFrame sends a data frame to a LoRa node.
func (s *LoRaDataService) SendFrame(nid uint32, data []byte) error {
	return s.sdk.SendFrame(nid, data)
}

// QueryRSSI queries RSSI from the gateway.
func (s *LoRaDataService) QueryRSSI(gatewayIP string, nid uint32) {
	s.sdk.QueryRSSI(gatewayIP, nid)
}

// SetTestFlag sets the test mode flag.
func (s *LoRaDataService) SetTestFlag(flag int) {
	s.sdk.SetTestFlag(flag)
}
