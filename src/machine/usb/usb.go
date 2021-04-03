package usb

type (
	// status is the common error code used internally for package operations.
	status int8
	// mode defines the operating mode of a USB port.
	mode int8

	// port represents a physical USB port, which may be configured as either a
	// device or as a host.
	port struct {
		mode   mode
		device device
		host   host
	}
)

// Constants for unexported types shared across entire package.
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
	statusDataOverRun                      // Endpoint data (Rx) exceeds max size

	modeIdle   mode = iota // USB core idle (unallocated)
	modeDevice             // USB device mode
	modeHost               // USB host mode
)

var (
	// portInstance holds instances for all available ports on the platform.
	portInstance [ConfigPortCount]port
)

func init() {
	// ensure all ports are in idle state by default
	for i := range portInstance {
		portInstance[i].mode = modeIdle
	}
}

// initPort configures the mode for a given USB port and initializes the
// hardware's port controller. If the port is invalid or not idle (it has
// already been configured), it returns nil and a status code.
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

// deinit disables the receiver USB port, changing its mode to idle, freeing it
// for reuse or reconfiguration.
func (p *port) deinit() status {
	switch p.mode {
	case modeDevice:
		return p.device.deinit()
	case modeHost:
		return p.host.deinit()
	default:
		return statusInvalidController
	}
}

// initCDCACM applies a CDC-ACM configuration to the receiver port p and then
// returns the configured deviceClassDriver and deviceClass that were assigned
// to the receiver.
//
// The given callback function will be called for any USB device-level event
// notifications received, which allows an upper-layer CDC-ACM driver (such as
// a UART interface implementation) the opportunity to handle device events.
func (p *port) initCDCACM(callback deviceEventFunc) (*deviceCDCACM, *deviceClass) {

	// verify a valid port was provided
	if nil == p || nil == p.device.controller || p.mode != modeDevice {
		return nil, nil
	}

	// get a reference to each of the class interfaces
	comm := &deviceCDCACMConfigInstance[p.device.port][0].info.interfaceList[0]
	data := &deviceCDCACMConfigInstance[p.device.port][0].info.interfaceList[1]

	// configDeviceCDCACM must be defined per package API. these settings will be
	// platform-specific, and will probably be implemented in a build tag-
	// constrained source file. the length of this array corresponds to the number
	// of USB CDC-ACM ports that are being created, and the index of each element
	// corresponds to the physical USB port (core index).

	// CDC-ACM Communication/control interface
	comm.interfaceNumber =
		configDeviceCDCACM[p.device.port].commInterfaceIndex

	comm.deviceInterface[0].endpoint[0].address =
		configDeviceCDCACM[p.device.port].commInterruptInEndpoint |
			specDescriptorEndpointAddressDirectionIn

	comm.deviceInterface[0].endpoint[0].maxPacketSize =
		configDeviceCDCACM[p.device.port].commInterruptInPacketSize

	comm.deviceInterface[0].endpoint[0].interval =
		configDeviceCDCACM[p.device.port].commInterruptInInterval

	// CDC-ACM Data interface
	data.interfaceNumber =
		configDeviceCDCACM[p.device.port].dataInterfaceIndex

	data.deviceInterface[0].endpoint[0].address =
		configDeviceCDCACM[p.device.port].dataBulkInEndpoint |
			specDescriptorEndpointAddressDirectionIn

	data.deviceInterface[0].endpoint[0].maxPacketSize =
		configDeviceCDCACM[p.device.port].dataBulkInPacketSize

	data.deviceInterface[0].endpoint[1].address =
		configDeviceCDCACM[p.device.port].dataBulkOutEndpoint |
			specDescriptorEndpointAddressDirectionOut

	data.deviceInterface[0].endpoint[1].maxPacketSize =
		configDeviceCDCACM[p.device.port].dataBulkOutPacketSize

	// assign our configured CDC-ACM class to the receiver's device and call its
	// class initialization routine(s).
	cls := p.device.initClass(deviceCDCACMConfigInstance[p.device.port], callback)
	acm := cls.config[0].driver.(*deviceCDCACM)

	return acm, cls
}

// OK returns true if and only if the receiver s is equal to statusSuccess.
//go:inline
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
