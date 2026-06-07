package lorasdk

// Callbacks defines the interface for LoRa SDK event notifications.
// All methods are called from background goroutines.
// Implementations must not call SDK synchronous I/O methods to avoid deadlocks.
type Callbacks interface {
	OnConnState(state ConnState)
	OnFrame(nid uint32, payload []byte)
	OnDeviceFound(mac, deviceName, swVersion, fromIP string)
	OnATResponse(response string)
	OnNetParams(ip, mask, gateway string)
	OnLog(message string, source LogSource)
	OnHexDump(prefix string, data []byte)
	OnError(message string, source LogSource)
}
