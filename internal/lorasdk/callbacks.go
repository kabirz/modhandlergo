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

// CallbackRouter routes callbacks to typed channel consumers.
// Useful when multiple UI pages need events from the same SDK.
type CallbackRouter struct {
	onConnState   func(ConnState)
	onFrame       func(uint32, []byte)
	onDeviceFound func(string, string, string, string)
	onATResponse  func(string)
	onNetParams   func(string, string, string)
	onLog         func(string, LogSource)
	onHexDump     func(string, []byte)
	onError       func(string, LogSource)
}

func (r *CallbackRouter) OnConnState(state ConnState) {
	if r.onConnState != nil {
		r.onConnState(state)
	}
}

func (r *CallbackRouter) OnFrame(nid uint32, payload []byte) {
	if r.onFrame != nil {
		r.onFrame(nid, payload)
	}
}

func (r *CallbackRouter) OnDeviceFound(mac, deviceName, swVersion, fromIP string) {
	if r.onDeviceFound != nil {
		r.onDeviceFound(mac, deviceName, swVersion, fromIP)
	}
}

func (r *CallbackRouter) OnATResponse(response string) {
	if r.onATResponse != nil {
		r.onATResponse(response)
	}
}

func (r *CallbackRouter) OnNetParams(ip, mask, gateway string) {
	if r.onNetParams != nil {
		r.onNetParams(ip, mask, gateway)
	}
}

func (r *CallbackRouter) OnLog(message string, source LogSource) {
	if r.onLog != nil {
		r.onLog(message, source)
	}
}

func (r *CallbackRouter) OnHexDump(prefix string, data []byte) {
	if r.onHexDump != nil {
		r.onHexDump(prefix, data)
	}
}

func (r *CallbackRouter) OnError(message string, source LogSource) {
	if r.onError != nil {
		r.onError(message, source)
	}
}

// SetConnStateHandler sets the connection state callback.
func (r *CallbackRouter) SetConnStateHandler(fn func(ConnState)) {
	r.onConnState = fn
}

// SetFrameHandler sets the frame received callback.
func (r *CallbackRouter) SetFrameHandler(fn func(uint32, []byte)) {
	r.onFrame = fn
}

// SetDeviceFoundHandler sets the device found callback.
func (r *CallbackRouter) SetDeviceFoundHandler(fn func(string, string, string, string)) {
	r.onDeviceFound = fn
}

// SetATResponseHandler sets the AT response callback.
func (r *CallbackRouter) SetATResponseHandler(fn func(string)) {
	r.onATResponse = fn
}

// SetNetParamsHandler sets the network params callback.
func (r *CallbackRouter) SetNetParamsHandler(fn func(string, string, string)) {
	r.onNetParams = fn
}

// SetLogHandler sets the log callback.
func (r *CallbackRouter) SetLogHandler(fn func(string, LogSource)) {
	r.onLog = fn
}

// SetHexDumpHandler sets the hex dump callback.
func (r *CallbackRouter) SetHexDumpHandler(fn func(string, []byte)) {
	r.onHexDump = fn
}

// SetErrorHandler sets the error callback.
func (r *CallbackRouter) SetErrorHandler(fn func(string, LogSource)) {
	r.onError = fn
}
