// +build mimxrt1062

package usb

// Implementation of a port controller for USB device mode on NXP iMXRT1062.

import (
	"device/arm"
	"device/nxp"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

// deviceControl represents a USB device controller for NXP iMXRT1062.
// It implements the USB API's deviceController interface.
type deviceControl struct {
	port   uint8
	device *device
	irq    interrupt.Interrupt

	messages deviceControllerInterruptQueue

	interruptMask uintptr            // interrupt state upon entering critical section
	criticalState volatile.Register8 // set to 1 if in critical section, else 0

	bus           *nxp.USB_Type
	phy           *nxp.USBPHY_Type
	nc            *nxp.USBNC_Type
	qh            *deviceControllerQH     // The QH structure base address
	dtd           *deviceControllerDTD    // The DTD structure base address
	dtdFree       *deviceControllerDTD    // The idle DTD list head
	dtdHead       deviceControllerDTDList // The transferring DTD list head for each endpoint
	dtdTail       deviceControllerDTDList // The transferring DTD list tail for each endpoint
	dtdCount      uint8                   // The idle DTD node count
	endpointCount uint8                   // The endpoint number of EHCI
	isResetting   bool                    // Whether a PORT reset is occurring or not
	controllerId  uint8                   // Controller ID
	speed         uint8                   // Current speed of EHCI
	isSuspending  bool                    // Is suspending of the PORT
}

var (
	// deviceControlInstance holds instances for all USB device controllers
	// available on the platform.
	deviceControlInstance [configDeviceCount]deviceControl

	//go:align 2048
	deviceControllerQHBuffer [deviceControllerQHBufferSize]uint8
	//go:align 32
	deviceControllerDTDBuffer [deviceControllerDTDBufferSize]uint8
)

// We cannot use the sleep timer from this context (import cycle), but we need
// an approximate method to spin CPU cycles for short periods of time.
//go:inline
func delayMicrosec(microsec uint32) {
	n := cycles(microsec, configCPUFrequencyHz)
	for i := uint32(0); i < n; i++ {
		arm.Asm(`nop`)
	}
}

// initController returns a deviceController for the receiver USB device.
// It allocates the registers and installs/enables an interrupt handler for
// the receiver USB device. It also initializes the shared buffers used by the
// USB controller hardware.
func (d *device) initController() deviceController {

	dc := &deviceControlInstance[d.port]

	dc.port = d.port
	dc.device = d

	// based on the selected port, install interrupt handler and initialize
	// register references
	switch d.port {
	case 0:
		dc.irq = interrupt.New(nxp.IRQ_USB_OTG1, func(interrupt.Interrupt) {
			portInstance[0].device.controller.interrupt()
		})
		dc.bus = nxp.USB1
		dc.phy = nxp.USBPHY1
		dc.nc = nxp.USBNC1
	case 1:
		dc.irq = interrupt.New(nxp.IRQ_USB_OTG2, func(interrupt.Interrupt) {
			//portInstance[1].device.controller.interrupt()
		})
		dc.bus = nxp.USB2
		dc.phy = nxp.USBPHY2
		dc.nc = nxp.USBNC2
	}

	// get base address of QH and DTD buffers
	dc.qh = (*deviceControllerQH)(unsafe.Pointer(
		&deviceControllerQHBuffer[int(d.port)*configDeviceControllerQHAlign]))
	dc.dtd = (*deviceControllerDTD)(unsafe.Pointer(
		&deviceControllerDTDBuffer[int(d.port)*configDeviceControllerDTDAlign]))

	dc.irq.SetPriority(configInterruptPriority)
	dc.irq.Enable()

	return dc
}

// init initializes the USB device subsystem for the receiver deviceControl.
func (dc *deviceControl) init() status {

	// reset the controller
	dc.bus.USBCMD.SetBits(nxp.USB_USBCMD_RST)
	for dc.bus.USBCMD.HasBits(nxp.USB_USBCMD_RST) {
	}

	// get hardware's endpoint count
	dc.endpointCount = uint8(dc.bus.DCCPARAMS.Get() & nxp.USB_DCCPARAMS_DEN_Msk)
	if dc.endpointCount < configDeviceMaxEndpoints {
		return statusError
	}

	// clear the controller mode field and set to device mode:
	//   controller mode (CM) 0x0=idle, 0x2=device-only, 0x3=host-only
	dc.bus.USBMODE.ReplaceBits(nxp.USB_USBMODE_CM_CM_2,
		nxp.USB_USBMODE_CM_Msk>>nxp.USB_USBMODE_CM_Pos, nxp.USB_USBMODE_CM_Pos)

	// reset the constroller state to default
	return dc.resetState()
}

func (dc *deviceControl) deinit() status {
	return statusSuccess
}

func (dc *deviceControl) enable(enable bool) status {
	if enable {
		// ensure D+ pulled down long enough for host to detect previous disconnect
		delayMicrosec(5000)
		return dc.control(deviceControlRun, nil)
	}
	// TODO: handle disabling
	return statusSuccess
}

// interrupt is the base interrupt handler for all USB device interrupts.
func (dc *deviceControl) interrupt() {

	// protect access to message queue
	for statusRetry == dc.critical(true) {
	}

	// read and clear the interrupts that fired
	status := dc.bus.USBSTS.Get() & dc.bus.USBINTR.Get()
	dc.bus.USBSTS.Set(status)

	// enqueue interrupts for runtime processing
	dc.messages.enq(uintptr(status))

	// release message queue
	_ = dc.critical(false)
}

func (dc *deviceControl) process() status {

	// protect access to message queue
	if dc.critical(true).OK() {

		// dequeue oldest interrupt in message queue
		status, ok := dc.messages.deq()

		// release message queue
		_ = dc.critical(false)

		// process message if queue was not empty
		if ok {

			if 0 != (status & nxp.USB_USBSTS_URI_Msk) { // USB reset
				dc.reset()
			}

			if 0 != (status & nxp.USB_USBSTS_UI_Msk) { // USB token done
				dc.tokenDone()
			}

			if 0 != (status & nxp.USB_USBSTS_PCI_Msk) { // USB port status change
				dc.portChange()
			}

			if 0 != (status & nxp.USB_USBSTS_SRI_Msk) { // USB start of frame (SOF)
				dc.frameStart()
			}
		}

		// message queue read and processed
		return statusSuccess
	}

	// could not acquire lock on message queue
	return statusBusy
}

func (dc *deviceControl) send(address uint8, buffer []uint8, length uint32) status {
	return statusSuccess
}

func (dc *deviceControl) receive(address uint8, buffer []uint8, length uint32) status {
	return statusSuccess
}

func (dc *deviceControl) cancel(address uint8) status {
	return statusSuccess
}

func (dc *deviceControl) control(command deviceControlID, param interface{}) (s status) {

	// assume success unless error condition deliberately detected
	s = statusSuccess

	switch command {
	case deviceControlRun:
		dc.bus.USBCMD.SetBits(nxp.USB_USBCMD_RS)

	case deviceControlStop:
		dc.bus.USBCMD.ClearBits(nxp.USB_USBCMD_RS)

	case deviceControlEndpointInit:
		config, ok := param.(deviceEndpointConfig)
		if !ok {
			return statusInvalidParameter
		}
		s = dc.initEndpoint(config)

	case deviceControlEndpointDeinit:
		address, ok := param.(uint8)
		if !ok {
			return statusInvalidParameter
		}
		s = dc.deinitEndpoint(address)

	case deviceControlEndpointStall:
		address, ok := param.(uint8)
		if !ok {
			return statusInvalidParameter
		}
		s = dc.stallEndpoint(address)

	case deviceControlEndpointUnstall:
		address, ok := param.(uint8)
		if !ok {
			return statusInvalidParameter
		}
		s = dc.unstallEndpoint(address)

	case deviceControlGetDeviceStatus:
		// param should be a pointer to uint16, acting as output parameter.
		status, ok := param.(*uint16)
		if !ok {
			return statusInvalidController
		}
		// configDeviceSelfPowered is a configuration constant on iMXRT1062
		*status = configDeviceSelfPowered <<
			specRequestStandardGetStatusDeviceSelfPoweredPos

	case deviceControlGetEndpointStatus:
		// TODO

	case deviceControlPreSetDeviceAddress:
		address, ok := param.(uint8)
		if !ok {
			return statusInvalidController
		}
		dc.bus.DEVICEADDR.Set((uint32(address) << nxp.USB_DEVICEADDR_USBADR_Pos) |
			nxp.USB_DEVICEADDR_USBADRA_Msk)

	case deviceControlSetDeviceAddress:
		// TODO

	case deviceControlGetSynchFrame:
		return statusNotSupported

	case deviceControlSetDefaultStatus:
		for i := uint8(0); i < configDeviceMaxEndpoints; i++ {
			_ = dc.deinitEndpoint(i | specDescriptorEndpointAddressDirectionIn)
			_ = dc.deinitEndpoint(i | specDescriptorEndpointAddressDirectionOut)
		}
		s = dc.resetState()

	case deviceControlGetSpeed:
		// param should be a pointer to uint8, acting as output parameter.
		speed, ok := param.(*uint8)
		if !ok {
			return statusInvalidController
		}
		*speed = dc.speed

	case deviceControlGetOTGStatus:
		return statusNotSupported

	case deviceControlSetOTGStatus:
		return statusNotSupported
	}

	return
}

func (dc *deviceControl) critical(enter bool) status {
	if enter {
		// check if critical section already locked
		if dc.criticalState.Get() != 0 {
			return statusRetry
		}
		// lock critical section
		dc.criticalState.Set(1)
		// disable interrupts, storing state in receiver
		dc.interruptMask = arm.DisableInterrupts()
	} else {
		// ensure critical section is locked
		if dc.criticalState.Get() != 0 {
			// re-enable interrupts, using state stored in receiver
			arm.EnableInterrupts(dc.interruptMask)
			// unlock critical section
			dc.criticalState.Set(0)
		}
	}
	return statusSuccess
}

func (dc *deviceControl) resetState() status {

	dc.dtdFree = dc.dtd
	p := dc.dtdFree
	for i := 1; i < configDeviceControllerMaxDTD; i++ {
		(*p).nextDTDPointer = getDTDBuffer(dc.port, i)
		p = (*p).nextDTDPointer
	}
	(*p).nextDTDPointer = nil
	dc.dtdCount = configDeviceControllerMaxDTD

	// no interrupt threshold
	dc.bus.USBCMD.ClearBits(nxp.USB_USBCMD_ITC_Msk)

	// disable setup lockout
	dc.bus.USBMODE.SetBits(nxp.USB_USBMODE_SLOM_Msk)

	// use little-endianness
	dc.bus.USBMODE.ClearBits(nxp.USB_USBMODE_ES_Msk)

	for i := 0; i < 2*configDeviceMaxEndpoints; i++ {
		qh := getQHBuffer(dc.port, 0, i)
		(*qh).capabilities =
			deviceControllerCapabilities{
				maxPacketSize: configDeviceControllerMaxPacketSize,
			}.pack()
		(*qh).endpointStatus =
			deviceControllerEndpointStatus{
				isOpened: 0,
			}.pack()
		(*qh).nextDTDPointer = deviceControllerDTDTerminate
		dc.dtdHead[i] = nil
		dc.dtdTail[i] = nil
	}
	dc.bus.ASYNCLISTADDR.Set(uint32(uintptr(unsafe.Pointer(
		getQHBuffer(dc.port, 0, 0)))))

	dc.bus.DEVICEADDR.Set(0)

	// enable interrupts: bus enable, bus error, port change detect, bus reset
	dc.bus.USBINTR.Set(nxp.USB_USBINTR_UE_Msk | nxp.USB_USBINTR_UEE_Msk |
		nxp.USB_USBINTR_PCE_Msk | nxp.USB_USBINTR_URE_Msk)

	dc.isResetting = false

	return statusSuccess
}

func (dc *deviceControl) reset() {

	// clear setup flag
	dc.bus.ENDPTSETUPSTAT.Set(dc.bus.ENDPTSETUPSTAT.Get())
	// clear endpoint complete flag
	dc.bus.ENDPTCOMPLETE.Set(dc.bus.ENDPTCOMPLETE.Get())

	// flush any pending transfers
	for dc.bus.ENDPTPRIME.HasBits(nxp.USB_ENDPTPRIME_PERB_Msk | nxp.USB_ENDPTPRIME_PETB_Msk) {
		dc.bus.ENDPTFLUSH.Set(nxp.USB_ENDPTFLUSH_FERB_Msk | nxp.USB_ENDPTFLUSH_FETB_Msk)
	}

	// set receiver flag if port reset bit is set; otherwise, notify device class.
	if dc.bus.PORTSC1.HasBits(nxp.USB_PORTSC1_PR_Msk) {
		dc.isResetting = true
	} else {
		// send reset notification to common device
		dc.device.notify(deviceNotification{code: deviceNotifyBusReset})
	}
}

func (dc *deviceControl) tokenDone() {

}

func (dc *deviceControl) portChange() {

}

func (dc *deviceControl) frameStart() {

}

func (dc *deviceControl) initEndpoint(config deviceEndpointConfig) status {
	return statusSuccess
}

func (dc *deviceControl) deinitEndpoint(address uint8) status {
	return statusSuccess
}

func (dc *deviceControl) stallEndpoint(address uint8) status {
	return statusSuccess
}

func (dc *deviceControl) unstallEndpoint(address uint8) status {
	return statusSuccess
}
