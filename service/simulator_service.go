package service

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SimulatorConfig holds parameters for the CAN device simulator.
type SimulatorConfig struct {
	Channel         string  `json:"channel"`
	Version         string  `json:"version"`
	NoHeartbeat     bool    `json:"noHeartbeat"`
	NoHandler       bool    `json:"noHandler"`
	HandlerInterval float64 `json:"handlerInterval"`
}

// SimulatorService manages a CAN device simulator subprocess.
type SimulatorService struct {
	app     *application.App
	common  *CommonService
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	running bool
}

// NewSimulatorService creates a new SimulatorService.
func NewSimulatorService(common *CommonService) *SimulatorService {
	return &SimulatorService{common: common}
}

// ServiceStartup is called when the Wails app starts.
func (s *SimulatorService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	s.app = application.Get()
	return nil
}

// ServiceShutdown stops the simulator if running.
func (s *SimulatorService) ServiceShutdown() error {
	s.Stop()
	return nil
}

// Start launches the CAN device simulator with the given config.
func (s *SimulatorService) Start(config SimulatorConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("simulator is already running")
	}

	// Resolve channel name from CommonService if not provided
	channel := config.Channel
	if channel == "" && s.common != nil {
		chIdx := s.common.GetConnectedChannel()
		if chIdx < 0 {
			return fmt.Errorf("CAN not connected, please connect first")
		}
		if runtime.GOOS == "windows" {
			// PCAN channel value as hex
			channel = fmt.Sprintf("0x%X", chIdx)
		} else {
			iface, err := net.InterfaceByIndex(chIdx)
			if err != nil {
				return fmt.Errorf("failed to get CAN interface: %w", err)
			}
			channel = iface.Name
		}
	}
	if channel == "" {
		return fmt.Errorf("CAN channel not specified")
	}

	scriptPath, err := findSimulatorScript()
	if err != nil {
		return err
	}

	pythonPath, err := findPython()
	if err != nil {
		return err
	}

	args := []string{scriptPath, "--channel", channel}
	if config.Version != "" {
		args = append(args, "--version", config.Version)
	}
	if config.NoHeartbeat {
		args = append(args, "--no-heartbeat")
	}
	if config.NoHandler {
		args = append(args, "--no-handler")
	}
	if config.HandlerInterval > 0 {
		args = append(args, "--handler-interval", fmt.Sprintf("%.2f", config.HandlerInterval))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, pythonPath, args...)
	cmd.Dir = filepath.Dir(scriptPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start simulator: %w", err)
	}

	s.cmd = cmd
	s.cancel = cancel
	s.running = true
	s.emitStatus(true)

	// Stream output to frontend
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			s.emitLog(scanner.Text())
		}
	}()

	// Wait for process exit in background
	go func() {
		_ = cmd.Wait()
		s.mu.Lock()
		s.running = false
		s.cmd = nil
		s.cancel = nil
		s.mu.Unlock()
		s.emitStatus(false)
	}()

	return nil
}

// Stop terminates the running simulator.
func (s *SimulatorService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.cancel == nil {
		return nil
	}
	s.cancel()
	s.running = false
	s.cmd = nil
	s.cancel = nil
	s.emitStatus(false)
	return nil
}

// IsRunning returns whether the simulator is currently running.
func (s *SimulatorService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *SimulatorService) emitLog(msg string) {
	if s.app != nil {
		s.app.Event.Emit("simulator:log", msg)
	}
}

func (s *SimulatorService) emitStatus(running bool) {
	if s.app != nil {
		s.app.Event.Emit("simulator:status", running)
	}
}

// findPython locates a suitable Python interpreter.
func findPython() (string, error) {
	// Check VIRTUAL_ENV first
	if venv := os.Getenv("VIRTUAL_ENV"); venv != "" {
		python := filepath.Join(venv, "bin", "python")
		if _, err := os.Stat(python); err == nil {
			return python, nil
		}
	}

	// Check common locations
	candidates := []string{
		"/home/zed/code/venv/zephyr/bin/python",
		"python3",
		"python",
	}
	if runtime.GOOS == "windows" {
		candidates = []string{"python", "python3"}
	}

	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("python interpreter not found")
}

// findSimulatorScript locates the can_upgrade_sim.py script.
func findSimulatorScript() (string, error) {
	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "scripts", "can_upgrade_sim.py")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Try relative to working directory
	for _, dir := range []string{".", ".."} {
		p := filepath.Join(dir, "scripts", "can_upgrade_sim.py")
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs, nil
		}
	}

	return "", fmt.Errorf("can_upgrade_sim.py not found in scripts/ directory")
}
