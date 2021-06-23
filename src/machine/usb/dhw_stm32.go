// +build stm32

package usb

// Implementation of USB device controller hardware abstraction (dhw) for STM32
// family of microcontrollers.

import (
	"device/stm32"
	"math/bits"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

// dhwInterruptPriority defines the priority for all USB device interrupts.
const dhwInterruptPriority = 1

type dhwState int

const (
	dhwStateInit dhwState = iota
	dhwStateError
	dhwStateDefault
	dhwStateAddress
	dhwStateConfigure
	dhwStateSuspend
)

// dhw implements USB device controller hardware abstraction for iMXRT1062.
type dhw struct {
	*dcd // USB device controller driver

	glo *stm32.USB_GLOBAL_Type // USB global registers
	dev *stm32.USB_DEVICE_Type // USB device registers
	pcc *volatile.Register32   // USB PWR/CLKCTL register

	irq interrupt.Interrupt // USB system interrupt

	state dhwState // State of the dhw's USB state machine
	stage dcdStage // USB transmission stage

	speed Speed // USB bus speed or transfer rate

	controlReply [8]uint8
	controlMask  uint32
	endpointMask uint32
	setup        dcdSetup

	address uint32 // USB device address (defined by the host)
}

// allocDHW returns a reference to the USB hardware abstraction for the given
// device controller driver. Should be called only one time and during device
// controller initialization.
func allocDHW(port, instance int, speed Speed, dc *dcd) *dhw {

	switch port {
	case 0:
		dhwInstance[instance].dcd = dc
		dhwInstance[instance].glo = stm32.USB_GLOBAL1
		dhwInstance[instance].dev = stm32.USB_DEVICE1
		dhwInstance[instance].pcc = stm32.USB_PCCTRL1
		dhwInstance[instance].irq = makeInterrupt(port)

	case 1:
		dhwInstance[instance].dcd = dc
		dhwInstance[instance].glo = stm32.USB_GLOBAL2
		dhwInstance[instance].dev = stm32.USB_DEVICE2
		dhwInstance[instance].pcc = stm32.USB_PCCTRL2
		dhwInstance[instance].irq = makeInterrupt(port)
	}

	// All ports default to full-speed during initialization. An interrupt event
	// is raised if/when a USB port connect/status change occurs in which the
	// host signals a different speed is to be used. At that point, this speed
	// field will be updated accordingly.
	// Port 1 supports full-speed only. You must use port 0 for high-speed.
	if 0 == speed || 1 == port {
		speed = FullSpeed
	}
	dhwInstance[instance].speed = speed

	return &dhwInstance[instance]
}

func runBootloader() {}

// cycleCount uses the ARM debug cycle counter available on iMXRT1062 (enabled
// in runtime_mimxrt1062_time.go) to return the number of CPU cycles since boot.
//go:inline
func cycleCount() uint32 {
	return (*volatile.Register32)(unsafe.Pointer(uintptr(0xe0001004))).Get()
}

// init configures the USB port for device mode operation by initializing all
// endpoint and transfer descriptor data structures, initializing core registers
// and interrupts, resetting the USB PHY, and enabling power on the bus.
func (d *dhw) init() status {

	// Initialize state of USB state machine
	d.state = dhwStateInit

	// Disable USB pullup and system interrupt
	d.enable(false)

	// Initialize USB peripheral clocks
	d.enableClocks(true)

	// Initialize USB peripheral core
	if st := d.initCore(); !st.ok() {
		return st
	}

	// Configure USB peripheral for device mode operation
	if st := d.initDevice(); !st.ok() {
		return st
	}

	// Device configuration is successful if nothing modified our device state.
	if dhwStateInit == d.state {
		// At this point, we transition to the USB-specified "Default" state,
		// representing an initialized device out of reset.
		d.state = dhwStateDefault
	}

	// Initialize USB interrupt priority
	d.irq.SetPriority(dhwInterruptPriority)

	// Clear and enable system interrupt
	d.resetInterrupt()

	return statusOK
}

// initCore initializes the USB PHY peripheral core.
func (d *dhw) initCore() status {

	switch d.speed {
	case LowSpeed, FullSpeed:

		// Select FS embedded PHY
		d.glo.GUSBCFG.SetBits(stm32.USB_GUSBCFG_PHYSEL)

		// Reset core
		if !d.resetCore() {
			d.state = dhwStateError
			return statusFail
		}

		// Activate USB full-speed transceiver
		d.glo.GCCFG.SetBits(stm32.USB_GCCFG_PWRDWN)

	case HighSpeed, SuperSpeed, DualSuperSpeed:

		// Deactivate USB full-speed transceiver
		d.glo.GCCFG.ClearBits(stm32.USB_GCCFG_PWRDWN)

		// Initialize ULPI interface
		d.glo.GUSBCFG.ClearBits(stm32.USB_GUSBCFG_TSDPS |
			stm32.USB_GUSBCFG_ULPIFSLS | stm32.USB_GUSBCFG_PHYSEL)

		// Select VBUS source
		d.glo.GUSBCFG.ClearBits(stm32.USB_GUSBCFG_ULPIEVBUSD |
			stm32.USB_GUSBCFG_ULPIEVBUSI)

		// Reset core
		if !d.resetCore() {
			d.state = dhwStateError
			return statusFail
		}
	}

	// Burst length: 4x 32-bit bus accesses
	d.glo.GAHBCFG.SetBits(0x3 << stm32.USB_GAHBCFG_HBSTLEN_Pos)

	// Enable internal DMA
	// d.glo.GAHBCFG.SetBits(stm32.USB_GAHBCFG_DMAEN)

	return statusOK
}

// initDevice initializes the USB core registers and core interrupts for device
// mode operation.
func (d *dhw) initDevice() status {

	// Set USB device mode, disable session request protocl (SRP) and host
	// negotiation protocol (HNP)
	d.glo.GUSBCFG.ClearBits(stm32.USB_GUSBCFG_FHMOD |
		stm32.USB_GUSBCFG_SRPCAP | stm32.USB_GUSBCFG_HNPCAP)
	d.glo.GUSBCFG.SetBits(stm32.USB_GUSBCFG_FDMOD)

	udelay(50000) // 50 ms

	// Disable USB pullup/pulldown
	d.dev.DCTL.SetBits(stm32.USB_DCTL_SDIS)

	// Disable VBUS sensing
	d.glo.GCCFG.ClearBits(stm32.USB_GCCFG_VBDEN)
	d.glo.GOTGCTL.SetBits(stm32.USB_GOTGCTL_BVALOEN | stm32.USB_GOTGCTL_BVALOVAL)

	// Restart PHY clock
	d.pcc.Set(0)

	// Configure device interface
	d.setDeviceSpeed(d.speed)
	d.dev.DCFG.ClearBits(stm32.USB_DCFG_NZLSOHSK) // Non-ZLP status handshake

	// Flush all FIFOs
	d.endpointFlush(descEndptAddrNumberMsk + 1) // all Tx FIFOs
	d.endpointFlush(rxEndpoint(0))              // Rx FIFO

	// Initialize all Tx FIFO sizes (0 words)
	for i := range d.glo.DIEPTXF {
		d.glo.DIEPTXF[i].Set(0)
	}

	// Clear all pending device interrupts
	d.dev.DIEPMSK.Set(0)
	d.dev.DOEPMSK.Set(0)
	d.dev.DAINTMSK.Set(0)

	for i := 0; i < int(d.endpointCount()); i++ {
		// IN endpoint
		ie, ic := d.glo.InEp(i), uint32(0)
		// If for some reason the endpoint is enabled, set NAK and disconnect
		if ie.DIEPCTL.HasBits(stm32.USB_DIEPCTL_EPENA) {
			ic |= stm32.USB_DIEPCTL_SNAK
			if i > 0 {
				ic |= stm32.USB_DIEPCTL_EPDIS
			}
		}
		ie.DIEPCTL.Set(ic)
		ie.DIEPTSIZ.Set(0)
		ie.DIEPINT.Set(0xFB7F)
		// OUT endpoint
		oe, oc := d.glo.OutEp(i), uint32(0)
		// If for some reason the endpoint is enabled, set NAK and disconnect
		if oe.DOEPCTL.HasBits(stm32.USB_DOEPCTL_EPENA) {
			oc |= stm32.USB_DOEPCTL_SNAK
			if i > 0 {
				oc |= stm32.USB_DOEPCTL_EPDIS
			}
		}
		oe.DOEPCTL.Set(oc)
		oe.DOEPTSIZ.Set(0)
		oe.DOEPINT.Set(0xFB7F)
	}

	// Trigger Tx FIFO empty interrupt when FIFO _completely_ empty. The only
	// other option available is to trigger interrupt when FIFO _half_ empty,
	// which will probably be useful once DMA support is added.
	d.glo.GAHBCFG.SetBits(stm32.USB_GAHBCFG_TXFELVL)

	d.glo.GINTMSK.Set(0)          // Disable all internal interrupts
	d.glo.GINTSTS.Set(0xFFFFFFFF) // Clear all internal interrupts

	// Unmask core interrupts
	d.glo.GINTMSK.SetBits(
		stm32.USB_GINTMSK_WUIM | //                  31: Resume/remote wakeup detected
			stm32.USB_GINTMSK_SRQIM | //               30: Session request/new session detected
			stm32.USB_GINTMSK_DISCINT | //             29: Disconnect detected
			stm32.USB_GINTMSK_CIDSCHGM | //            28: Connector ID status change
			stm32.USB_GINTMSK_LPMINTM | //             27: Low-power mode
			// stm32.USB_GINTMSK_PTXFEM |  //          26: Periodic Tx FIFO empty
			// stm32.USB_GINTMSK_HCIM |  //            25: Host channels
			// stm32.USB_GINTMSK_PRTIM |  //           24: Host port
			stm32.USB_GINTMSK_RSTDEM | //              23: Reset detected
			stm32.USB_GINTMSK_FSUSPM | //              22: Data fetch suspended
			// stm32.USB_GINTMSK_PXFRM_IISOOXFRM | //  21: Incomplete periodic transfer
			//                                         21: Incomplete isochronous OUT transfer
			// stm32.USB_GINTMSK_IISOIXFRM | //        20: Incomplete isochronous IN transfer
			stm32.USB_GINTMSK_OEPINT | //              19: OUT endpoints
			stm32.USB_GINTMSK_IEPINT | //              18: IN endpoints
			//                                         17: Reserved
			//                                         16: Reserved
			// stm32.USB_GINTMSK_EOPFM | //            15: End of periodic frame
			// stm32.USB_GINTMSK_ISOODRPM | //         14: Isochronous OUT packet dropped
			stm32.USB_GINTMSK_ENUMDNEM | //            13: Enumeration done
			stm32.USB_GINTMSK_USBRST | //              12: USB reset
			stm32.USB_GINTMSK_USBSUSPM | //            11: USB suspend
			stm32.USB_GINTMSK_ESUSPM | //              10: Early suspend
			//                                          9: Reserved
			//                                          8: Reserved
			stm32.USB_GINTMSK_GONAKEFFM | //            7: Global OUT NAK effective
			stm32.USB_GINTMSK_GINAKEFFM | //            6: Global non-periodic IN NAK effective
			stm32.USB_GINTMSK_NPTXFEM | //              5: Non-periodic Tx FIFO empty
			stm32.USB_GINTMSK_RXFLVLM | //              4: Rx FIFO non-empty
			stm32.USB_GINTMSK_SOFM | //                 3: Start-of-frame
			stm32.USB_GINTMSK_OTGINT | //               2: OTG interrupt
			stm32.USB_GINTMSK_MMISM, //                 1: Mode mismatch
	)

	return statusOK
}

// resetCore performs a soft reset on the USB core and returns true if and only
// if the core reset to its initial operating state successfully.
func (d *dhw) resetCore() bool {
	// Wait for bus idle
	for !d.glo.GRSTCTL.HasBits(stm32.USB_GRSTCTL_AHBIDL) {
	}
	// Core soft reset
	d.glo.GRSTCTL.SetBits(stm32.USB_GRSTCTL_CSRST)
	// Wait for reset bit to clear
	for d.glo.GRSTCTL.HasBits(stm32.USB_GRSTCTL_CSRST) {
	}
	return true
}

func (d *dhw) resetEndpoints() {

	// Initialize the USB FIFOs and endpoint buffers
	switch d.cc.id {
	// CDC-ACM (single)
	case classDeviceCDCACM:

		// Device Rx FIFO recommended size (in 32-bit words):
		//   (5 * number of control endpoints + 8) +
		//   (largest USB packet used / 4 + 1 for status information) +
		//   (2 * number of OUT endpoints) + (1 for Global NAK)
		//
		// Note: the entire sum is multiplied by 4 to convert from words to bytes
		const rxSize = 4 *
			((5*1 + 8) + (descEndptMaxPktSize/4 + 1) + (2 * 2) + (1))

		// Control endpoint 0 (Rx FIFO)
		d.endpointAlloc(rxEndpoint(descCDCACMEndpointCtrl),
			descEndptTypeControl, rxSize)
		// Control endpoint 0 (Tx FIFO 0)
		d.endpointAlloc(txEndpoint(descCDCACMEndpointCtrl),
			descEndptTypeControl, descEndptMaxPktSize)
		// Status interrupt endpoint 1 (Tx FIFO 1)
		d.endpointAlloc(txEndpoint(descCDCACMEndpointStatus),
			descEndptTypeInterrupt, descCDCACMStatusPacketSize)
		// Data Rx bulk endpoint 2 (Rx FIFO)
		d.endpointAlloc(rxEndpoint(descCDCACMEndpointDataRx),
			descEndptTypeBulk, descCDCACMDataRxPacketSize)
		// Data Tx bulk endpoint 3 (Tx FIFO 3)
		d.endpointAlloc(txEndpoint(descCDCACMEndpointDataTx),
			descEndptTypeBulk, descCDCACMDataTxPacketSize)

	// HID
	case classDeviceHID:
	default:
		// Unhandled device class
	}

}

// enable enables or disables the pullup/pulldown resistor on the USB D+ line,
// forcing electrical termination between host and device, and enables or
// disables USB system interrupts.
func (d *dhw) enable(enable bool) {
	if enable {
		// Enable pullup/pulldown
		d.dev.DCTL.ClearBits(stm32.USB_DCTL_SDIS)
		// Enable controller's global interrupt
		d.glo.GAHBCFG.SetBits(stm32.USB_GAHBCFG_GINT)
		// Enable USB interrupts
		d.irq.Enable()
	} else {
		// Disable USB interrupts
		d.irq.Disable()
		// Disable controller's global interrupt
		d.glo.GAHBCFG.ClearBits(stm32.USB_GAHBCFG_GINT)
		// Disable pullup/pulldown
		d.dev.DCTL.SetBits(stm32.USB_DCTL_SDIS)
	}
	// Ensure line pulled long enough for host to detect change
	udelay(3000)
}

// enableSOF enables or disables start-of-frame (SOF) interrupts on the given
// USB device interface.
func (d *dhw) enableSOF(enable bool, iface uint8) {

}

// interrupt handles the USB hardware interrupt events and notifies the device
// controller driver using a common "virtual interrupt" code.
func (d *dhw) interrupt() {
	d.irq.Disable()

	// Identify all core interrupts we need to handle based on the configured
	// interrupt mask, and then clear each of their interrupt pending flags.
	coreStatus := d.glo.GINTSTS.Get() & d.glo.GINTMSK.Get()
	d.glo.GINTSTS.Set(coreStatus)

	// Bit 1 MMIS: Mode mismatch interrupt
	//   The core sets this bit when the application is trying to access:
	//     – A host mode register, when the core is operating in device mode.
	//     – A device mode register, when the core is operating in host mode.
	//   The register access is completed on AHB with result OK, but is ignored
	//   by the core internally and does not affect the operation of the core.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_MMIS) {
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_MMIS)
	}

	// Bit 4 RXFLVL: Rx FIFO non-empty
	//   Indicates that there is at least one packet pending to be read from the
	//   Rx FIFO.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_RXFLVL) {
		d.glo.GINTMSK.ClearBits(stm32.USB_GINTSTS_RXFLVL)

		// Constant values for the packet status (PKTSTS) field of Rx status read &
		// pop register (GRXSTSP). Since most of these trigger a distinct interrupt,
		// we only need to handle the ones that do not.
		const (
			statusNAK              = 1 // 0001 - triggers interrupt
			statusDataReceived     = 2 // 0010
			statusTransferComplete = 3 // 0011 - triggers interrupt
			statusSetupComplete    = 4 // 0100 - triggers interrupt
			statusSetupReceived    = 6 // 0110
		)

		// Pop top of the Rx FIFO (required to trigger the interrupts above).
		status := d.glo.GRXSTSP.Get()
		info := d.endpointInfo(rxEndpoint(uint8(status & stm32.USB_GRXSTSP_EPNUM)))
		size := (status & stm32.USB_GRXSTSP_BCNT) >> stm32.USB_GRXSTSP_BCNT_Pos
		// dpid := (status & stm32.USB_GRXSTSP_DPID) >> stm32.USB_GRXSTSP_DPID_Pos

		switch (status & stm32.USB_GRXSTSP_PKTSTS) >> stm32.USB_GRXSTSP_PKTSTS_Pos {
		case statusDataReceived:
			if size > 0 {
				d.endpointRead(rxEndpoint(0), d.controlBuffer(int(info.count)), size)
			}
		case statusSetupReceived:
			// Read 8 bytes from control endpoint 0 Rx FIFO
			d.endpointRead(rxEndpoint(0), d.controlBuffer(0), size)
			d.setup = setupFrom(d.controlBuffer(0))
		}
		d.glo.GINTMSK.SetBits(stm32.USB_GINTSTS_RXFLVL)
	}

	// Bit 19 OEPINT: OUT endpoint interrupt
	//   The core sets this bit to indicate that an interrupt is pending on one of
	//   the OUT endpoints of the core (in device mode). The application must read
	//   the OTG_DAINT register to determine the exact number of the OUT endpoint
	//   on which the interrupt occurred, and then read the corresponding
	//   OTG_DOEPINTx register to determine the exact cause of the interrupt. The
	//   application must clear the appropriate status bit in the corresponding
	//   OTG_DOEPINTx register to clear this bit.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_OEPINT) {
		rx := (d.dev.DAINT.Get() & d.dev.DAINTMSK.Get()) >> 16
		for 0 != rx {
			num := bits.TrailingZeros32(rx)
			rx &^= 1 << num

			reg := d.glo.OutEp(num).DOEPINT.Get() & d.dev.DOEPMSK.Get()

			if stm32.USB_DOEPINT_XFRC == reg&stm32.USB_DOEPINT_XFRC {
				d.glo.OutEp(num).DOEPINT.Set(stm32.USB_DOEPINT_XFRC)
				// out transfer complete
				if (0 == num) && (0 == d.endpointInfo(rxEndpoint(0)).len) {
					d.endpointEnable(0, true, 0)
				}
				// d.endpointDataOut(rxEndpoint(uint8(num)))
			}
			if stm32.USB_DOEPINT_STUP == reg&stm32.USB_DOEPINT_STUP {
				d.glo.OutEp(num).DOEPINT.Set(stm32.USB_DOEPINT_STUP)
				d.event(dcdEvent{
					id:    dcdEventControlSetup,
					setup: d.setup,
				})
			}
			if stm32.USB_DOEPINT_OTEPDIS == reg&stm32.USB_DOEPINT_OTEPDIS {
				d.glo.OutEp(num).DOEPINT.Set(stm32.USB_DOEPINT_OTEPDIS)
			}
			if stm32.USB_DOEPINT_OTEPSPR == reg&stm32.USB_DOEPINT_OTEPSPR {
				d.glo.OutEp(num).DOEPINT.Set(stm32.USB_DOEPINT_OTEPSPR)
			}
			if stm32.USB_DOEPINT_NAK == reg&stm32.USB_DOEPINT_NAK {
				d.glo.OutEp(num).DOEPINT.Set(stm32.USB_DOEPINT_NAK)
			}
		}
	}

	// Bit 18 IEPINT: IN endpoint interrupt
	//   The core sets this bit to indicate that an interrupt is pending on one of
	//   the IN endpoints of the core (in device mode). The application must read
	//   the OTG_DAINT register to determine the exact number of the IN endpoint
	//   on which the interrupt occurred, and then read the corresponding
	//   OTG_DIEPINTx register to determine the exact cause of the interrupt. The
	//   application must clear the appropriate status bit in the corresponding
	//   OTG_DIEPINTx register to clear this bit.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_IEPINT) {
		tx := (d.dev.DAINT.Get() & d.dev.DAINTMSK.Get()) & 0xFFFF
		for 0 != tx {
			num := bits.TrailingZeros32(tx)
			tx &^= 1 << num

			reg := d.glo.InEp(num).DIEPINT.Get() & d.dev.DIEPMSK.Get()
			// (d.dev.DIEPMSK.Get() | (((d.dev.DIEPEMPMSK.Get() >>
			// 	(num & descEndptAddrNumberMsk)) & 0x1) << 7))

			if stm32.USB_DIEPINT_XFRC == reg&stm32.USB_DIEPINT_XFRC {
				d.glo.InEp(num).DIEPINT.Set(stm32.USB_DIEPINT_XFRC)
				d.dev.DIEPEMPMSK.ClearBits(1 << (num & descEndptAddrNumberMsk))
				// data in stage
				// d.endpointDataIn(txEndpoint(uint8(num)),
				// 	d.endpointInfo(txEndpoint(uint8(num))).data)
			}
			if stm32.USB_DIEPINT_TOC == reg&stm32.USB_DIEPINT_TOC {
				d.glo.InEp(num).DIEPINT.Set(stm32.USB_DIEPINT_TOC)
			}
			if stm32.USB_DIEPINT_INEPNE == reg&stm32.USB_DIEPINT_INEPNE {
				d.glo.InEp(num).DIEPINT.Set(stm32.USB_DIEPINT_INEPNE)
			}
			if stm32.USB_DIEPINT_EPDISD == reg&stm32.USB_DIEPINT_EPDISD {
				d.glo.InEp(num).DIEPINT.Set(stm32.USB_DIEPINT_EPDISD)
			}

			// drainTx := false
			if stm32.USB_DIEPINT_ITTXFE == reg&stm32.USB_DIEPINT_ITTXFE {
				d.glo.InEp(num).DIEPINT.Set(stm32.USB_DIEPINT_ITTXFE)
				// drainTx = true
			}
			if stm32.USB_DIEPINT_TXFE == reg&stm32.USB_DIEPINT_TXFE {
				// drainTx = true
			}

			// if drainTx {
			// 	// write empty Tx FIFO
			// 	for {
			// 		size := d.endpointTransmitRemain(uint8(num))
			// 		if !(0 < size && size <= d.endpointTransmitAvail(uint8(num))) {
			// 			break
			// 		}
			// 		d.endpointWrite(txEndpoint(uint8(num)),
			// 			d.endpointInfo(txEndpoint(uint8(num))).data, size)
			// 	}
			// 	if d.endpointTransmitRemain(uint8(num)) <= 0 {
			// 		// Disable Tx FIFO empty interrupt once all data has been transmitted
			// 		d.dev.DIEPEMPMSK.ClearBits(1 << num)
			// 	}
			// }
		}
	}

	// Bit 31 WKUPINT: Resume/remote wakeup detected interrupt
	//   Wakeup interrupt during suspend(L2) or LPM(L1) state.
	//     – During suspend(L2):
	//         In device mode, this interrupt is asserted when a resume is
	//         detected on the USB.
	//     – During LPM(L1):
	//         This interrupt is asserted for either host initiated resume or
	//         device initiated remote wakeup on USB.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_WKUINT) {
		d.dev.DCTL.ClearBits(stm32.USB_DCTL_RWUSIG)
		// TODO: resume
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_WKUINT)
	}

	// Bit 11 USBSUSP: USB suspend
	//   The core sets this bit to indicate that a suspend was detected on the
	//   USB. The core enters the suspended state when there is no activity on the
	//   data lines for an extended period of time.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_USBSUSP) {
		// TODO: suspend
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_USBSUSP)
	}

	// Bit 27 LPMINT: LPM interrupt
	//   In device mode, this interrupt is asserted when the device receives an
	//   LPM transaction and responds with a non-ERRORed response.
	//   This field is valid only if the LPMEN bit in OTG_GLPMCFG is set to 1.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_LPMINT) {
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_LPMINT)
		// TODO: suspend
	}

	// Bit 12 USBRST: USB reset
	// The core sets this bit to indicate that a reset is detected on the USB.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_USBRST) {

		d.dev.DCTL.ClearBits(stm32.USB_DCTL_RWUSIG)
		d.endpointFlush(descEndptAddrNumberMsk + 1) // all Tx FIFOs

		// Abort any ongoing transfers
		for i := 0; i < int(d.endpointCount()); i++ {
			d.glo.InEp(i).DIEPINT.Set(0xFB7F)
			d.glo.InEp(i).DIEPCTL.ClearBits(stm32.USB_DIEPCTL_STALL)
			d.glo.InEp(i).DIEPCTL.SetBits(stm32.USB_DIEPCTL_SNAK)
			d.glo.OutEp(i).DOEPINT.Set(0xFB7F)
			d.glo.OutEp(i).DOEPCTL.ClearBits(stm32.USB_DOEPCTL_STALL)
			d.glo.OutEp(i).DOEPCTL.SetBits(stm32.USB_DOEPCTL_SNAK)
		}

		// Unmask device endpoint interrupts
		d.dev.DAINTMSK.SetBits(0x00010001) // IN+OUT on control endpoint 0
		d.dev.DOEPMSK.SetBits(stm32.USB_DOEPMSK_STUPM | stm32.USB_DOEPMSK_XFRCM |
			stm32.USB_DOEPMSK_EPDM | stm32.USB_DOEPMSK_OTEPSPRM |
			stm32.USB_DOEPMSK_NAKM | stm32.USB_DOEPMSK_OTEPDM)
		d.dev.DIEPMSK.SetBits(stm32.USB_DIEPMSK_TOM | stm32.USB_DIEPMSK_XFRCM |
			stm32.USB_DIEPMSK_EPDM | stm32.USB_DIEPMSK_ITTXFEMSK |
			stm32.USB_DIEPMSK_INEPNEM)

		// Allocate RAM for Rx/Tx FIFOs
		d.resetEndpoints()

		// Set device address to default (0) during initialization.
		//
		// Address 0 is used for communication from the host to all devices after
		// reset and during initialization. The protocol is designed so that only
		// one device on the bus may be assigned address 0 at any given instant.
		// During enumeration, the host will perform a control transfer using the
		// SET_ADDRESS standard request, which will assign a bus-unique address to
		// our device, and after which time the device must never again respond to
		// requests at device address 0.
		d.setDeviceAddress(0)

		d.event(dcdEvent{id: dcdEventStatusReset})

		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_USBRST)
	}

	// Bit 13 ENUMDNE: Enumeration done
	//   The core sets this bit to indicate that speed enumeration is complete.
	//   The application must read the OTG_DSTS register to obtain the enumerated
	//   speed.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_ENUMDNE) {

		// Get the USB bus speed
		if d.dev.DSTS.HasBits(stm32.USB_DSTS_ENUMSPD) {
			d.speed = FullSpeed
		} else {
			d.speed = HighSpeed
		}

		// Set USB turnaround time
		d.glo.GUSBCFG.ReplaceBits(
			d.turnaroundTime(descHCLKFrequencyHz)<<stm32.USB_GUSBCFG_TRDT_Pos,
			stm32.USB_GUSBCFG_TRDT_Msk, 0)

		// Reset maximum packet size on IN (Tx) control endpoint 0. Reconfigured in
		// endpointEnable, which is called on event dcdEventPeripheralReady below.
		d.glo.InEp(0).DIEPCTL.ClearBits(stm32.USB_DIEPCTL_MPSIZ)

		// Clear global NAKs
		d.dev.DCTL.SetBits(stm32.USB_DCTL_CGINAK)

		d.event(dcdEvent{id: dcdEventPeripheralReady})

		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_ENUMDNE)
	}

	// Bit 3 SOF: Start of frame
	//   In device mode, the core sets this bit to indicate that an SOF token has
	//   been received on the USB. The application can read the OTG_DSTS register
	//   to get the current frame number.
	//   This interrupt is seen only when the core is operating in FS.
	//   Note:
	//     This register may return '1' if read immediately after power on reset.
	//     If the register bit reads '1' immediately after power on reset it does
	//     not indicate that an SOF has been sent (in case of host mode) or SOF
	//     has been received (in case of device mode). The read value of this
	//     interrupt is valid only after a valid connection between host and
	//     device is established. If the bit is set after power on reset the
	//     application can clear the bit.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_SOF) {
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_SOF)
	}

	// Bit 20 IISOIXFR: Incomplete isochronous IN transfer
	//   The core sets this interrupt to indicate that there is at least one
	//   isochronous IN endpoint on which the transfer is not completed in the
	//   current frame. This interrupt is asserted along with the End of periodic
	//   frame interrupt (EOPF) bit in this register.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_IISOIXFR) {
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_IISOIXFR)
	}

	// Bit 21 INCOMPISOOUT: Incomplete isochronous OUT transfer
	//   In device mode, the core sets this interrupt to indicate that there is at
	//   least one isochronous OUT endpoint on which the transfer is not completed
	//   in the current frame. This interrupt is asserted along with the End of
	//   periodic frame interrupt (EOPF) bit in this register.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_PXFR_INCOMPISOOUT) {
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_PXFR_INCOMPISOOUT)
	}

	// Bit 30 SRQINT: Session request/new session detected interrupt
	//   In device mode, this interrupt is asserted when VBUS is in the valid
	//   range for a B-peripheral device.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_SRQINT) {
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_SRQINT)
	}

	// Bit 2 OTGINT: OTG interrupt
	//   The core sets this bit to indicate an OTG protocol event. The application
	//   must read the OTG interrupt status (OTG_GOTGINT) register to determine
	//   the exact event that caused this interrupt. The application must clear
	//   the appropriate status bit in the OTG_GOTGINT register to clear this bit.
	if d.glo.GINTSTS.HasBits(stm32.USB_GINTSTS_OTGINT) {
		d.glo.GINTSTS.Set(d.glo.GINTSTS.Get() & stm32.USB_GINTSTS_OTGINT)
	}

	d.irq.Enable()
}

func (d *dhw) setDeviceAddress(addr uint16) {
	d.address = uint32(addr) & (stm32.USB_DCFG_DAD_Msk >> stm32.USB_DCFG_DAD_Pos)
	d.dev.DCFG.ClearBits(stm32.USB_DCFG_DAD_Msk)
	if addr > 0 {
		// Transition to "addressed" peripheral state if we received a non-default
		// (non-zero) address.
		d.state = dhwStateAddress
		d.dev.DCFG.SetBits(d.address << stm32.USB_DCFG_DAD_Pos)
	}
}

func (d *dhw) setDeviceSpeed(speed Speed) {
	d.dev.DCFG.ClearBits(stm32.USB_DCFG_DSPD_Msk)
	switch d.port {
	// USB OTG port 1 supports both high- and full-speed modes
	case 0:
		switch speed {
		// Configured as either full-speed (FS) or low-speed (LS)
		case LowSpeed, FullSpeed:
			d.dev.DCFG.SetBits(0x1 << stm32.USB_DCFG_DSPD_Pos)
		// Configured as either high-speed (HS) or super-speed (SS [unsupported])
		case HighSpeed, SuperSpeed, DualSuperSpeed:
			// leave speed bitfield cleared (0)
		}
	// USB OTG port 2 supports full-speed (FS) mode ONLY
	case 1:
		d.dev.DCFG.SetBits(0x3 << stm32.USB_DCFG_DSPD_Pos)
	}
}

func (d *dhw) turnaroundTime(hclkFreq uint32) uint32 {
	switch d.speed {
	case FullSpeed:
		if hclkFreq < 14200000 {
			return 0xF //             HCLK < 14.2 MHz
		} else if hclkFreq < 15000000 {
			return 0xF // 14.2 MHz <= HCLK < 15.0 MHz
		} else if hclkFreq < 16000000 {
			return 0xE // 15.0 MHz <= HCLK < 16.0 MHz
		} else if hclkFreq < 17200000 {
			return 0xD // 16.0 MHz <= HCLK < 17.2 MHz
		} else if hclkFreq < 18500000 {
			return 0xC // 17.2 MHz <= HCLK < 18.5 MHz
		} else if hclkFreq < 20000000 {
			return 0xB // 18.5 MHz <= HCLK < 20.0 MHz
		} else if hclkFreq < 21800000 {
			return 0xA // 20.0 MHz <= HCLK < 21.8 MHz
		} else if hclkFreq < 24000000 {
			return 0x9 // 21.8 MHz <= HCLK < 24.0 MHz
		} else if hclkFreq < 27700000 {
			return 0x8 // 24.0 MHz <= HCLK < 27.7 MHz
		} else if hclkFreq < 32000000 {
			return 0x7 // 27.7 MHz <= HCLK < 32.0 MHz
		} else {
			return 0x6 // 32.0 MHz <= HCLK
		}
	default:
		return 0x9
	}
}

// =============================================================================
//  Control Endpoint 0
// =============================================================================

// controlBuffer returns a pointer into the control endpoint 0 transmit/receive
// buffer at the given index offset. Offset is treated as a circular index, such
// that negative values offset from the end of the buffer (wrap on underflow),
// and positive values offset from the beginning (wrap on overflow).
func (d *dhw) controlBuffer(offset int) uintptr {

	switch d.cc.id {
	// CDC-ACM (single)
	case classDeviceCDCACM:
		acm := &descCDCACM[d.cc.config-1]
		return uintptr(unsafe.Pointer(&acm.cx[wrap(offset, len(acm.cx))]))
	// HID
	case classDeviceHID:
		hid := &descHID[d.cc.config-1]
		return uintptr(unsafe.Pointer(&hid.cx[wrap(offset, len(hid.cx))]))
	// Unhandled device class
	default:
	}

	return 0
}

// controlStall stalls a transfer on control endpoint 0. To stall a transfer on
// any other endpoint, use method endpointStall().
func (d *dhw) controlStall() {
	d.endpointStall(rxEndpoint(0))
	d.endpointStall(txEndpoint(0))
}

// controlReceive receives (Rx, OUT) data on control endpoint 0.
func (d *dhw) controlReceive(data uintptr, size uint32, notify bool) {

	d.endpointInfo(rxEndpoint(0)).data = data
	d.endpointInfo(rxEndpoint(0)).len = size
	d.endpointInfo(rxEndpoint(0)).count = 0

	ep := d.glo.OutEp(int(rxEndpoint(0)))
	ep.DOEPTSIZ.ClearBits(stm32.USB_DOEPTSIZ_PKTCNT | stm32.USB_DOEPTSIZ_XFRSIZ)
	if size > 0 {
		size = descEndptMaxPktSize
	}
	ep.DOEPTSIZ.Set((1 << stm32.USB_DOEPTSIZ_PKTCNT_Pos) |
		(size << stm32.USB_DOEPTSIZ_XFRSIZ_Pos) |
		stm32.USB_DOEPTSIZ_STUPCNT)
	ep.DOEPCTL.SetBits(stm32.USB_DOEPCTL_CNAK | stm32.USB_DOEPCTL_EPENA)
}

// controlTransmit transmits (Tx, IN) data on control endpoint 0.
func (d *dhw) controlTransmit(data uintptr, size uint32, notify bool) {

	d.endpointInfo(txEndpoint(0)).data = data
	d.endpointInfo(txEndpoint(0)).len = size
	d.endpointInfo(txEndpoint(0)).count = 0

	ep := d.glo.InEp(int(txEndpoint(0)))
	if 0 == size {
		// zero-length packet (1 packet, 0 bytes)
		ep.DIEPTSIZ.Set(1 << stm32.USB_DIEPTSIZ_PKTCNT_Pos)
	} else {
		pkts := uint32(1)
		if size > descEndptMaxPktSize {
			pkts += size / descEndptMaxPktSize
			size = descEndptMaxPktSize
		}
		ep.DIEPTSIZ.Set(0)
		ep.DIEPTSIZ.Set((pkts << stm32.USB_DIEPTSIZ_PKTCNT_Pos) |
			(size << stm32.USB_DIEPTSIZ_XFRSIZ_Pos))
	}
	ep.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_STALL)
	ep.DIEPCTL.SetBits(stm32.USB_DIEPCTL_CNAK | stm32.USB_DIEPCTL_EPENA)
	if size > 0 {
		d.dev.DIEPEMPMSK.SetBits(1)
	}

	// write empty Tx FIFO
	for {
		size := d.endpointTransmitRemain(0)
		if !(0 < size && size <= d.endpointTransmitAvail(0)) {
			break
		}
		d.endpointWrite(txEndpoint(0), d.endpointInfo(txEndpoint(0)).data, size)
	}
	if d.endpointTransmitRemain(0) <= 0 {
		// Disable Tx FIFO empty interrupt once all data has been transmitted
		d.dev.DIEPEMPMSK.ClearBits(1)
	}
}

// =============================================================================
//  Endpoint Descriptor
// =============================================================================

// dhwEndpoint defines a USB standard endpoint, used as the general channel of
// communication between host and device.
type dhwEndpoint struct {
	num   uint8 // Endpoint number
	isTx  bool  // Endpoint direction (is IN endpoint)
	stall bool  // Endpoint stall condition
	kind  uint8 // Endpoint type
	// parity bool    // IFrame parity
	fifo  uint8   // Transmission FIFO number
	size  uint32  // Endpoint max packet size (bytes)
	data  uintptr // Pointer to transfer buffer
	len   uint32  // Current transfer length
	count uint32  // Partial transfer length in case of multi-packet transfer
}

// endpointAlloc allocates FIFO buffer space and initializes the endpoint state
// buffer for the given endpoint.
func (d *dhw) endpointAlloc(endpoint, kind uint8, size uint32) {

	var fifo uint8
	var isTx bool
	num, _ := unpackEndpoint(endpoint)

	switch endpoint {
	case rxEndpoint(endpoint):
		// Only modify Rx FIFO for control endpoint 0, which must account for space
		// required to receive all Rx endpoints for the current device class
		// configuration. Ignore all other Rx (OUT) endpoints given.
		if 0 == num {
			words := size/4 + 1
			d.glo.GRXFSIZ.Set(words)
		}
	case txEndpoint(endpoint):
		txOffset := d.glo.GRXFSIZ.Get()
		words := size/4 + 1
		fifo = num
		isTx = true
		if 0 == num {
			d.glo.DIEPTXF0.Set((words << 16) | txOffset)
		} else {
			txOffset += d.glo.DIEPTXF0.Get() >> 16
			for i := uint8(0); i < num-1; i++ {
				txOffset += d.glo.DIEPTXF[i].Get() >> 16
			}
			// Double the Tx buffer size (<< 1) for better performance, according to
			// reference manual:
			//
			//  | More space allocated in the transmit IN endpoint FIFO results in
			//  | better performance on the USB.
			//
			d.glo.DIEPTXF[num-1].Set(((words << 1) << 16) | txOffset)
		}
	}

	// Initialize endpoint state info buffer
	*d.endpointInfo(endpoint) = dhwEndpoint{
		num:   num,   // Endpoint number
		isTx:  isTx,  // Endpoint direction (is IN endpoint)
		stall: false, // Endpoint stall condition
		kind:  kind,  // Endpoint type
		fifo:  fifo,  // Transmission FIFO number
		size:  size,  // Endpoint max packet size (bytes)
		data:  0,     // Pointer to transfer buffer
		len:   0,     // Current transfer length
		count: 0,     // Partial transfer length in case of multi-packet transfer
	}
}

// endpointCount returns the number of endpoints used by the receiver's device
// class configuration.
func (d *dhw) endpointCount() (count uint32) {
	switch d.cc.id {
	// CDC-ACM (single)
	case classDeviceCDCACM:
		return descCDCACMEndpointCount
	// HID
	case classDeviceHID:
		return descHIDEndpointCount
	default:
		// Unhandled device class
	}
	return 0
}

// endpointInfo returns the buffered dhwEndpoint, containing status and transfer
// details, for the given endpoint address.
func (d *dhw) endpointInfo(endpoint uint8) *dhwEndpoint {
	idx := endpointIndex(endpoint)
	switch d.cc.id {
	// CDC-ACM (single)
	case classDeviceCDCACM:
		acm := &descCDCACM[d.cc.config-1]
		if int(idx) < len(acm.ep) {
			return &acm.ep[idx]
		}
	// HID
	case classDeviceHID:
		//hid := &descHID[d.cc.config-1]
		//if int(idx) < len(hid.ep) {
		//	return &hid.ep[idx]
		//}
	default:
		// Unhandled device class
	}
	return nil
}

// endpointEnable configures the given endpoint's type and maximum packet size,
// unmasks its core interrupt, and sets its core endpoint enabled flag.
//
// The given endpoint is encoded using the uint8 bitmap defined per USB spec
// (i.e., direction is high bit, address is low bits), which is constructed
// using either function rxEndpoint or txEndpoint.
//
// If the control flag is true, then the given endpoint number is ignored and
// endpoint 0 is assumed. Note that in this case, you should call endpointEnable
// only ONE time, as it implicitly enables both Rx and Tx on control endpoint 0.
//
// The config parameter is unused on STM32 devices.
func (d *dhw) endpointEnable(endpoint uint8, control bool, config uint32) {

	if control {
		// Prepare endpoint 0 for setup packet transactions
		d.glo.OutEp(0).DOEPTSIZ.Set(
			(stm32.USB_DOEPTSIZ_PKTCNT & (1 << stm32.USB_DOEPTSIZ_PKTCNT_Pos)) |
				((3 * 8) << stm32.USB_DOEPTSIZ_XFRSIZ_Pos) | stm32.USB_DOEPTSIZ_STUPCNT)

		// Enable both Rx and Tx for control endpoint 0. These are recursive calls
		// to the base case (control == false).
		d.endpointEnable(rxEndpoint(0), false, 0)
		d.endpointEnable(txEndpoint(0), false, 0)

	} else {
		num, _ := unpackEndpoint(endpoint)
		info := d.endpointInfo(endpoint)

		switch endpoint {
		case rxEndpoint(endpoint): // OUT
			ep := d.glo.OutEp(int(num))
			d.dev.DAINTMSK.SetBits(stm32.USB_DAINTMSK_OEPM & ((1 << num) << 16))
			if !ep.DOEPCTL.HasBits(stm32.USB_DOEPCTL_USBAEP) {
				ep.DOEPCTL.SetBits((info.size & stm32.USB_DOEPCTL_MPSIZ) |
					(uint32(info.kind) << stm32.USB_DOEPCTL_EPTYP_Pos) |
					stm32.USB_DIEPCTL_SD0PID_SEVNFRM | stm32.USB_DOEPCTL_USBAEP)
			}
		case txEndpoint(endpoint): // IN
			ep := d.glo.InEp(int(num))
			d.dev.DAINTMSK.SetBits(stm32.USB_DAINTMSK_IEPM & (1 << num))
			if !ep.DIEPCTL.HasBits(stm32.USB_DIEPCTL_USBAEP) {
				ep.DIEPCTL.SetBits((info.size & stm32.USB_DIEPCTL_MPSIZ) |
					(uint32(info.kind) << stm32.USB_DIEPCTL_EPTYP_Pos) |
					(uint32(num) << stm32.USB_DIEPCTL_TXFNUM_Pos) |
					stm32.USB_DIEPCTL_SD0PID_SEVNFRM | stm32.USB_DIEPCTL_USBAEP)
			}
		}
	}
}

// endpointDisable configures the given endpoint by masking its core interrupt,
// clearing its core endpoint enabled flag, clearing its type and maximum packet
// size, and setting its endpoint disconnect and NAK bits.
//
// The given endpoint is encoded using the uint8 bitmap defined per USB spec
// (i.e., direction is high bit, address is low bits), which is constructed
// using either function rxEndpoint or txEndpoint.
func (d *dhw) endpointDisable(endpoint uint8) {

	num, _ := unpackEndpoint(endpoint)

	switch endpoint {
	case rxEndpoint(endpoint): // OUT
		ep := d.glo.OutEp(int(num))
		if ep.DOEPCTL.HasBits(stm32.USB_DOEPCTL_EPENA) {
			ep.DOEPCTL.SetBits(stm32.USB_DOEPCTL_SNAK | stm32.USB_DOEPCTL_EPDIS)
		}
		d.dev.DEACHMSK.ClearBits(stm32.USB_DAINTMSK_OEPM & ((1 << num) << 16))
		d.dev.DAINTMSK.ClearBits(stm32.USB_DAINTMSK_OEPM & ((1 << num) << 16))
		ep.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_USBAEP | stm32.USB_DOEPCTL_MPSIZ |
			stm32.USB_DOEPCTL_SD0PID_SEVNFRM | stm32.USB_DOEPCTL_EPTYP)

	case txEndpoint(endpoint): // IN
		ep := d.glo.InEp(int(num))
		if ep.DIEPCTL.HasBits(stm32.USB_DIEPCTL_EPENA) {
			ep.DIEPCTL.SetBits(stm32.USB_DIEPCTL_SNAK | stm32.USB_DIEPCTL_EPDIS)
		}
		d.dev.DEACHMSK.ClearBits(stm32.USB_DAINTMSK_IEPM & (1 << num))
		d.dev.DAINTMSK.ClearBits(stm32.USB_DAINTMSK_IEPM & (1 << num))
		ep.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_USBAEP |
			stm32.USB_DIEPCTL_MPSIZ | stm32.USB_DIEPCTL_TXFNUM |
			stm32.USB_DIEPCTL_SD0PID_SEVNFRM | stm32.USB_DIEPCTL_EPTYP)
	}
}

// endpointFlush flushes the given endpoint FIFO. If the given endpoint equals
// 0x10 (16), then all Tx FIFOs are flushed.
func (d *dhw) endpointFlush(endpoint uint8) {
	num, _ := unpackEndpoint(endpoint)
	switch endpoint {
	case 0x10:
		num = 0x10
		fallthrough
	case txEndpoint(endpoint):
		d.glo.GRSTCTL.Set(stm32.USB_GRSTCTL_TXFFLSH |
			(uint32(num) << stm32.USB_GRSTCTL_TXFNUM_Pos))
		for d.glo.GRSTCTL.HasBits(stm32.USB_GRSTCTL_TXFFLSH) {
		}
	case rxEndpoint(endpoint):
		d.glo.GRSTCTL.Set(stm32.USB_GRSTCTL_RXFFLSH)
		for d.glo.GRSTCTL.HasBits(stm32.USB_GRSTCTL_RXFFLSH) {
		}
	}
}

// endpointStall stalls a transfer on the given endpoint.
func (d *dhw) endpointStall(endpoint uint8) {

	d.endpointInfo(endpoint).stall = true
	num, _ := unpackEndpoint(endpoint)

	switch endpoint {
	case rxEndpoint(endpoint): // OUT
		ep := d.glo.OutEp(int(num))
		if !ep.DOEPCTL.HasBits(stm32.USB_DOEPCTL_EPENA) && 0 != num {
			ep.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_EPDIS)
		}
		ep.DOEPCTL.SetBits(stm32.USB_DOEPCTL_STALL)

	case txEndpoint(endpoint): // IN
		ep := d.glo.InEp(int(num))
		if !ep.DIEPCTL.HasBits(stm32.USB_DIEPCTL_EPENA) && 0 != num {
			ep.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_EPDIS)
		}
		ep.DIEPCTL.SetBits(stm32.USB_DIEPCTL_STALL)
	}

	if 0 == endpoint {
		// Prepare endpoint 0 for setup packet transactions
		d.glo.OutEp(0).DOEPTSIZ.Set(
			(stm32.USB_DOEPTSIZ_PKTCNT & (1 << stm32.USB_DOEPTSIZ_PKTCNT_Pos)) |
				((3 * 8) << stm32.USB_DOEPTSIZ_XFRSIZ_Pos) | stm32.USB_DOEPTSIZ_STUPCNT)
	}
}

// endpointUnstall unstalls a transfer on the given endpoint.
func (d *dhw) endpointUnstall(endpoint uint8) {

	var isBulkOrInt bool

	d.endpointInfo(endpoint).stall = false
	switch d.endpointInfo(endpoint).kind {
	case descEndptTypeBulk, descEndptTypeInterrupt:
		isBulkOrInt = true
	}
	num, _ := unpackEndpoint(endpoint)

	switch endpoint {
	case rxEndpoint(endpoint): // OUT
		ep := d.glo.OutEp(int(num))
		ep.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_STALL)
		if isBulkOrInt {
			ep.DOEPCTL.SetBits(stm32.USB_DOEPCTL_SD0PID_SEVNFRM)
		}

	case txEndpoint(endpoint): // IN
		ep := d.glo.InEp(int(num))
		ep.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_STALL)
		if isBulkOrInt {
			ep.DIEPCTL.SetBits(stm32.USB_DIEPCTL_SD0PID_SEVNFRM)
		}
	}
}

func (d *dhw) endpointStatus(endpoint uint8) uint16 {
	status := uint16(0)
	switch endpoint {
	case rxEndpoint(endpoint):
	case txEndpoint(endpoint):
	}
	return status
}

func (d *dhw) endpointClearFeature(endpoint uint8) {
	switch endpoint {
	case rxEndpoint(endpoint):
	case txEndpoint(endpoint):
	}
}

func (d *dhw) endpointSetFeature(endpoint uint8) {
	switch endpoint {
	case rxEndpoint(endpoint):
	case txEndpoint(endpoint):
	}
}

/**
 * @brief  USBD_LL_SetupStage
 *         Handle the setup stage
 * @param  pdev: device instance
 * @retval status
 */
// func (d *dhw) endpointSetup(setup dcdSetup) 	{

// 		d.stage = dcdStageSetup;

// 		pdev->ep0_data_len = pdev->request.wLength;

// 		switch (pdev->request.bmRequest & 0x1FU)
// 		{
// 			case USB_REQ_RECIPIENT_DEVICE:
// 				ret = USBD_StdDevReq(pdev, &pdev->request);
// 				break;

// 			case USB_REQ_RECIPIENT_INTERFACE:
// 				ret = USBD_StdItfReq(pdev, &pdev->request);
// 				break;

// 			case USB_REQ_RECIPIENT_ENDPOINT:
// 				ret = USBD_StdEPReq(pdev, &pdev->request);
// 				break;

// 			default:
// 				ret = USBD_LL_StallEP(pdev, (pdev->request.bmRequest & 0x80U));
// 				break;
// 		}

// 		return ret;
// 	}

// func (d *dhw) endpointDataOut(endpoint uint8) {
// 	num, _ := unpackEndpoint(endpoint)
// 	if 0 == num {
// 		info := d.endpointInfo(endpoint)
// 		// if d.stage == dcdStageDataOut {
// 		if info.len > info.size {
// 			info.len -= info.size
// 			size := info.len
// 			if size > info.size {
// 				size = info.size
// 			}
// 			d.controlReceive(info.data, size, false)
// 		} else {
// 			d.stage = dcdStageStatusIn
// 			d.controlTransmit(0, 0, false)
// 		}
// 		// }
// 	}
// }

// func (d *dhw) endpointDataIn(endpoint uint8, data uintptr) {
// 	num, _ := unpackEndpoint(endpoint)
// 	if 0 == num {
// 		info := d.endpointInfo(endpoint)
// 		// if d.stage == dcdStageDataIn {
// 		if info.len > info.size {
// 			info.len -= info.size
// 			d.controlTransmit(info.data, info.len, false)
// 			d.stage = dcdStageStatusOut
// 			d.controlReceive(0, 0, false)
// 		} else {
// 			// last packet is MPS multiple, so send ZLP packet
// 			if (info.size == info.len) && (info.count+info.len >= info.size) {
// 				d.controlTransmit(info.data, info.size, false)
// 				d.stage = dcdStageStatusOut
// 				d.controlReceive(0, 0, false)
// 			} else {
// 				d.endpointStall(txEndpoint(0))
// 				d.stage = dcdStageStatusOut
// 				d.controlReceive(0, 0, false)
// 			}
// 		}
// 		// }
// 	}
// }

// endpointComplete handles transfer completion of a data endpoint.
func (d *dhw) endpointComplete(endpoint uint8) {
}

// endpointConfigure configures the given endpoint for transfer.
func (d *dhw) endpointConfigure(endpoint uint8, packetSize uint16, zlp bool) {

}

// endpointReceiveCount returns the number of bytes received from the given
// endpoint's Rx FIFO.
func (d *dhw) endpointReceiveCount(endpoint uint8) uint32 {
	return d.endpointInfo(rxEndpoint(endpoint)).count
}

// endpointTransmitCount returns the number of bytes transmitted on the given
// endpoint's Tx FIFO.
func (d *dhw) endpointTransmitCount(endpoint uint8) uint32 {
	return d.endpointInfo(txEndpoint(endpoint)).count
}

// endpointTransmitAvail returns the number of bytes available for transmission
// in the given endpoint's Tx FIFO.
func (d *dhw) endpointTransmitAvail(endpoint uint8) uint32 {
	num, _ := unpackEndpoint(txEndpoint(endpoint))
	return (d.glo.InEp(int(num)).DTXFSTS.Get() & stm32.USB_DTXFSTS_INEPTFSAV) << 3
}

// endpointTransmitRemain returns the number of bytes remaining for transmission
// on the given endpoint's Tx FIFO. If the number of bytes remaining is greater
// than the given endpoint's maximum packet size, returns maximum packet size.
func (d *dhw) endpointTransmitRemain(endpoint uint8) uint32 {
	info := d.endpointInfo(txEndpoint(endpoint))
	size := info.len - info.count
	if size > info.size {
		size = info.size
	}
	return size
}

// endpointRead reads a packet from the given endpoint's Rx FIFO.
func (d *dhw) endpointRead(endpoint uint8, data uintptr, size uint32) {
	fifo := d.glo.EpFifo(int(d.endpointInfo(endpoint).fifo))
	end := data + uintptr(size)
	for data+4 < end {
		*(*uint32)(unsafe.Pointer(data)) = fifo.Get()
		data += 4
	}
	if data < end {
		var u [4]uint8
		*(*uint32)(unsafe.Pointer(&u[0])) = fifo.Get()
		for i := uintptr(0); data+i < end; i++ {
			*(*uint8)(unsafe.Pointer(data + i)) = u[i]
		}
	}
	d.endpointInfo(endpoint).data = end
	d.endpointInfo(endpoint).count += size
}

// endpointWrite writes a packet to the given endpoint's Tx FIFO.
func (d *dhw) endpointWrite(endpoint uint8, data uintptr, size uint32) {
	fifo := d.glo.EpFifo(int(d.endpointInfo(endpoint).fifo))
	for i := uint32(0); i < (size+3)>>2; i++ {
		fifo.Set(*(*uint32)(unsafe.Pointer(data)))
		data += 4
	}
	d.endpointInfo(endpoint).data += uintptr(size)
	d.endpointInfo(endpoint).count += size
}

// =============================================================================
//  [CDC-ACM] Serial UART (Virtual COM Port)
// =============================================================================

func (d *dhw) uartConfigure() {

	d.state = dhwStateConfigure
	acm := &descCDCACM[d.cc.config-1]

	switch d.speed {
	case LowSpeed, FullSpeed:
		acm.rxSize = descCDCACMDataRxFSPacketSize
		acm.txSize = descCDCACMDataTxFSPacketSize
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		acm.rxSize = descCDCACMDataRxHSPacketSize
		acm.txSize = descCDCACMDataTxHSPacketSize
	}

	d.endpointEnable(txEndpoint(descCDCACMEndpointStatus), false, 0)
	d.endpointEnable(rxEndpoint(descCDCACMEndpointDataRx), false, 0)
	d.endpointEnable(txEndpoint(descCDCACMEndpointDataTx), false, 0)
}

func (d *dhw) uartSetLineState(dtr, rts bool) {
	// TBD: does the PHY need to handle on iMXRT1062 (e.g., Teensyduino Loader)?
}

func (d *dhw) uartSetLineCoding(coding descCDCACMLineCoding) {
	if 134 == coding.baud {
		d.enableSOF(true, descCDCACMInterfaceCount)
	}
}

func (d *dhw) uartReceive(endpoint uint8) {
	// acm := &descCDCACM[d.cc.config-1]
	// num := uint16(endpoint) & descEndptAddrNumberMsk
}

// func (d *dhw) uartNotify(transfer *dhwTransfer) {
// 	// acm := &descCDCACM[d.cc.config-1]
// }

// uartFlush discards all buffered input (Rx) data.
func (d *dhw) uartFlush() {
	// acm := &descCDCACM[d.cc.config-1]
}

func (d *dhw) uartAvailable() int {
	return 0
}

func (d *dhw) uartPeek() (uint8, bool) {
	// acm := &descCDCACM[d.cc.config-1]
	return 0, true
}

func (d *dhw) uartReadByte() (uint8, bool) {
	b := []uint8{0}
	ok := d.uartRead(b) > 0
	return b[0], ok
}

func (d *dhw) uartRead(data []uint8) int {
	// acm := &descCDCACM[d.cc.config-1]
	read := uint16(0)
	return int(read)
}

func (d *dhw) uartWriteByte(c uint8) bool {
	return 1 == d.uartWrite([]uint8{c})
}

func (d *dhw) uartWrite(data []uint8) int {
	// acm := &descCDCACM[d.cc.config-1]
	sent := 0
	return sent
}

func (d *dhw) uartSync() {
}

// =============================================================================
//  [HID] Serial
// =============================================================================

func (d *dhw) serialConfigure() {

	d.state = dhwStateConfigure
	hid := &descHID[d.cc.config-1]

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.rxSerialSize = descHIDSerialRxHSPacketSize
		hid.txSerialSize = descHIDSerialTxHSPacketSize
	case LowSpeed, FullSpeed:
		hid.rxSerialSize = descHIDSerialRxFSPacketSize
		hid.txSerialSize = descHIDSerialTxFSPacketSize
	}

	// Rx and Tx are on same endpoint
	d.endpointEnable(rxEndpoint(descHIDEndpointSerialRx), false, 0)
	d.endpointEnable(txEndpoint(descHIDEndpointSerialTx), false, 0)
}

func (d *dhw) serialReceive(endpoint uint8) {
	// hid := &descHID[d.cc.config-1]
	// num := uint16(endpoint) & descEndptAddrNumberMsk
}

func (d *dhw) serialTransmit() {
	// hid := &descHID[d.cc.config-1]
}

// func (d *dhw) serialNotify(transfer *dhwTransfer) {
// 	// hid := &descHID[d.cc.config-1]
// }

// serialFlush discards all buffered input (Rx) data.
func (d *dhw) serialFlush() {
	// hid := &descHID[d.cc.config-1]
}

func (d *dhw) serialSync() {
	// hid := &descHID[d.cc.config-1]
}

// =============================================================================
//  [HID] Keyboard
// =============================================================================

func (d *dhw) keyboard() *Keyboard { return descHID[d.cc.config-1].keyboard }

func (d *dhw) keyboardConfigure() {

	d.state = dhwStateConfigure
	hid := &descHID[d.cc.config-1]

	// Initialize keyboard
	hid.keyboard.configure(d.dcd, hid)

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.txKeyboardSize = descHIDKeyboardTxHSPacketSize
	case LowSpeed, FullSpeed:
		hid.txKeyboardSize = descHIDKeyboardTxFSPacketSize
	}

	d.endpointEnable(txEndpoint(descHIDEndpointKeyboard), false, 0)
	d.endpointEnable(txEndpoint(descHIDEndpointMediaKey), false, 0)
}

func (d *dhw) keyboardSendKeys(consumer bool) bool {
	// hid := &descHID[d.cc.config-1]
	return true
}

func (d *dhw) keyboardWrite(endpoint uint8, data []uint8) bool {
	// hid := &descHID[d.cc.config-1]
	return true
}

// =============================================================================
//  [HID] Mouse
// =============================================================================

func (d *dhw) mouseConfigure() {

	d.state = dhwStateConfigure
	hid := &descHID[d.cc.config-1]

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.txMouseSize = descHIDMouseTxHSPacketSize
	case LowSpeed, FullSpeed:
		hid.txMouseSize = descHIDMouseTxFSPacketSize
	}

	d.endpointEnable(txEndpoint(descHIDEndpointMouse), false, 0)
}

// =============================================================================
//  [HID] Joystick
// =============================================================================

func (d *dhw) joystickConfigure() {

	d.state = dhwStateConfigure
	hid := &descHID[d.cc.config-1]

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.txJoystickSize = descHIDJoystickTxHSPacketSize
	case LowSpeed, FullSpeed:
		hid.txJoystickSize = descHIDJoystickTxFSPacketSize
	}

	d.endpointEnable(txEndpoint(descHIDEndpointJoystick), false, 0)
}
