package upgrade

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Callbacks provides log and progress notifications during upgrade.
type Callbacks struct {
	OnLog      func(string)
	OnProgress func(int)
}

func (c *Callbacks) log(msg string) {
	if c.OnLog != nil {
		c.OnLog(msg)
	}
}

func (c *Callbacks) progress(pct int) {
	if c.OnProgress != nil {
		c.OnProgress(pct)
	}
}

// GetVersion queries the firmware version via the given transport.
func GetVersion(t Transport) (uint32, error) {
	if !t.IsConnected() {
		return 0, fmt.Errorf("disconnected, please reconnect")
	}

	if err := t.SendCommand(CmdVersion, 0); err != nil {
		return 0, fmt.Errorf("send version query failed: %w", err)
	}

	code, val, err := t.WaitForResponse(5 * time.Second)
	if err != nil {
		return 0, fmt.Errorf("read version response timeout: %w", err)
	}
	if code != FWCodeVersion {
		return 0, fmt.Errorf("read version response data error")
	}
	return val, nil
}

// Reboot sends a reboot command via the given transport.
func Reboot(t Transport) error {
	if !t.IsConnected() {
		return fmt.Errorf("disconnected, please reconnect")
	}
	if err := t.SendCommand(CmdReboot, 0); err != nil {
		return fmt.Errorf("send reboot command failed: %w", err)
	}
	return nil
}

// RunUpgrade performs a complete firmware upgrade via the given transport.
// padByte is 0x00 for CAN, 0xFF for UART.
func RunUpgrade(t Transport, filePath string, testMode bool, padByte byte, cb *Callbacks) error {
	if !t.IsConnected() {
		return fmt.Errorf("disconnected, please reconnect")
	}

	// Open firmware file
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %s", filePath)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("cannot get file info: %w", err)
	}
	fileSize := fi.Size()
	cb.log(fmt.Sprintf("Starting firmware upgrade, size: %d bytes", fileSize))

	// Step 1: Send start update command with file size
	if err := t.SendCommand(CmdStartUpdate, uint32(fileSize)); err != nil {
		return fmt.Errorf("send firmware size failed: %w", err)
	}

	// Step 2: Wait for flash erase
	code, offset, err := t.WaitForResponse(15 * time.Second)
	if err != nil {
		return fmt.Errorf("flash erase timeout")
	}
	if code != FWCodeOffset || offset != 0 {
		return fmt.Errorf("flash erase failed: code(%d), offset(%d)", code, offset)
	}
	cb.log("Flash erase complete")

	// Step 3: Send firmware data 8 bytes at a time
	var bytesSent int64
	var lastPct int
	dataBuf := make([]byte, 8)
	transferComplete := false

	for {
		n, readErr := f.Read(dataBuf)
		if n == 0 {
			break
		}
		if readErr != nil && readErr != io.EOF {
			return fmt.Errorf("read firmware file failed: %w", readErr)
		}

		// Pad remaining bytes
		for i := n; i < 8; i++ {
			dataBuf[i] = padByte
		}

		if err := t.SendData(dataBuf); err != nil {
			return fmt.Errorf("send file data failed: %w", err)
		}

		bytesSent += 8

		// Every 64 bytes or at end, check acknowledgment
		if bytesSent%64 == 0 || bytesSent >= fileSize {
			if pct := int(bytesSent * 100 / fileSize); pct != lastPct {
				cb.progress(pct)
				lastPct = pct
			}

			code, _, err := t.WaitForResponse(5 * time.Second)
			if err != nil {
				return fmt.Errorf("firmware update timeout")
			}
			if code == FWCodeSuccess {
				transferComplete = true
				break
			}
			if code != FWCodeOffset {
				return fmt.Errorf("firmware upgrade failed: code(%d)", code)
			}
		}
	}

	cb.progress(100)

	// Step 4: If not already complete, wait for final transfer confirmation
	if !transferComplete && bytesSent > 0 {
		code, _, err := t.WaitForResponse(5 * time.Second)
		if err != nil {
			return fmt.Errorf("wait firmware transfer complete timeout")
		}
		if code != FWCodeSuccess {
			return fmt.Errorf("firmware transfer failed: code(%d)", code)
		}
	}

	// Step 5: Send confirm command
	confirmVal := uint32(1)
	if testMode {
		confirmVal = 0
	}
	if err := t.SendCommand(CmdConfirm, confirmVal); err != nil {
		return fmt.Errorf("send confirm command failed: %w", err)
	}

	// Step 6: Wait for confirmation
	code, respVal, err := t.WaitForResponse(30 * time.Second)
	if err != nil {
		return fmt.Errorf("firmware confirm timeout")
	}
	if code == FWCodeConfirm && respVal == 0x55AA55AA {
		cb.log(fmt.Sprintf("File %s upload complete. Click reboot, board will restart within 45-60 seconds", filePath))
		return nil
	}
	if code == FWCodeTransferErr {
		return fmt.Errorf("firmware update failed")
	}

	return fmt.Errorf("firmware confirm failed: code(%d), val(0x%08X)", code, respVal)
}

// FormatVersion formats a firmware version uint32 as "vX.Y.Z".
func FormatVersion(version uint32) string {
	return fmt.Sprintf("v%d.%d.%d", (version>>24)&0xFF, (version>>16)&0xFF, (version>>8)&0xFF)
}
