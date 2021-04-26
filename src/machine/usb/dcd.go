package usb

// Implementation of 32-bit target-agnostic USB device controller driver (dcd).

import (
	"math/bits"
	"unsafe"
)

func init() {
	if unsafe.Sizeof(uintptr(0)) > 4 {
		panic("USB device controller is only supported on 32-bit systems")
	}
}

// dcdCount defines the number of USB cores to configure for device mode. It is
// computed as the sum of all declared device configuration descriptors.
const dcdCount = descCDCACMCount // + ...

// dcdInstance provides statically-allocated instances of each USB device
// controller configured on this platform.
var dcdInstance [dcdCount]dcd

// dhwInstance provides statically-allocated instances of each USB hardware
// abstraction for ports configured as device on this platform.
var dhwInstance [dcdCount]dhw

// dcd implements a generic USB device controller driver (dcd) for 32-bit ARM
// targets.
type dcd struct {
	core *core // Parent USB core this instance is attached to
	port int   // USB port index
	cc   class // USB device class
	id   int   // USB device controller index
	hw   *dhw  // USB hardware

	stat *dcdEndpoint // endpoint 0 Rx ("out" direction)
	ctrl *dcdEndpoint // endpoint 0 Tx ("in" direction)

	controlMask  uint32
	endpointMask uint32
	setup        dcdSetup
	controlReply [8]uint8
}

// initDCD initializes and assigns a free device controller instance to the
// given USB port. Returns the initialized device controller or nil if no free
// device controller instances remain.
func initDCD(port int, class class) (*dcd, status) {
	if 0 == dcdCount {
		return nil, statusInvalid // Must have defined device controllers
	}
	switch class.id {
	case classDeviceCDCACM:
		if 0 == class.config || class.config > descCDCACMCount {
			return nil, statusInvalid // Must have defined descriptors
		}
	default:
	}
	// Return the first instance whose assigned core is currently nil.
	for i := range dcdInstance {
		if nil == dcdInstance[i].core {
			// Initialize device controller.
			dcdInstance[i].core = &coreInstance[port]
			dcdInstance[i].port = port
			dcdInstance[i].cc = class
			dcdInstance[i].id = i
			dcdInstance[i].hw = allocDHW(&dcdInstance[i])
			return &dcdInstance[i], statusOK
		}
	}
	return nil, statusBusy // No free device controller instances available.
}

// class returns the receiver's current device class configuration.
func (d *dcd) class() class { return d.cc }

// init configures the USB port for device mode operation by initializing all
// endpoint and transfer descriptor data structures, initializing core registers
// and interrupts, resetting the USB PHY, and enabling power on the bust.
func (d *dcd) init() status {

	// Initialize endpoint 0 (control/status)
	d.stat = d.endpointQueueHead(rxEndpoint(0))
	d.ctrl = d.endpointQueueHead(txEndpoint(0))
	d.stat.config = (descEndptMaxPktSize << 16) | (1 << 15)
	d.ctrl.config = (descEndptMaxPktSize << 16)

	// Initialize target hardware
	return d.hw.init()
}

// enable causes the USB core to enter (or exit) the normal run state and
// enables (or disables) all interrupts associated with the receivers USB port.
func (d *dcd) enable(enable bool) {
	if enable {
		d.hw.interruptsEnableUSB(true)
		d.hw.enable(true)
	} else {
		d.hw.enable(false)
		d.hw.interruptsEnableUSB(false)
	}
}

// dcdEndpointSize defines the size (bytes) of a structure containing a USB
// standard endpoint.
const dcdEndpointSize = 64 // bytes

// dcdEndpoint defines a USB standard endpoint, used as the general channel of
// communication between host and device.
type dcdEndpoint struct {
	config   uint32
	current  *dcdTransfer
	transfer dcdTransfer
	setup    dcdSetup
	// Endpoints are 48-byte data structures. The remaining data extends this to
	// 64-byte, and also makes it simpler for implementations to align endpoints
	// allocated contiguously on 64-byte boundaries.
	first *dcdTransfer
	last  *dcdTransfer
	// After some discussion, perhaps the simplest change to support 64-bit (or
	// future TinyGo versions that don't use 8 bytes to refer to a function) is to
	// allocate a separate buffer of callbacks. The device controller would assign
	// callbacks to unused elements in that buffer, and only the index of that
	// callback would be stored here in the descriptor.
	callback func(transfer *dcdTransfer)
}

// endpointQueueHead returns the queue head for the given endpoint address,
// encoded as direction D and endpoint number N with the 8-bit mask DxxxNNNN.
//go:inline
func (d *dcd) endpointQueueHead(endpoint uint8) *dcdEndpoint {
	// endpoint queue head is device class-specific
	switch d.cc.id {
	case classDeviceCDCACM:
		return &descCDCACM[d.cc.config-1].qh[endpointIndex(endpoint)]
	default:
		return nil
	}
}

// endpointConfigure configures the given bulk data endpoint for transfer.
func (d *dcd) endpointConfigure(
	ep *dcdEndpoint, packetSize uint16, zlp bool, callback func(transfer *dcdTransfer)) {

	ep.config = uint32(packetSize) << 16
	if !zlp {
		ep.config |= 1 << 29
	}
	ep.current = nil
	ep.transfer.next = dcdTransferEOL
	ep.transfer.token = 0
	for i := range ep.transfer.pointer {
		ep.transfer.pointer[i] = 0
	}
	ep.transfer.param = 0
	ep.setup.bmRequestType = 0
	ep.setup.bRequest = 0
	ep.setup.wValue = 0
	ep.setup.wIndex = 0
	ep.setup.wLength = 0
	ep.first = nil
	ep.last = nil
	ep.callback = callback
}

// endpointComplete handles transfer completion of a data endpoint.
func (d *dcd) endpointComplete(endpoint uint8) {
	ep := d.endpointQueueHead(endpoint)
	if nil == ep.first {
		return
	}
	count := 0
	first := ep.first
	for t, eol := first, false; !eol; t, eol = t.nextTransfer() {
		if eol {
			// reached end of list, new list empty
			ep.first = nil
			ep.last = nil
		} else {
			if 0 != t.token&(1<<7) {
				// active transfer, new list begins here
				ep.first = t
				break
			} else {
				count += 1
			}
		}
	}
	// invoke all callbacks
	for i := 0; i < count; i++ {
		next := first.next
		ep.callback(first)
		first = next
	}
}

// endpointConfigureRx configures the given bulk data receive (Rx, OUT) endpoint
// for transfer.
func (d *dcd) endpointConfigureRx(
	endpoint uint8, packetSize uint16, zlp bool, callback func(transfer *dcdTransfer)) {

	// Configure based on our device class configuration
	switch d.cc.id {

	// CDC-ACM (single)
	case classDeviceCDCACM:
		if endpoint < descCDCACMEndpointStatus ||
			endpoint > descCDCACMEndpointCount {
			return
		}
		ep := d.endpointQueueHead(rxEndpoint(endpoint))
		d.endpointConfigure(ep, packetSize, zlp, callback)
		if nil != callback {
			d.endpointMask |= (uint32(1) << endpoint) << descEndptConfigAttrRxPos
		}

	default:
		// Unhandled device class
	}
}

// endpointConfigureTx configures the given bulk data transmit (Tx, IN) endpoint
// for transfer.
func (d *dcd) endpointConfigureTx(
	endpoint uint8, packetSize uint16, zlp bool, callback func(transfer *dcdTransfer)) {

	// Configure based on our device class configuration
	switch d.cc.id {

	// CDC-ACM (single)
	case classDeviceCDCACM:
		if endpoint < descCDCACMEndpointStatus ||
			endpoint > descCDCACMEndpointCount {
			return
		}
		ep := d.endpointQueueHead(txEndpoint(endpoint))
		d.endpointConfigure(ep, packetSize, zlp, callback)
		if nil != callback {
			d.endpointMask |= (uint32(1) << endpoint) << descEndptConfigAttrTxPos
		}

	default:
		// Unhandled device class
	}
}

// endpointReceive schedules a receive (Rx, OUT) transfer on the given endpoint.
func (d *dcd) endpointReceive(endpoint uint8, transfer *dcdTransfer) {

	// Configure based on our device class configuration
	switch d.cc.id {

	// CDC-ACM (single)
	case classDeviceCDCACM:
		if endpoint < descCDCACMEndpointStatus ||
			endpoint > descCDCACMEndpointCount {
			return
		}
		ep := d.endpointQueueHead(rxEndpoint(endpoint))
		em := (uint32(1) << endpoint) << descEndptConfigAttrRxPos
		d.transferSchedule(ep, em, transfer)

	default:
		// Unhandled device class
	}
}

// endpointTransmit schedules a transmit (Tx, IN) transfer on the given
// endpoint.
func (d *dcd) endpointTransmit(endpoint uint8, transfer *dcdTransfer) {

	// Configure based on our device class configuration
	switch d.cc.id {

	// CDC-ACM (single)
	case classDeviceCDCACM:
		if endpoint < descCDCACMEndpointStatus ||
			endpoint > descCDCACMEndpointCount {
			return
		}
		ep := d.endpointQueueHead(txEndpoint(endpoint))
		em := (uint32(1) << endpoint) << descEndptConfigAttrTxPos
		d.transferSchedule(ep, em, transfer)

	default:
		// Unhandled device class
	}
}

// dcdTransferSize defines the size (bytes) of a USB standard transfer packet.
const dcdTransferSize = 32 // bytes

// dcdTransfer describes the size and location of data to be transferred to or
// from a USB endpoint.
type dcdTransfer struct {
	next    *dcdTransfer
	token   uint32
	pointer [5]uintptr
	param   uint32
}

// dcdTransferEOL is a sentinel value used to indicate the final node in a
// linked list of transfer descriptors.
var dcdTransferEOL = (*dcdTransfer)(unsafe.Pointer(uintptr(1)))

// nextTransfer returns the next transfer descriptor pointed to by the receiver
// transfer descriptor, and whether or not that next descriptor is the final
// descriptor in the list.
func (t dcdTransfer) nextTransfer() (*dcdTransfer, bool) {
	return t.next, 1 == uintptr(unsafe.Pointer(t.next))
}

// transferControl returns the data and ackowledgement transfer descriptors for
// the control endpoint (i.e., endpoint 0).
//go:inline
func (d *dcd) transferControl() (dat, ack *dcdTransfer) {
	// control endpoint is device class-specific
	switch d.cc.id {
	case classDeviceCDCACM:
		return descCDCACM[d.cc.config-1].cd, descCDCACM[d.cc.config-1].ad
	default:
		return nil, nil
	}
}

func (d *dcd) transferPrepare(
	transfer *dcdTransfer, data *uint8, size uint16, param uint32) {
	transfer.next = dcdTransferEOL
	transfer.token = (uint32(size) << 16) | (1 << 7)
	addr := uintptr(unsafe.Pointer(data))
	for i := range transfer.pointer {
		transfer.pointer[i] = addr + uintptr(i)*4096
	}
	transfer.param = param
}

func (d *dcd) transferSchedule(
	endpoint *dcdEndpoint, mask uint32, transfer *dcdTransfer) {

	if nil != endpoint.callback {
		transfer.token |= 1 << 15
	}
	// ivm := arm.DisableInterrupts()
	d.hw.interruptsEnableUSB(false)
	last := endpoint.last
	if nil != last {
		// last.next = transfer
		// if d.hw.bus.ENDPTPRIME.HasBits(mask) {
		// 	goto endTransfer
		// }
		// start := cycleCount()
		// estat := uint32(0)
		// for !d.hw.bus.USBCMD.HasBits(nxp.USB_USBCMD_ATDTW) &&
		// 	(cycleCount()-start < 2400) {
		// 	d.hw.bus.USBCMD.SetBits(nxp.USB_USBCMD_ATDTW)
		// 	estat = d.hw.bus.ENDPTSTAT.Get()
		// }
		// if 0 != estat&mask {
		// goto endTransfer
		// }
	}
	endpoint.transfer.next = transfer
	endpoint.transfer.token = 0
	//d.hw.bus.ENDPTPRIME.SetBits(mask)
	d.hw.endpointPrime(mask, transfer)
	endpoint.first = transfer
	// endTransfer:
	endpoint.last = transfer
	// arm.EnableInterrupts(ivm)
	d.hw.interruptsEnableUSB(true)
}

// dcdSetupSize defines the size (bytes) of a USB standard setup packet.
const dcdSetupSize = 8 // bytes

// dcdSetup contains the USB standard setup packet used to configure a device.
type dcdSetup struct {
	bmRequestType uint8
	bRequest      uint8
	wValue        uint16
	wIndex        uint16
	wLength       uint16
}

// pack returns the receiver setup packet encoded as uint64.
func (s dcdSetup) pack() uint64 {
	return ((uint64(s.bmRequestType) & 0xFF) << 0) |
		((uint64(s.bRequest) & 0xFF) << 8) |
		((uint64(s.wValue) & 0xFFFF) << 16) |
		((uint64(s.wIndex) & 0xFFFF) << 32) |
		((uint64(s.wLength) & 0xFFFF) << 48)
}

// dcdEvent is used to describe virtual interrupts on the USB bus to a device
// controller.
//
// Since the device controller software is intended for use with multiple TinyGo
// targets, all of which may not have exactly the same USB bus interrupts, a
// "virtual interrupt" is defined that is common to all targets. The target's
// hardware implementation (type dhw) is responsible for translating real system
// interrupts it receives into the appropriate virtual interrupt code, defined
// below, and notifying the device controller via method (*dcd).event(dcdEvent).
type dcdEvent struct {
	id    uint8
	setup dcdSetup
	mask  uint32
}

// Enumerated constants for all possible USB device controller interrupt codes.
const (
	dcdEventInvalid          uint8 = iota // Invalid interrupt
	dcdEventStatusReset                   // USB reset received
	dcdEventStatusRun                     // USB controller entered run state
	dcdEventStatusSuspend                 // USB suspend received
	dcdEventStatusError                   // USB error condition detected on bus
	dcdEventControlSetup                  // USB setup received
	dcdEventPeripheralReady               // USB PHY powered and ready to _go_
	dcdEventTransactComplete              // USB transaction complete
	dcdEventTimer                         // USB (system) timer
)

func (d *dcd) event(ev dcdEvent) {

	switch ev.id {
	case dcdEventInvalid:
	case dcdEventStatusReset:
		d.endpointMask = 0

	case dcdEventPeripheralReady:
		// Configure and enable control endpoint 0
		d.hw.endpointEnable(0, true, 0)

	case dcdEventStatusRun:
	case dcdEventStatusSuspend:
	case dcdEventStatusError:
	case dcdEventControlSetup:
		// On control endpoint 0 setup events, the ev.setup field will be defined
		switch d.controlSetup(ev.setup) {
		case dcdStageSetup:
		case dcdStageData:
		case dcdStageStatus:
			d.hw.controlStatus()
		case dcdStageStall:
			d.hw.controlStall()
		}

	case dcdEventTransactComplete:
		// On transaction completion events, the event mask identifies which
		// endpoints signalled completion. The upper 16 bits represent Tx (IN), and
		// the lower 16 are the Rx (OUT) endpoints.
		if 0 != ev.mask&d.controlMask {
			d.controlComplete(ev.mask)
		}
		ev.mask &= d.endpointMask
		if 0 != ev.mask {
			tx := ev.mask >> 16
			for 0 != tx {
				num := uint8(bits.TrailingZeros32(tx))
				d.endpointComplete(txEndpoint(num))
				tx &^= 1 << num
			}
			rx := ev.mask & 0xFFFF
			for 0 != rx {
				num := uint8(bits.TrailingZeros32(rx))
				d.endpointComplete(rxEndpoint(num))
				rx &^= 1 << num
			}
		}

	case dcdEventTimer:

	default:
	}
}

// dcdStage represents the USB transaction stage of a control request.
type dcdStage uint8

// Enumerated constants for all possible USB transaction stages.
const (
	dcdStageSetup  dcdStage = iota // Indicates no stage transition required
	dcdStageData                   // IN/OUT data transfer
	dcdStageStatus                 // Setup request complete
	dcdStageStall                  // Unhandled or invalid request
)

// controlSetup handles setup messages on control endpoint 0.
func (d *dcd) controlSetup(sup dcdSetup) dcdStage {

	// println(strconv.FormatUint(setup.pack(), 16))

	// Reset endpoint 0 notify mask
	d.controlMask = 0

	// First, switch on the type of request (standard, class, or vendor)
	switch sup.bmRequestType & descRequestTypeTypeMsk {

	// === STANDARD REQUEST ===
	case descRequestTypeTypeStandard:

		// Switch on the recepient and direction of the request
		switch sup.bmRequestType &
			(descRequestTypeRecipientMsk | descRequestTypeDirMsk) {

		// --- DEVICE Rx (OUT) ---
		case descRequestTypeRecipientDevice | descRequestTypeDirOut:

			// Identify which request was received
			switch sup.bRequest {

			// SET ADDRESS (0x05):
			case descRequestStandardSetAddress:
				d.hw.controlDeviceAddress(sup.wValue)
				d.controlReceive(uintptr(0), 0, false)
				return dcdStageSetup

			// SET CONFIGURATION (0x09):
			case descRequestStandardSetConfiguration:
				d.cc.config = int(sup.wValue)
				if 0 == d.cc.config || d.cc.config > dcdCount {
					// Use default if invalid index received
					d.cc.config = 1
				}

				// Respond based on our device class configuration
				switch d.cc.id {

				// CDC-ACM (single)
				case classDeviceCDCACM:
					d.uartConfigure()
					d.controlReceive(uintptr(0), 0, false)

				default:
					// Unhandled device class
				}
				return dcdStageSetup

			default:
				// Unhandled request
			}

		// --- DEVICE Tx (IN) ---
		case descRequestTypeRecipientDevice | descRequestTypeDirIn:

			// Identify which request was received
			switch sup.bRequest {

			// GET STATUS (0x00):
			case descRequestStandardGetStatus:
				d.controlReply[0] = 0
				d.controlReply[1] = 0
				d.controlTransmit(
					uintptr(unsafe.Pointer(&d.controlReply[0])), 2, false)
				return dcdStageSetup

			// GET DESCRIPTOR (0x06):
			case descRequestStandardGetDescriptor:
				d.controlDescriptor(sup)
				return dcdStageSetup

			// GET CONFIGURATION (0x08):
			case descRequestStandardGetConfiguration:
				d.controlReply[0] = uint8(d.cc.config)
				d.controlTransmit(
					uintptr(unsafe.Pointer(&d.controlReply[0])), 1, false)
				return dcdStageSetup

			default:
				// Unhandled request
			}

		// --- INTERFACE Tx (IN) ---
		case descRequestTypeRecipientInterface | descRequestTypeDirIn:

			// Identify which request was received
			switch sup.bRequest {

			// GET DESCRIPTOR (0x06):
			case descRequestStandardGetDescriptor:
				d.controlDescriptor(sup)
				return dcdStageSetup

			default:
				// Unhandled request
			}

		// --- ENDPOINT Rx (OUT) ---
		case descRequestTypeRecipientEndpoint | descRequestTypeDirOut:

			// Identify which request was received
			switch sup.bRequest {

			// CLEAR FEATURE (0x01):
			case descRequestStandardClearFeature:
				// TODO

			// SET FEATURE (0x03):
			case descRequestStandardSetFeature:
				// TODO

			default:
				// Unhandled request
			}

		// --- ENDPOINT Tx (IN) ---
		case descRequestTypeRecipientEndpoint | descRequestTypeDirIn:

			// Identify which request was received
			switch sup.bRequest {

			// GET STATUS (0x00):
			case descRequestStandardGetStatus:
				status := d.hw.endpointStatus(uint8(sup.wIndex))
				d.controlReply[0] = uint8(status)
				d.controlReply[1] = uint8(status >> 8)
				d.controlTransmit(
					uintptr(unsafe.Pointer(&d.controlReply[0])), 2, false)
				return dcdStageSetup

			default:
				// Unhandled request
			}

		default:
			// Unhandled request recepient or direction
		}

	// === CLASS REQUEST ===
	case descRequestTypeTypeClass:

		// Switch on the recepient and direction of the request
		switch sup.bmRequestType &
			(descRequestTypeRecipientMsk | descRequestTypeDirMsk) {

		// --- INTERFACE Rx (OUT) ---
		case descRequestTypeRecipientInterface | descRequestTypeDirOut:

			// Identify which request was received
			switch sup.bRequest {

			// CDC | SET LINE CODING (0x20):
			case descCDCRequestSetLineCoding:

				// Respond based on our device class configuration
				switch d.cc.id {

				// CDC-ACM (single)
				case classDeviceCDCACM:
					// line coding must contain exactly 7 bytes
					if descCDCACMCodingSize == sup.wLength {
						d.setup = sup
						d.controlReceive(
							uintptr(unsafe.Pointer(&descCDCACM[d.cc.config-1].cx[0])),
							descCDCACMCodingSize, true)
						return dcdStageSetup
					}

				default:
					// Unhandled device class
				}

			// CDC | SET CONTROL LINE STATE (0x22):
			case descCDCRequestSetControlLineState:

				// Respond based on our device class configuration
				switch d.cc.id {

				// CDC-ACM (single)
				case classDeviceCDCACM:

					// Determine interface destination of the notification
					switch sup.wIndex {

					// Control/status interface:
					case descCDCACMInterfaceCtrl:
						// Notify PHY to handle triggers like special baud rates, which
						// signal to reboot into bootloader or begin receiving OTA updates
						d.hw.controlLineState(descCDCACM[d.cc.config-1].lineCoding,
							0 != sup.wValue&0x01, 0 != sup.wValue&0x02)
						d.controlReceive(uintptr(0), 0, false)
						return dcdStageSetup

					default:
						// Unhandled device interface
					}

				default:
					// Unhandled device class
				}

			// CDC | SEND BREAK (0x23):
			case descCDCRequestSendBreak:

				// Respond based on our device class configuration
				switch d.cc.id {

				// CDC-ACM (single)
				case classDeviceCDCACM:
					d.controlReceive(uintptr(0), 0, false)
					return dcdStageSetup

				default:
					// Unhandled device class
				}

			default:
				// Unhandled request
			}

		default:
			// Unhandled request recepient or direction
		}

	case descRequestTypeTypeVendor:
	default:
		// Unhandled request type
	}

	// All successful requests return early. If we reach this point, the request
	// was invalid or unhandled. Stall the endpoint.
	return dcdStageStall
}

// controlComplete handles the setup completion of control endpoint 0.
func (d *dcd) controlComplete(status uint32) {

	// Reset endpoint 0 notify mask
	d.controlMask = 0

	// First, switch on the type of request (standard, class, or vendor)
	switch d.setup.bmRequestType & descRequestTypeTypeMsk {

	// === CLASS REQUEST ===
	case descRequestTypeTypeClass:

		// Switch on the recepient and direction of the request
		switch d.setup.bmRequestType &
			(descRequestTypeRecipientMsk | descRequestTypeDirMsk) {

		// --- INTERFACE Rx (OUT) ---
		case descRequestTypeRecipientInterface | descRequestTypeDirOut:

			// Identify which request was received
			switch d.setup.bRequest {

			// CDC | SET LINE CODING (0x20):
			case descCDCRequestSetLineCoding:

				// Respond based on our device class configuration
				switch d.cc.id {

				// CDC-ACM (single)
				case classDeviceCDCACM:
					acm := &descCDCACM[d.cc.config-1]

					// Determine interface destination of the notification
					switch d.setup.wIndex {

					// Control/status interface:
					case descCDCACMInterfaceCtrl:
						acm.lineCoding.baud = packU32(acm.cx[:])
						acm.lineCoding.stopBits = acm.cx[4]
						if 0 == acm.lineCoding.stopBits {
							acm.lineCoding.stopBits = 1
						}
						acm.lineCoding.parity = acm.cx[5]
						acm.lineCoding.numBits = acm.cx[6]
						// Notify PHY to handle triggers like special baud rates, which
						// signal to reboot into bootloader or begin receiving OTA updates
						d.hw.controlLineCoding(acm.lineCoding)

					default:
						// Unhandled device interface
					}

				default:
					// Unhandled device class
				}

			default:
				// Unhandled request
			}

		default:
			// Unhandled recepient or direction
		}

	default:
		// Unhandled request type
	}
}

func (d *dcd) controlDescriptor(sup dcdSetup) {

	// Respond based on our device class configuration
	switch d.cc.id {

	// CDC-ACM (single)
	case classDeviceCDCACM:
		acm := &descCDCACM[d.cc.config-1]
		dxn := uint8(0)

		// Determine the type of descriptor being requested
		switch sup.wValue >> 8 {

		// Device descriptor
		case descTypeDevice:
			dxn = descLengthDevice
			_ = copy(acm.dx[:], acm.device[:dxn])

		// Configuration descriptor
		case descTypeConfigure:
			dxn = uint8(descCDCACMConfigSize)
			_ = copy(acm.dx[:], acm.config[:dxn])

		// String descriptor
		case descTypeString:
			if 0 == len(acm.locale) {
				break // No string descriptors defined!
			}
			var sd []uint8
			if 0 == uint8(sup.wValue) {

				// setup.wIndex contains an arbitrary index referring to a collection of
				// strings in some given language. This case (setup.wValue = [0x03]00)
				// is a string request from the host to determine what that language is.
				//
				// In subsequent string requests, the host will populate setup.wIndex
				// with the language code we return here in this string descriptor.
				//
				// This way all strings returned to the host are in the same language,
				// whatever language that may be.
				code := int(sup.wIndex)
				if code >= len(acm.locale) {
					code = 0
				}
				sd = acm.locale[code].descriptor[sup.wValue&0xFF][:]

			} else {

				// setup.wIndex now contains a language code, which we specified in a
				// previous request (above: setup.wValue = [0x03]00). We need to locate
				// the set of strings whose language matches the language code given in
				// this new setup.wIndex.
				for code := range acm.locale {
					if sup.wIndex == acm.locale[code].language {
						// Found language, check if string descriptor at given index exists
						if int(sup.wValue&0xFF) < len(acm.locale[code].descriptor) {

							// Found language with a string defined at the requested index.
							//
							// TODO: Add API methods to device controller that allows the user
							//       to provide these strings at/before driver initialization.
							//
							// For now, we just always use the descCommon* strings.
							var s string
							switch uint8(sup.wValue) {
							case 1:
								s = descCommonManufacturer
							case 2:
								s = descCommonProduct
							case 3:
								s = descCommonSerialNumber
							}

							// Construct a string descriptor dynamically to be transmitted on
							// the serial bus.
							sd = acm.locale[code].descriptor[int(sup.wValue&0xFF)][:]
							// String descriptor format is 2-byte header + 2-bytes per rune
							sd[0] = uint8(2 + 2*len(s)) // header[0] = descriptor length
							sd[1] = descTypeString      // header[1] = descriptor type
							// Copy UTF-8 string into string descriptor as UTF-16
							for n, c := range s {
								if 2+2*n >= len(sd) {
									break
								}
								sd[2+2*n] = uint8(c)
								sd[3+2*n] = 0
							}
							break // end search for matching language code
						}
					}
				}
			}
			// Copy string descriptor into descriptor transmit buffer
			if nil != sd && len(sd) >= 0 {
				dxn = sd[0]
				_ = copy(acm.dx[:], sd[:dxn])
			}

		// Device qualification descriptor
		case descTypeQualification:
			dxn = descLengthQualification
			_ = copy(acm.dx[:], acm.qualif[:dxn])

		// Alternate configuration descriptor
		case descTypeOtherSpeedConfiguration:
			// TODO

		default:
			// Unhandled descriptor type
		}

		if dxn > 0 {
			if dxn > uint8(sup.wLength) {
				dxn = uint8(sup.wLength)
			}
			d.hw.flushCache(
				uintptr(unsafe.Pointer(&acm.dx[0])), uintptr(dxn))
			d.controlTransmit(
				uintptr(unsafe.Pointer(&acm.dx[0])), uint32(dxn), false)
		}

	default:
		// Unhandled device class
	}

}

// controlReceive receives (Rx, OUT) data on control endpoint 0.
func (d *dcd) controlReceive(
	data uintptr, size uint32, notify bool) {
	const (
		rm = uint32(1 << descEndptConfigAttrRxPos)
		tm = uint32(1 << descEndptConfigAttrTxPos)
	)
	cd, ad := d.transferControl()
	if size > 0 {
		cd.next = dcdTransferEOL
		cd.token = (size << 16) | (1 << 7)
		for i := range cd.pointer {
			cd.pointer[i] = data + uintptr(i)*4096
		}
		// linked list is empty
		qr := d.endpointQueueHead(rxEndpoint(0))
		qr.transfer.next = cd
		qr.transfer.token = 0
		//d.hw.bus.ENDPTPRIME.SetBits(rm)
		d.hw.endpointPrime(rm, cd)
		for 0 != d.hw.endpointPrimed() {
		} // wait for endpoint finish priming
	}
	ad.next = dcdTransferEOL
	ad.token = 1 << 7
	if notify {
		ad.token |= 1 << 15
	}
	for i := range ad.pointer {
		ad.pointer[i] = 0
	}
	qt := d.endpointQueueHead(txEndpoint(0))
	qt.transfer.next = ad
	qt.transfer.token = 0
	d.hw.endpointUnprime(rm | tm)
	//d.hw.bus.ENDPTPRIME.SetBits(tm)
	d.hw.endpointPrime(tm, ad)
	if notify {
		d.controlMask = tm
	}
	for 0 != d.hw.endpointPrimed() {
	} // wait for endpoint finish priming
}

// controlTransmit transmits (Tx, IN) data on control endpoint 0.
func (d *dcd) controlTransmit(
	data uintptr, size uint32, notify bool) {
	const (
		rm = uint32(1 << descEndptConfigAttrRxPos)
		tm = uint32(1 << descEndptConfigAttrTxPos)
	)
	cd, ad := d.transferControl()
	if size > 0 {
		cd.next = dcdTransferEOL
		cd.token = (size << 16) | (1 << 7)
		for i := range cd.pointer {
			cd.pointer[i] = data + uintptr(i)*4096
		}
		// linked list is empty
		qt := d.endpointQueueHead(txEndpoint(0))
		qt.transfer.next = cd
		qt.transfer.token = 0
		//d.hw.bus.ENDPTPRIME.SetBits(tm)
		d.hw.endpointPrime(tm, cd)
		for 0 != d.hw.endpointPrimed() {
		} // wait for endpoint finish priming
	}
	ad.next = dcdTransferEOL
	ad.token = 1 << 7
	if notify {
		ad.token |= 1 << 15
	}
	for i := range ad.pointer {
		ad.pointer[i] = 0
	}
	qr := d.endpointQueueHead(rxEndpoint(0))
	qr.transfer.next = ad
	qr.transfer.token = 0
	d.hw.endpointUnprime(rm | tm)
	//d.hw.bus.ENDPTPRIME.SetBits(rm)
	d.hw.endpointPrime(rm, ad)
	if notify {
		d.controlMask = rm
	}
	for 0 != d.hw.endpointPrimed() {
	} // wait for endpoint finish priming
}

func (d *dcd) uartConfigure() {
	acm := &descCDCACM[d.cc.config-1]
	switch d.hw.controlSpeed() {
	case descDeviceSpeedHigh:
		acm.rxSize = descCDCACMDataRxHSPacketSize
		acm.txSize = descCDCACMDataTxHSPacketSize
	default:
		acm.rxSize = descCDCACMDataRxFSPacketSize
		acm.txSize = descCDCACMDataTxFSPacketSize
	}
	acm.txHead = 0
	acm.txFree = 0
	acm.rxHead = 0
	acm.rxTail = 0
	acm.rxFree = 0

	d.hw.endpointEnable(descCDCACMEndpointStatus,
		false, descCDCACMConfigAttrStatus)
	d.hw.endpointEnable(descCDCACMEndpointDataRx,
		false, descCDCACMConfigAttrDataRx)
	d.hw.endpointEnable(descCDCACMEndpointDataTx,
		false, descCDCACMConfigAttrDataTx)

	d.endpointConfigureTx(descCDCACMEndpointStatus,
		acm.cxSize, false, nil)
	d.endpointConfigureRx(descCDCACMEndpointDataRx,
		acm.rxSize, false, d.uartNotify)
	d.endpointConfigureTx(descCDCACMEndpointDataTx,
		acm.txSize, true, nil)
	for i := range acm.rd {
		d.uartReceive(uint8(i))
	}
	// d.hw.timerConfigure(0, descCDCACMTxSyncUs, d.uartSync)
}

func (d *dcd) uartReceive(endpoint uint8) {
	acm := &descCDCACM[d.cc.config-1]
	num := uint16(endpoint) & descEndptAddrNumberMsk
	buf := &acm.rx[num*descCDCACMRxSize]
	d.hw.interruptsEnableUSB(false)
	d.transferPrepare(&acm.rd[num], buf, acm.rxSize, uint32(endpoint))
	d.hw.deleteCache(uintptr(unsafe.Pointer(buf)), uintptr(acm.rxSize))
	d.endpointReceive(descCDCACMEndpointDataRx, &acm.rd[num])
	d.hw.interruptsEnableUSB(true)
}

func (d *dcd) uartNotify(transfer *dcdTransfer) {
	acm := &descCDCACM[d.cc.config-1]
	len := acm.rxSize - (uint16(transfer.token>>16) & 0x7FFF)
	p := transfer.param
	if 0 == len {
		// zero-length packet (ZLP)
		d.uartReceive(uint8(p))
	} else {
		// data packet
		h := acm.rxHead
		if h != acm.rxTail {
			// previous packet is still buffered
			q := acm.rxQueue[h]
			n := acm.rxCount[q]
			if len <= descCDCACMRxSize-n {
				// previous buffer has enough free space for this packet's data
				_ = copy(acm.rx[q*descCDCACMRxSize+n:],
					acm.rx[p*descCDCACMRxSize:uint16(p)*descCDCACMRxSize+len])
				acm.rxCount[q] = n + len
				acm.rxFree += len
				d.uartReceive(uint8(p))
				return
			}
		}
		// add this packet to Rx buffer
		acm.rxCount[p] = len
		acm.rxIndex[p] = 0
		h += 1
		if h > descCDCACMRDCount { // should be >=
			h = 0
		}
		acm.rxQueue[h] = uint16(p)
		acm.rxHead = h
		acm.rxFree += len
	}
}

// uartFlush discards all buffered input (Rx) data.
func (d *dcd) uartFlush() {
	acm := &descCDCACM[d.cc.config-1]
	tail := acm.rxTail
	for tail != acm.rxHead {
		tail += 1
		if tail > descCDCACMRDCount {
			tail = 0
		}
		i := acm.rxQueue[tail]
		acm.rxFree -= acm.rxCount[i] - acm.rxIndex[i]
		d.uartReceive(uint8(i))
		acm.rxTail = tail
	}
}

func (d *dcd) uartAvailable() int {
	return int(descCDCACM[d.cc.config-1].rxFree)
}

func (d *dcd) uartPeek() (uint8, bool) {
	acm := &descCDCACM[d.cc.config-1]
	tail := acm.rxTail
	if tail == acm.rxHead {
		return 0, false
	}
	tail += 1
	if tail > descCDCACMRDCount {
		tail = 0
	}
	i := acm.rxQueue[tail]
	return acm.rx[i*descCDCACMRxSize+acm.rxIndex[i]], true
}

func (d *dcd) uartReadByte() (uint8, bool) {
	b := []uint8{0}
	ok := d.uartRead(b) > 0
	return b[0], ok
}

func (d *dcd) uartRead(data []uint8) int {
	acm := &descCDCACM[d.cc.config-1]
	read := uint16(0)
	size := uint16(len(data))
	tail := acm.rxTail
	dest := uint16(0)
	d.hw.interruptsEnableUSB(false)
	for read < size && tail != acm.rxHead {
		tail += 1
		if tail > descCDCACMRDCount {
			tail = 0
		}
		i := acm.rxQueue[tail]
		count := uint16(size - read)
		avail := acm.rxCount[i] - acm.rxIndex[i]
		start := i*descCDCACMRxSize + acm.rxIndex[i]
		if avail > count {
			// partially consume packet
			_ = copy(data[dest:], acm.rx[start:start+count])
			acm.rxFree -= count
			acm.rxIndex[i] += count
			read += count
		} else {
			// fully consume packet
			_ = copy(data[dest:], acm.rx[start:start+avail])
			dest += avail //* uint16(unsafe.Sizeof(&data[0]))
			read += avail
			acm.rxFree -= avail
			acm.rxTail = tail
			d.uartReceive(uint8(i))
		}
	}
	d.hw.interruptsEnableUSB(true)
	return int(read)
}

func (d *dcd) uartWriteByte(c uint8) bool {
	return 1 == d.uartWrite([]uint8{c})
}

func (d *dcd) uartWrite(data []uint8) int {
	acm := &descCDCACM[d.cc.config-1]
	sent := 0
	size := len(data)
	for size > 0 {
		xfer := &acm.td[acm.txHead]
		wait := false
		when := int64(0)
		for 0 == acm.txFree {
			if 0 == xfer.token&0x80 {
				if 0 != xfer.token&0x68 {
					// TODO: token contains error, how to handle?
				}
				acm.txFree = descCDCACMTxSize
				acm.txPrev = false
				break
			}
			if !wait {
				wait = true
				when = ticks()
			}
			if acm.txPrev {
				return sent
			}
			if ticks()-when > descCDCACMTxTimeoutMs {
				acm.txPrev = true
				return sent
			}
		}
		buff := acm.tx[(int(acm.txHead)*descCDCACMTxSize)+
			(descCDCACMTxSize-int(acm.txFree)):]
		if size > int(acm.txFree) {
			_ = copy(buff, data[sent:sent+int(acm.txFree)])
			tx := &acm.tx[int(acm.txHead)*descCDCACMTxSize]
			d.transferPrepare(xfer, tx, descCDCACMTxSize, 0)
			d.hw.flushCache(uintptr(unsafe.Pointer(tx)), descCDCACMTxSize)
			d.endpointTransmit(descCDCACMEndpointDataTx, xfer)
			acm.txHead += 1
			if acm.txHead >= descCDCACMTDCount {
				acm.txHead = 0
			}
			size -= int(acm.txFree)
			sent += int(acm.txFree)
			acm.txFree = 0
			// d.hw.timerStop(0)
		} else {
			_ = copy(buff, data[:size])
			acm.txFree -= uint16(size)
			sent += size
			size = 0
			// d.hw.timerOneShot(0)
		}
	}
	return sent
}

func (d *dcd) uartSync() {
	const autoFlushTx = true
	if !autoFlushTx {
		return
	}
	acm := &descCDCACM[d.cc.config-1]
	if 0 == acm.txFree {
		return
	}
	xfer := &acm.td[acm.txHead]
	buff := &acm.tx[uint16(acm.txHead)*descCDCACMTxSize]
	size := descCDCACMTxSize - acm.txFree
	d.transferPrepare(xfer, buff, size, 0)
	d.hw.flushCache(uintptr(unsafe.Pointer(buff)), uintptr(size))
	d.endpointTransmit(descCDCACMEndpointDataTx, xfer)
	acm.txHead += 1
	if acm.txHead >= descCDCACMTDCount {
		acm.txHead = 0
	}
	acm.txFree = 0
}
