// +build mimxrt1062

package usb

import (
	"device/nxp"
	"runtime/interrupt"
	"unsafe"
)

type deviceControl struct {
	port   uint8
	device *device
	irq    interrupt.Interrupt

	bus           *nxp.USB_Type
	phy           *nxp.USBPHY_Type
	nc            *nxp.USBNC_Type
	qh            deviceControllerQHHandle  // The QH structure base address
	dtd           deviceControllerDTDHandle // The DTD structure base address
	dtdFree       deviceControllerDTDHandle // The idle DTD list head
	dtdHead       deviceControllerDTDList   // The transferring DTD list head for each endpoint
	dtdTail       deviceControllerDTDList   // The transferring DTD list tail for each endpoint
	dtdCount      uint8                     // The idle DTD node count
	endpointCount uint8                     // The endpoint number of EHCI
	isResetting   bool                      // Whether a PORT reset is occurring or not
	controllerId  uint8                     // Controller ID
	speed         uint8                     // Current speed of EHCI
	isSuspending  bool                      // Is suspending of the PORT
}

var (
	deviceControlInstance [configDeviceCount]deviceControl

	//go:align 2048
	deviceControllerQHBuffer [deviceControllerQHBufferSize]uint8
	//go:align 32
	deviceControllerDTDBuffer [deviceControllerDTDBufferSize]uint8
)

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
	dc.qh = (deviceControllerQHHandle)(unsafe.Pointer(
		&deviceControllerQHBuffer[int(d.port)*configDeviceControllerQHAlign]))
	dc.dtd = (deviceControllerDTDHandle)(unsafe.Pointer(
		&deviceControllerDTDBuffer[int(d.port)*configDeviceControllerDTDAlign]))

	dc.irq.SetPriority(configInterruptPriority)
	dc.irq.Enable()

	return dc
}

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

func (dc *deviceControl) interrupt() {

	// mask interrupts with enabled bits
	status := dc.bus.USBSTS.Get() & dc.bus.USBINTR.Get()

	// clear interrupts
	dc.bus.USBSTS.Set(status)

	if 0 != (status & nxp.USB_USBSTS_URI_Msk) { // Reset
		dc.interruptReset()
	}

	if 0 != (status & nxp.USB_USBSTS_UI_Msk) { // Token done
		dc.interruptTokenDone()
	}

	if 0 != (status & nxp.USB_USBSTS_PCI_Msk) { // Port status change
		dc.interruptPortChange()
	}

	if 0 != (status & nxp.USB_USBSTS_SRI_Msk) { // Sof
		dc.interruptSof()
	}
}

func (dc *deviceControl) send(endpointAddress uint8, buffer []uint8, length uint32) status {
	return statusSuccess
}

func (dc *deviceControl) receive(endpointAddress uint8, buffer []uint8, length uint32) status {
	return statusSuccess
}

func (dc *deviceControl) cancel(endpointAddress uint8) status {
	return statusSuccess
}

func (dc *deviceControl) control(command deviceControlID, param interface{}) (s status) {

	// assume success unless error condition deliberately detected
	s = statusSuccess

	switch command {
	case deviceControlRun:
		dc.bus.USBCMD.SetBits(nxp.USB_USBCMD_RS)
	}

	return
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

	// use little endianness
	dc.bus.USBMODE.ClearBits(nxp.USB_USBMODE_ES_Msk)

	for i := 0; i < 2*configDeviceMaxEndpoints; i++ {
		qh := getQHBuffer(dc.port, 0, i)
		(*qh).capabilities = deviceControllerCapabilities{
			maxPacketSize: configDeviceControllerMaxPacketSize,
		}.pack()
		(*qh).endpointStatus = deviceControllerEndpointStatus{
			isOpened: 0,
		}.pack()
		(*qh).nextDTDPointer = (deviceControllerDTDHandle)(unsafe.Pointer(uintptr(1)))
		dc.dtdHead[i] = nil
		dc.dtdTail[i] = nil
	}
	dc.bus.ASYNCLISTADDR.Set(uint32(uintptr(unsafe.Pointer(
		getQHBuffer(dc.port, 0, 0)))))

	dc.bus.DEVICEADDR.Set(0)

	// enable interrupts: enable, error, port change detect, reset
	dc.bus.USBINTR.Set(nxp.USB_USBINTR_UE_Msk | nxp.USB_USBINTR_UEE_Msk |
		nxp.USB_USBINTR_PCE_Msk | nxp.USB_USBINTR_URE_Msk)

	dc.isResetting = false

	return statusSuccess
}

func (dc *deviceControl) interruptReset() {

	// clear setup flag
	status := dc.bus.ENDPTSETUPSTAT.Get()
	dc.bus.ENDPTSETUPSTAT.Set(status)
	// clear endpoint complete flag
	status = dc.bus.ENDPTCOMPLETE.Get()
	dc.bus.ENDPTCOMPLETE.Set(status)

	flush := true
	for flush {
		// flush the pending transfers
		dc.bus.ENDPTFLUSH.Set(nxp.USB_ENDPTFLUSH_FERB_Msk | nxp.USB_ENDPTFLUSH_FETB_Msk)
		flush = dc.bus.ENDPTPRIME.HasBits(nxp.USB_ENDPTPRIME_PERB_Msk | nxp.USB_ENDPTPRIME_PETB_Msk)
	}

	// if port reset, set flag; otherwise, notify device class
	if dc.bus.PORTSC1.HasBits(nxp.USB_PORTSC1_PR_Msk) {
		dc.isResetting = true
	} else {
		dc.device.notification(&deviceCallbackMessage{
			buffer:  nil,
			code:    deviceNotifyBusReset,
			length:  0,
			isSetup: false,
		})
	}
}

func (dc *deviceControl) interruptTokenDone() {

}

func (dc *deviceControl) interruptPortChange() {

}

func (dc *deviceControl) interruptSof() {

}
