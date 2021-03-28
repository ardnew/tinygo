package usb

type (
	status int8
	mode   int8

	port struct {
		mode   mode
		device device
		host   host
	}
)

// Enumerated constant values of type status represent all available USB
// status codes.
const (
	statusSuccess            status = iota // Success
	statusError                            // Failed
	statusBusy                             // Busy
	statusInvalidHandle                    // Invalid handle
	statusInvalidParameter                 // Invalid parameter
	statusInvalidRequest                   // Invalid request
	statusControllerNotFound               // Controller cannot be found
	statusInvalidController                // Invalid controller interface
	statusNotSupported                     // Configuration is not supported
	statusRetry                            // Enumeration get configuration retry
	statusTransferStall                    // Transfer stalled
	statusTransferFailed                   // Transfer failed
	statusAllocFail                        // Allocation failed
	statusLackSwapBuffer                   // Insufficient swap buffer for KHCI
	statusTransferCancel                   // The transfer cancelled
	statusBandwidthFail                    // Allocate bandwidth failed
	statusMSDStatusFail                    // For MSD, the CSW status means fail
	statusEHCIAttached                     // EHCI attached
	statusEHCIDetached                     // EHCI detached
	// The amount of data returned by the endpoint exceeded either the size of the
	// maximum data packet allowed from the endpoint or the remaining buffer size.
	statusDataOverRun

	modeIdle   mode = iota // USB core idle (unallocated)
	modeDevice             // USB device mode
	modeHost               // USB host mode
)

var (
	portInstance [ConfigPortCount]port
)

func init() {
	for i := range portInstance {
		portInstance[i].mode = modeIdle
	}
}

func initPort(port uint8, mode mode) (*port, status) {
	if port >= ConfigPortCount || int(port) >= len(portInstance) {
		return nil, statusInvalidController
	}
	if modeIdle != portInstance[port].mode {
		return nil, statusBusy
	}
	portInstance[port].mode = mode
	switch mode {
	case modeDevice:
		if s := portInstance[port].device.init(port); !s.OK() {
			return nil, s
		}
	case modeHost:
		if s := portInstance[port].host.init(port); !s.OK() {
			return nil, s
		}
	}
	return &portInstance[port], statusSuccess
}

// OK returns true if and only if the receiver s is equal to statusSuccess.
func (s status) OK() bool { return statusSuccess == s }

// Error returns a simple descriptive error string of the receiver s.
func (s status) Error() string {
	switch s {
	case statusSuccess:
		return ""
	case statusError:
		return "failed"
	case statusBusy:
		return "busy"
	case statusInvalidHandle:
		return "invalid handle"
	case statusInvalidParameter:
		return "invalid parameter"
	case statusInvalidRequest:
		return "invalid request"
	case statusControllerNotFound:
		return "controller not found"
	case statusInvalidController:
		return "invalid controller interface"
	case statusNotSupported:
		return "configuration not supported"
	case statusRetry:
		return "retry enumeration"
	case statusTransferStall:
		return "transfer stalled"
	case statusTransferFailed:
		return "transfer failed"
	case statusAllocFail:
		return "allocation failed"
	case statusLackSwapBuffer:
		return "insufficient swap buffer"
	case statusTransferCancel:
		return "transfer cancelled"
	case statusBandwidthFail:
		return "bandwidth allocation failed"
	case statusMSDStatusFail:
		return "mass-storage device failed"
	case statusEHCIAttached:
		return "host attached"
	case statusEHCIDetached:
		return "host detached"
	case statusDataOverRun:
		return "data overrun"
	default:
		return "unknown error"
	}
}
