package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/kabirz/modhandlergo/internal/canhal"
	"github.com/kabirz/modhandlergo/internal/canmanager"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Firmware response codes (matching embedded mod-can.h fw_error_code)
const (
	fwCodeOffset       = 0
	fwCodeSuccess      = 1
	fwCodeVersion      = 2
	fwCodeConfirm      = 3
	fwCodeFlashError   = 4
	fwCodeTransferErr  = 5
)

// SimulatorConfig holds parameters for the CAN device simulator.
type SimulatorConfig struct {
	Channel         string  `json:"channel"`
	Version         string  `json:"version"`
	NoHeartbeat     bool    `json:"noHeartbeat"`
	NoHandler       bool    `json:"noHandler"`
	HandlerInterval float64 `json:"handlerInterval"`
}

// SimulatorService simulates an embedded CAN device using the shared CAN HAL.
type SimulatorService struct {
	app     *application.App
	common  *CommonService
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	unsub   func()

	// Device state
	version     uint32
	fwOffset    uint32
	fwTotal     uint32
	fwData      []byte
	upgrading   bool

	// LoRa config state
	loraProt uint8
	loraMode uint8
	loraSpd1 uint8
	loraCh1  uint16
	loraSpd2 uint8
	loraCh2  uint16
	loraPnum uint8
	loraNID  uint32
	loraGWID uint32
	loraTest bool
	loraPower bool
}

func NewSimulatorService(common *CommonService) *SimulatorService {
	return &SimulatorService{
		common:     common,
		loraProt:   1,
		loraMode:   2,
		loraSpd1:   7,
		loraCh1:    4700,
		loraSpd2:   7,
		loraCh2:    4800,
		loraNID:    0x01020304,
		loraGWID:   0x11223344,
		loraPower:  true,
	}
}

func (s *SimulatorService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	s.app = application.Get()
	return nil
}

func (s *SimulatorService) ServiceShutdown() error {
	s.Stop()
	return nil
}

func (s *SimulatorService) Start(config SimulatorConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("simulator is already running")
	}
	if s.common == nil {
		return fmt.Errorf("common service not initialized")
	}
	backend := s.common.backend
	dispatcher := s.common.dispatcher
	if backend == nil || dispatcher == nil {
		return fmt.Errorf("CAN HAL not initialized, please connect CAN first")
	}
	if !backend.IsConnected() {
		return fmt.Errorf("CAN not connected, please connect first")
	}

	// Parse version
	var ver uint32
	if config.Version != "" {
		fmt.Sscanf(config.Version, "0x%X", &ver)
	}
	s.version = ver
	s.fwData = nil
	s.fwOffset = 0
	s.fwTotal = 0
	s.upgrading = false

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	// Register callback to receive frames sent by the PC tool
	s.common.setOnFrameSent(func(frame *canhal.Frame) {
		s.onFrame(frame)
	})

	// Periodic heartbeat
	if !config.NoHeartbeat {
		go s.heartbeatLoop(ctx)
	}

	// Periodic handler state
	handlerInterval := 100 * time.Millisecond
	if config.HandlerInterval > 0 {
		handlerInterval = time.Duration(config.HandlerInterval * float64(time.Second))
	}
	if !config.NoHandler {
		go s.handlerStateLoop(ctx, handlerInterval)
	}

	s.emitStatus(true)
	s.log("CAN device simulator started")
	return nil
}

func (s *SimulatorService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	if s.unsub != nil {
		s.unsub()
		s.unsub = nil
	}
	// Clear the frame callback so PC tool frames no longer reach the simulator
	if s.common != nil {
		s.common.setOnFrameSent(nil)
	}
	s.emitStatus(false)
	s.log("CAN device simulator stopped")
	return nil
}

func (s *SimulatorService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// ── Frame handling ──

func (s *SimulatorService) onFrame(frame *canhal.Frame) {
	switch frame.ID {
	case canmanager.PlatformRx:
		s.handleCommand(frame)
	case canmanager.FWDataRx:
		s.handleFWData(frame)
	case canmanager.LoraConfigCmd:
		s.handleLoraConfig(frame)
	}
}

func (s *SimulatorService) handleCommand(frame *canhal.Frame) {
	d := canmanager.DecodeCANFrameData(frame.Data[:])
	ts := time.Now().Format("15:04:05")

	switch d.Code {
	case 0: // START_UPDATE
		s.mu.Lock()
		s.fwTotal = d.Val
		s.fwOffset = 0
		s.fwData = make([]byte, 0, d.Val)
		s.upgrading = true
		s.mu.Unlock()
		s.logf("[%s] RX StartUpdate size=%d", ts, d.Val)
		s.sendResponse(fwCodeOffset, 0)
		s.logf("[%s] Flash erase complete", ts)

	case 1: // CONFIRM
		s.mu.Lock()
		offset := s.fwOffset
		total := s.fwTotal
		s.mu.Unlock()
		if offset < total {
			s.logf("[%s] Confirm failed: offset=%d < total=%d", ts, offset, total)
			s.sendResponse(fwCodeTransferErr, offset)
		} else {
			mode := "test"
			if d.Val == 1 {
				mode = "real"
			}
			s.logf("[%s] Firmware confirmed (%s mode)", ts, mode)
			s.sendResponse(fwCodeConfirm, 0x55AA55AA)
			s.mu.Lock()
			s.upgrading = false
			s.mu.Unlock()
		}

	case 2: // VERSION
		s.mu.Lock()
		ver := s.version
		s.mu.Unlock()
		s.sendResponse(fwCodeVersion, ver)
		s.logf("[%s] Version query -> 0x%08X", ts, ver)

	case 3: // REBOOT
		s.mu.Lock()
		s.upgrading = false
		s.fwData = nil
		s.fwOffset = 0
		s.fwTotal = 0
		s.mu.Unlock()
		s.logf("[%s] Reboot command received", ts)
	}
}

func (s *SimulatorService) handleFWData(frame *canhal.Frame) {
	s.mu.Lock()
	if !s.upgrading {
		s.mu.Unlock()
		return
	}
	s.fwData = append(s.fwData, frame.Data[:frame.DLC]...)
	s.fwOffset += uint32(frame.DLC)
	offset := s.fwOffset
	total := s.fwTotal
	s.mu.Unlock()

	if offset >= total {
		s.sendResponse(fwCodeSuccess, total)
		s.mu.Lock()
		s.upgrading = false
		s.mu.Unlock()
		s.logf("[%s] Firmware transfer complete: %d bytes", time.Now().Format("15:04:05"), total)
	} else if offset%64 == 0 {
		s.sendResponse(fwCodeOffset, offset)
	}
}

func (s *SimulatorService) handleLoraConfig(frame *canhal.Frame) {
	if frame.DLC < 1 {
		return
	}
	cmd := frame.Data[0]
	ts := time.Now().Format("15:04:05")

	s.mu.Lock()
	defer s.mu.Unlock()

	var resp [8]byte
	resp[0] = cmd

	switch cmd {
	case 0x01: // SET_MODE
		s.loraProt = frame.Data[1] >> 4
		s.loraMode = frame.Data[1] & 0x0F
		resp[1] = (s.loraProt << 4) | (s.loraMode & 0x0F)
		s.logf("[%s] LoRa SET_MODE: prot=%d mode=%d", ts, s.loraProt, s.loraMode)

	case 0x02: // QUERY_MODE
		resp[1] = (s.loraProt << 4) | (s.loraMode & 0x0F)
		s.logf("[%s] LoRa QUERY_MODE: prot=%d mode=%d", ts, s.loraProt, s.loraMode)

	case 0x03: // SET_CH1
		s.loraSpd1 = frame.Data[1]
		s.loraCh1 = canmanager.GetBE16(frame.Data[2:4])
		resp[1] = s.loraSpd1
		canmanager.PutBE16(resp[2:4], s.loraCh1)
		s.logf("[%s] LoRa SET_CH1: spd=%d ch=%d", ts, s.loraSpd1, s.loraCh1)

	case 0x04: // QUERY_CH1
		resp[1] = s.loraSpd1
		canmanager.PutBE16(resp[2:4], s.loraCh1)
		s.logf("[%s] LoRa QUERY_CH1: spd=%d ch=%d", ts, s.loraSpd1, s.loraCh1)

	case 0x05: // SET_CH2
		s.loraSpd2 = frame.Data[1]
		s.loraCh2 = canmanager.GetBE16(frame.Data[2:4])
		resp[1] = s.loraSpd2
		canmanager.PutBE16(resp[2:4], s.loraCh2)
		s.logf("[%s] LoRa SET_CH2: spd=%d ch=%d", ts, s.loraSpd2, s.loraCh2)

	case 0x06: // QUERY_CH2
		resp[1] = s.loraSpd2
		canmanager.PutBE16(resp[2:4], s.loraCh2)
		s.logf("[%s] LoRa QUERY_CH2: spd=%d ch=%d", ts, s.loraSpd2, s.loraCh2)

	case 0x07: // QUERY_NID
		canmanager.PutBE32(resp[4:8], s.loraNID)
		s.logf("[%s] LoRa QUERY_NID: 0x%08X", ts, s.loraNID)

	case 0x08: // SET_NID (not supported, just echo)
		canmanager.PutBE32(resp[4:8], s.loraNID)
		s.logf("[%s] LoRa SET_NID: not supported", ts)

	case 0x09: // QUERY_GWID
		canmanager.PutBE32(resp[4:8], s.loraGWID)
		s.logf("[%s] LoRa QUERY_GWID: 0x%08X", ts, s.loraGWID)

	case 0x0A: // SET_GWID
		if frame.DLC >= 8 {
			s.loraGWID = canmanager.GetBE32(frame.Data[4:8])
		}
		canmanager.PutBE32(resp[4:8], s.loraGWID)
		s.logf("[%s] LoRa SET_GWID: 0x%08X", ts, s.loraGWID)

	case 0x0B: // QUERY_PNUM
		resp[1] = s.loraPnum
		s.logf("[%s] LoRa QUERY_PNUM: %d", ts, s.loraPnum)

	case 0x0C: // SET_PNUM
		s.loraPnum = frame.Data[1]
		resp[1] = s.loraPnum
		s.logf("[%s] LoRa SET_PNUM: %d", ts, s.loraPnum)

	case 0x0D: // SET_TEST
		s.loraTest = frame.Data[1] != 0
		if s.loraTest {
			resp[1] = 1
		}
		s.logf("[%s] LoRa SET_TEST: %v", ts, s.loraTest)

	case 0x0F: // SET_POWER
		s.loraPower = frame.Data[1] != 0
		if s.loraPower {
			resp[1] = 1
		}
		s.logf("[%s] LoRa SET_POWER: %v", ts, s.loraPower)

	default:
		s.logf("[%s] LoRa unknown cmd: 0x%02X", ts, cmd)
		return
	}

	s.sendFrame(canmanager.LoraConfigResp, resp[:])
}

// ── Sending ──

func (s *SimulatorService) sendResponse(code, val uint32) {
	time.Sleep(5 * time.Millisecond)
	var data [8]byte
	canmanager.EncodeCANFrameData(canmanager.CANFrameData{Code: code, Val: val}, data[:])
	s.sendFrame(canmanager.PlatformTx, data[:])
}

func (s *SimulatorService) sendFrame(id uint32, data []byte) {
	dispatcher := s.common.dispatcher
	if dispatcher == nil {
		return
	}
	var d [8]byte
	n := copy(d[:], data)
	frame := &canhal.Frame{ID: id, DLC: uint8(n), Data: d}
	dispatcher.FeedFrame(frame)
}

// ── Periodic goroutines ──

func (s *SimulatorService) heartbeatLoop(ctx context.Context) {
	frame := &canhal.Frame{ID: canmanager.Heartbeat, DLC: 1, Data: [8]byte{5}}
	for {
		s.sendFrame(canmanager.Heartbeat, frame.Data[:frame.DLC])
		select {
		case <-ctx.Done():
			return
		case <-time.After(800 * time.Millisecond):
		}
	}
}

func (s *SimulatorService) handlerStateLoop(ctx context.Context, interval time.Duration) {
	t := 0.0
	for {
		x := int16(math.Sin(t*0.5) * 90)
		y := int16(math.Cos(t*0.3) * 60)
		btn := byte(0x01 | (byte(rand.Intn(2)) << 1))

		var data [8]byte
		canmanager.PutBE16(data[0:2], uint16(x))
		canmanager.PutBE16(data[2:4], uint16(y))
		data[4] = btn
		data[5] = 0xFF
		data[6] = 0xFF
		data[7] = 0xFF

		s.sendFrame(canmanager.ControllerState, data[:])

		t += interval.Seconds()
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// ── Helpers ──

func (s *SimulatorService) log(msg string) {
	if s.app != nil {
		s.app.Event.Emit("simulator:log", msg)
	}
}

func (s *SimulatorService) logf(format string, args ...interface{}) {
	s.log(fmt.Sprintf(format, args...))
}

func (s *SimulatorService) emitStatus(running bool) {
	if s.app != nil {
		s.app.Event.Emit("simulator:status", running)
	}
}
