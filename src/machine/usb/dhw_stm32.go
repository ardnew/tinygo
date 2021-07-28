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
const dhwInterruptPriority = 0x10

// dhwState defines the USB-specified standard device states, which reflects the
// device's enumeration status and bus availability.
type dhwState int

const (
	dhwStateInit      dhwState = iota // Power-on/reset state, not ready to communicate.
	dhwStateError                     // Initialization error, cannot/will not enumerate.
	dhwStateDefault                   // Initialized, ready for control transfers.
	dhwStateAddress                   // Device address assigned by host, no device class.
	dhwStateConfigure                 // Device class configuration complete. READY state.
	dhwStateLowPower                  // Device entered low-power suspend/sleep mode.
)

// dhw implements USB device controller hardware abstraction for iMXRT1062.
type dhw struct {
	*dcd // USB device controller driver

	glo *stm32.USB_GLOBAL_Type // USB global registers
	dev *stm32.USB_DEVICE_Type // USB device registers
	pcc *volatile.Register32   // USB PWR/CLKCTL register

	irq interrupt.Interrupt // USB system interrupt

	tim *dhwTimer // USB timer interrupt

	state dhwState // State of the dhw's USB state machine
	stage dcdStage // USB transmission stage

	acm *descCDCACMClass

	speed Speed // USB bus speed or transfer rate

	controlReply [8]uint8
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

	dhwInstance[instance].acm = &descCDCACM[0]

	return &dhwInstance[instance]
}

func runBootloader() {}

// cycleCount uses the ARM debug cycle counter (DWT) available on ARM CoreSight
// devices (enabled in runtime_stm32h7x5.go, function initSysTick) to return the
// number of CPU cycles since boot.
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

	// TODO: Enable internal DMA
	//d.glo.GAHBCFG.SetBits(stm32.USB_GAHBCFG_DMAEN)

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

	// Enable/disable VBUS sensing
	const enableVbusSense = true
	if enableVbusSense {
		d.glo.GCCFG.SetBits(stm32.USB_GCCFG_VBDEN)
	} else {
		d.glo.GCCFG.ClearBits(stm32.USB_GCCFG_VBDEN)
		d.glo.GOTGCTL.SetBits(stm32.USB_GOTGCTL_BVALOEN |
			stm32.USB_GOTGCTL_BVALOVAL)
	}

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
	// d.glo.GAHBCFG.SetBits(stm32.USB_GAHBCFG_TXFELVL)

	d.glo.GINTMSK.Set(0)          // Disable all internal interrupts
	d.glo.GINTSTS.Set(0xFFFFFFFF) // Clear all internal interrupts

	// Unmask core interrupts
	d.glo.GINTMSK.SetBits(
		stm32.USB_GINTMSK_WUIM | //                  31: Resume/remote wakeup detected
			// stm32.USB_GINTMSK_SRQIM | //            30: Session request/new session detected
			// stm32.USB_GINTMSK_DISCINT | //          29: Disconnect detected
			// stm32.USB_GINTMSK_CIDSCHGM | //         28: Connector ID status change
			// stm32.USB_GINTMSK_LPMINTM | //          27: Low-power mode
			// stm32.USB_GINTMSK_PTXFEM |  //          26: Periodic Tx FIFO empty
			// stm32.USB_GINTMSK_HCIM |  //            25: Host channels
			// stm32.USB_GINTMSK_PRTIM |  //           24: Host port
			// stm32.USB_GINTMSK_RSTDEM | //           23: Reset detected
			// stm32.USB_GINTMSK_FSUSPM | //           22: Data fetch suspended
			// stm32.USB_GINTMSK_PXFRM_IISOOXFRM | //  21: Incomplete periodic transfer,
			//                                               Incomplete isochronous OUT transfer
			// stm32.USB_GINTMSK_IISOIXFRM | //        20: Incomplete isochronous IN transfer
			stm32.USB_GINTMSK_OEPINT | //              19: OUT endpoints
			stm32.USB_GINTMSK_IEPINT | //              18: IN endpoints
			//                                           - ( Reserved )
			//                                           - ( Reserved )
			// stm32.USB_GINTMSK_EOPFM | //            15: End of periodic frame
			// stm32.USB_GINTMSK_ISOODRPM | //         14: Isochronous OUT packet dropped
			stm32.USB_GINTMSK_ENUMDNEM | //            13: Enumeration done
			stm32.USB_GINTMSK_USBRST | //              12: USB reset
			stm32.USB_GINTMSK_USBSUSPM | //            11: USB suspend
			// stm32.USB_GINTMSK_ESUSPM | //           10: Early suspend
			//                                           - ( Reserved )
			//                                           - ( Reserved )
			// stm32.USB_GINTMSK_GONAKEFFM | //         7: Global OUT NAK effective
			// stm32.USB_GINTMSK_GINAKEFFM | //         6: Global non-periodic IN NAK effective
			// stm32.USB_GINTMSK_NPTXFEM | //           5: Non-periodic Tx FIFO empty
			stm32.USB_GINTMSK_RXFLVLM | //              4: Rx FIFO non-empty
			// stm32.USB_GINTMSK_SOFM | //              3: Start-of-frame
			stm32.USB_GINTMSK_OTGINT, //                2: OTG interrupt
		// stm32.USB_GINTMSK_MMISM //                 1: Mode mismatch
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
		// Control endpoint 0 (Rx FIFO)
		d.endpointAlloc(
			rxEndpoint(descCDCACMEndpointCtrl), descEndptTypeControl,
			descCDCACMRxFIFOSize, descEndptMaxPktSize)
		// Control endpoint 0 (Tx FIFO 0)
		d.endpointAlloc(
			txEndpoint(descCDCACMEndpointCtrl), descEndptTypeControl,
			descCDCACMTxFIFOSize, descEndptMaxPktSize)
		// Status interrupt endpoint 1 (Tx FIFO 1)
		d.endpointAlloc(
			txEndpoint(descCDCACMEndpointStatus), descEndptTypeInterrupt,
			descCDCACMTxFIFOSize, descCDCACMStatusPacketSize)
		// Data Rx bulk endpoint 2 (Rx FIFO)
		d.endpointAlloc(
			rxEndpoint(descCDCACMEndpointDataRx), descEndptTypeBulk,
			descCDCACMTxFIFOSize, descCDCACMDataRxPacketSize)
		// Data Tx bulk endpoint 3 (Tx FIFO 3)
		d.endpointAlloc(
			txEndpoint(descCDCACMEndpointDataTx), descEndptTypeBulk,
			descCDCACMTxFIFOSize, descCDCACMDataTxPacketSize)

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
		d.interruptEnable(true)
	} else {
		// Disable USB interrupts
		d.interruptEnable(false)
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
func (d *dhw) enableSOF(enable bool, iface uint8) {}

// interrupt handles the USB hardware interrupt events and notifies the device
// controller driver using a common "virtual interrupt" code.
func (d *dhw) interrupt() {
	d.interruptEnable(false)

	// Bit 1 MMIS: Mode mismatch interrupt
	//   The core sets this bit when the application is trying to access:
	//     – A host mode register, when the core is operating in device mode.
	//     – A device mode register, when the core is operating in host mode.
	//   The register access is completed on AHB with result OK, but is ignored
	//   by the core internally and does not affect the operation of the core.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_MMIS != 0 {
		// Clear interrupt
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_MMIS)
	}

	// Do not respond to any endpoint or FIFO-related interrupts until we have
	// completed initialization without error. The dhwStateDefault state indicates
	// we have a default (0) device address, and anything greater means we have
	// been assigned a bus-unique device address from the host.
	if d.state >= dhwStateDefault {
		// Bit 4 RXFLVL: Rx FIFO non-empty
		//   Indicates that there is at least one packet pending to be read from the
		//   Rx FIFO.
		if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_RXFLVL != 0 {
			d.glo.GINTMSK.ClearBits(stm32.USB_GINTMSK_RXFLVLM)

			// Constant values for the packet status (PKTSTS) field of Rx status read
			// and pop register (GRXSTSP). Since most of these trigger a distinct
			// interrupt, we only need to handle the ones that do not.
			const (
				statusNAK              = 1 // 0001 - triggers interrupt
				statusDataReceived     = 2 // 0010
				statusTransferComplete = 3 // 0011 - triggers interrupt
				statusSetupComplete    = 4 // 0100 - triggers interrupt
				statusSetupReceived    = 6 // 0110
			)

			// Pop the Rx FIFO (required to trigger the interrupts above).
			s := d.glo.GRXSTSP.Get()
			n := rxEndpoint(uint8(s & stm32.USB_GRXSTSP_EPNUM))
			z := (s & stm32.USB_GRXSTSP_BCNT) >> stm32.USB_GRXSTSP_BCNT_Pos

			ep := d.endpointInfo(n)
			//id := (s & stm32.USB_GRXSTSP_DPID) >> stm32.USB_GRXSTSP_DPID_Pos

			// Determine the USB core status, and respond to ONLY the events which do
			// not have dedicated interrupts.
			switch (s & stm32.USB_GRXSTSP_PKTSTS) >> stm32.USB_GRXSTSP_PKTSTS_Pos {
			// Data received by USB core and placed in Rx FIFO. Call endpointRead to
			// drain the FIFO and copy its content to application memory (into the
			// buffer declared by current device class configuration).
			case statusDataReceived:
				if z > 0 {
					d.endpointRead(n, ep.zfer.data, z)
				}
			// SETUP data received by USB core and placed in Rx FIFO. Call
			// endpointRead to drain the FIFO and decode it as a dcdSetup structure,
			// stored in the receiver's d.setup. The data is processed by the generic
			// event handler (*dcd).event (event ID: dcdEventControlSetup) after the
			// core raises the OUT endpoint interrupt (OEPINT), and then sets the STUP
			// bit in the endpoint status register of control endpoint 0.
			case statusSetupReceived:
				// Read 8 bytes from control endpoint 0 Rx FIFO
				d.endpointRead(n, d.controlBuffer(0), z)
				d.setup = setupFrom(d.controlBuffer(0))
			}
			d.glo.GINTMSK.SetBits(stm32.USB_GINTMSK_RXFLVLM)
		}

		// Bit 19 OEPINT: OUT endpoint interrupt
		//   The core sets this bit to indicate that an interrupt is pending on one
		//   of the OUT endpoints of the core (in device mode). The application must
		//   read the OTG_DAINT register to determine the exact number of the OUT
		//   endpoint on which the interrupt occurred, and then read the
		//   corresponding OTG_DOEPINTx register to determine the exact cause of the
		//   interrupt. The application must clear the appropriate status bit in the
		//   corresponding OTG_DOEPINTx register to clear this bit.
		if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_OEPINT != 0 {
			d.glo.GINTMSK.ClearBits(stm32.USB_GINTMSK_OEPINT)

			// Handle each endpoint that has its OUT endpoint interrupt bit set.
			rx := (d.dev.DAINT.Get() & d.dev.DAINTMSK.Get()) >> 16
			for 0 != rx {
				// The endpoint number (without direction bit)
				num := bits.TrailingZeros32(rx) & descEndptAddrNumberMsk
				// Clear this endpoint to indicate it has been processed
				rx &^= 1 << num

				// Fully-qualified endpoint address
				n := rxEndpoint(uint8(num))
				// The block of registers for this individual OUT endpoint
				e := d.glo.OutEp(num)
				// The interrupt status for this individual OUT endpoint
				s := e.DOEPINT.Get() & d.dev.DOEPMSK.Get()
				x := d.endpointInfo(n).zfer

				// OUT transfer complete
				if stm32.USB_DOEPINT_XFRC == s&stm32.USB_DOEPINT_XFRC {
					e.DOEPINT.Set(stm32.USB_DOEPINT_XFRC)
					if nil != x {
						x.active = false
					}
					// Handle control transfers and data endpoints separately
					if 0 == num {
						d.controlComplete()
						if nil == x || 0 == x.len {
							d.endpointEnable(0, true, 0)
						}
					} else {
						// Notify upper-layer device class driver and event handlers.
						if nil != x && nil != x.complete {
							x.complete()
						}
					}
				}
				// SETUP packet reception complete
				if stm32.USB_DOEPINT_STUP == s&stm32.USB_DOEPINT_STUP {
					e.DOEPINT.Set(stm32.USB_DOEPINT_STUP)
					if 0 == num {
						// Process the SETUP data, which schedules any necessary transmit or
						// receive transfers that need to occur.
						d.event(dcdEvent{
							id:    dcdEventControlSetup,
							setup: d.setup,
						})
						if nil != x {
							x.active = false
						}
						d.controlComplete()
					}
				}
				if stm32.USB_DOEPINT_OTEPDIS == s&stm32.USB_DOEPINT_OTEPDIS {
					e.DOEPINT.Set(stm32.USB_DOEPINT_OTEPDIS)
				}
				if stm32.USB_DOEPINT_OTEPSPR == s&stm32.USB_DOEPINT_OTEPSPR {
					e.DOEPINT.Set(stm32.USB_DOEPINT_OTEPSPR)
				}
				if stm32.USB_DOEPINT_NAK == s&stm32.USB_DOEPINT_NAK {
					e.DOEPINT.Set(stm32.USB_DOEPINT_NAK)
				}
			}
			d.glo.GINTMSK.SetBits(stm32.USB_GINTMSK_OEPINT)
		}

		// Bit 18 IEPINT: IN endpoint interrupt
		//   The core sets this bit to indicate that an interrupt is pending on one
		//   of the IN endpoints of the core (in device mode). The application must
		//   read the OTG_DAINT register to determine the exact number of the IN
		//   endpoint on which the interrupt occurred, and then read the
		//   corresponding OTG_DIEPINTx register to determine the exact cause of the
		//   interrupt. The application must clear the appropriate status bit in the
		//   corresponding OTG_DIEPINTx register to clear this bit.
		if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_IEPINT != 0 {
			d.glo.GINTMSK.ClearBits(stm32.USB_GINTMSK_IEPINT)

			// Handle each endpoint that has its OUT endpoint interrupt bit set.
			tx := (d.dev.DAINT.Get() & d.dev.DAINTMSK.Get()) & 0xFFFF
			for 0 != tx {
				// The endpoint number (without direction bit)
				num := bits.TrailingZeros32(tx) & descEndptAddrNumberMsk
				// Clear this endpoint to indicate it has been processed
				tx &^= 1 << num

				// Fully-qualified endpoint address
				n := txEndpoint(uint8(num))
				// The block of registers for this individual IN endpoint
				e := d.glo.InEp(num)
				// The interrupt status for this individual IN endpoint
				s := e.DIEPINT.Get() & (d.dev.DIEPMSK.Get() |
					(((d.dev.DIEPEMPMSK.Get() >> num) & 0x1) <<
						descEndptAddrDirectionPos))
				x := d.endpointInfo(n).zfer

				// IN transfer complete
				if stm32.USB_DIEPINT_XFRC == s&stm32.USB_DIEPINT_XFRC {
					e.DIEPINT.Set(stm32.USB_DIEPINT_XFRC)
					// Disable Tx FIFO empty interrupt if still set for some reason.
					d.dev.DIEPEMPMSK.ClearBits(1 << num)
					if nil != x {
						x.active = false
					}
					// Handle control transfers and data endpoints separately
					if 0 == num {
						d.controlComplete()
					} else {
						// Notify upper-layer device class driver and event handlers.
						if nil != x && nil != x.complete {
							x.complete()
						}
					}
				}
				if stm32.USB_DIEPINT_TOC == s&stm32.USB_DIEPINT_TOC {
					e.DIEPINT.Set(stm32.USB_DIEPINT_TOC)
				}
				if stm32.USB_DIEPINT_INEPNE == s&stm32.USB_DIEPINT_INEPNE {
					e.DIEPINT.Set(stm32.USB_DIEPINT_INEPNE)
				}
				if stm32.USB_DIEPINT_EPDISD == s&stm32.USB_DIEPINT_EPDISD {
					e.DIEPINT.Set(stm32.USB_DIEPINT_EPDISD)
				}
				if stm32.USB_DIEPINT_ITTXFE == s&stm32.USB_DIEPINT_ITTXFE {
					e.DIEPINT.Set(stm32.USB_DIEPINT_ITTXFE)
				}
				// IN transmit FIFO empty, ready to send any buffered data stored on
				// this endpoint.
				if stm32.USB_DIEPINT_TXFE == s&stm32.USB_DIEPINT_TXFE {
					if nil != x && 0 != x.data && 0 != x.len {
						// Tx FIFO empty interrupt is unmasked only when the respective
						// endpoint has requested a transmission. We can now write its data
						// from memory to the empty FIFO. Afterwards, if all data has been
						// written, we mask the interrupt again to indicate this endpoint is
						// not waiting for an empty FIFO to transmit data.
						for {
							size := d.endpointTransmitRemain(uint8(num))
							if !(0 < size && size <= d.endpointTransmitAvail(uint8(num))) {
								// Transfer size is zero or greater than free space in Tx FIFO,
								// so do not attempt to write into FIFO. The FIFO empty interrupt
								// will remain enabled to retry transmission once sufficient space
								// is available.
								break
							}
							d.endpointWrite(n, x.data, size)
						}
						if d.endpointTransmitRemain(uint8(num)) <= 0 {
							// Disable Tx FIFO empty interrupt once all data has transmitted
							// (or no data exists in transmit buffer)
							d.dev.DIEPEMPMSK.ClearBits(1 << num)
							// Clear the endpoint transfer info
							x.data, x.len, x.count, x.active = 0, 0, 0, false
						}
					}
				}
			}
			d.glo.GINTMSK.SetBits(stm32.USB_GINTMSK_IEPINT)
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
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_WKUINT != 0 {
		d.dev.DCTL.ClearBits(stm32.USB_DCTL_RWUSIG)
		// TODO: resume
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_WKUINT)
	}

	// Bit 10 ESUSP: Early suspend
	//   The core sets this bit to indicate that an Idle state has been detected
	//   on the USB for 3 ms.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_ESUSP != 0 {
		// TODO: early suspend
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_ESUSP)
	}

	// Bit 15 EOPF: End of periodic frame interrupt
	//   Indicates that the period specified in the periodic frame interval field
	//   of the OTG_DCFG register (PFIVL bit in OTG_DCFG) has been reached in the
	//   current frame.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_EOPF != 0 {
		// TODO: end of periodic frame
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_EOPF)
	}

	// Bit 23 RSTDET: Reset detected interrupt
	//   In device mode, this interrupt is asserted when a reset is detected on
	//   the USB in partial power-down mode when the device is in suspend.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_RSTDET != 0 {
		// TODO: suspend
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_RSTDET)
	}

	// Bit 11 USBSUSP: USB suspend
	//   The core sets this bit to indicate that a suspend was detected on the
	//   USB. The core enters the suspended state when there is no activity on the
	//   data lines for an extended period of time.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_USBSUSP != 0 {
		// TODO: suspend
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_USBSUSP)
	}

	// Bit 27 LPMINT: LPM interrupt
	//   In device mode, this interrupt is asserted when the device receives an
	//   LPM transaction and responds with a non-ERRORed response.
	//   This field is valid only if the LPMEN bit in OTG_GLPMCFG is set to 1.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_LPMINT != 0 {
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_LPMINT)
		// TODO: suspend
	}

	// if d.state < dhwStateDefault {
	// Bit 12 USBRST: USB reset
	// The core sets this bit to indicate that a reset is detected on the USB.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_USBRST != 0 {

		d.dev.DCTL.ClearBits(stm32.USB_DCTL_RWUSIG)
		d.endpointFlush(descEndptAddrNumberMsk + 1) // all Tx FIFOs

		// Abort any ongoing transfers
		for i := 0; i < int(d.endpointCount()); i++ {
			ie, oe := d.glo.InEp(i), d.glo.OutEp(i)
			ie.DIEPINT.Set(0xFB7F)
			ie.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_STALL)
			ie.DIEPCTL.SetBits(stm32.USB_DIEPCTL_SNAK)
			oe.DOEPINT.Set(0xFB7F)
			oe.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_STALL)
			oe.DOEPCTL.SetBits(stm32.USB_DOEPCTL_SNAK)
		}

		// Unmask device endpoint interrupts
		d.dev.DAINTMSK.SetBits(0x00010001) // IN+OUT on control endpoint 0
		d.dev.DOEPMSK.SetBits(stm32.USB_DOEPMSK_STUPM | stm32.USB_DOEPMSK_XFRCM |
			stm32.USB_DOEPMSK_EPDM | stm32.USB_DOEPMSK_OTEPSPRM |
			stm32.USB_DOEPMSK_NAKM | stm32.USB_DOEPMSK_OTEPDM)
		d.dev.DIEPMSK.SetBits(stm32.USB_DIEPMSK_TOM | stm32.USB_DIEPMSK_XFRCM |
			stm32.USB_DIEPMSK_EPDM)

		// Allocate RAM for Rx/Tx FIFOs
		d.resetEndpoints()
		d.event(dcdEvent{id: dcdEventStatusReset})
		// Clear interrupt
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_USBRST)
	}

	// Bit 13 ENUMDNE: Enumeration done
	//   The core sets this bit to indicate that speed enumeration is complete.
	//   The application must read the OTG_DSTS register to obtain the
	//   enumerated speed.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_ENUMDNE != 0 {
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
		// Reset maximum packet size on IN (Tx) control endpoint 0. Reconfigured
		// in endpointEnable, which is called on event dcdEventPeripheralReady.
		d.glo.InEp(0).DIEPCTL.ClearBits(stm32.USB_DIEPCTL_MPSIZ)
		// Clear global NAKs
		d.dev.DCTL.SetBits(stm32.USB_DCTL_CGINAK)

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
		d.event(dcdEvent{id: dcdEventPeripheralReady})
		// Clear interrupt
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_ENUMDNE)
	}
	// }

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
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_SOF != 0 {
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_SOF)
	}

	// Bit 20 IISOIXFR: Incomplete isochronous IN transfer
	//   The core sets this interrupt to indicate that there is at least one
	//   isochronous IN endpoint on which the transfer is not completed in the
	//   current frame. This interrupt is asserted along with the End of periodic
	//   frame interrupt (EOPF) bit in this register.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_IISOIXFR != 0 {
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_IISOIXFR)
	}

	// Bit 21 INCOMPISOOUT: Incomplete isochronous OUT transfer
	//   In device mode, the core sets this interrupt to indicate that there is at
	//   least one isochronous OUT endpoint on which the transfer is not completed
	//   in the current frame. This interrupt is asserted along with the End of
	//   periodic frame interrupt (EOPF) bit in this register.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_PXFR_INCOMPISOOUT != 0 {
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_PXFR_INCOMPISOOUT)
	}

	// Bit 30 SRQINT: Session request/new session detected interrupt
	//   In device mode, this interrupt is asserted when VBUS is in the valid
	//   range for a B-peripheral device.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_SRQINT != 0 {
		d.glo.GINTSTS.Set(status & stm32.USB_GINTSTS_SRQINT)
	}

	// Bit 2 OTGINT: OTG interrupt
	//   The core sets this bit to indicate an OTG protocol event. The application
	//   must read the OTG interrupt status (OTG_GOTGINT) register to determine
	//   the exact event that caused this interrupt. The application must clear
	//   the appropriate status bit in the OTG_GOTGINT register to clear this bit.
	if status := d.glo.GINTSTS.Get(); status&stm32.USB_GINTSTS_OTGINT != 0 {
	}

	d.interruptEnable(true)
}

func (d *dhw) interruptEnable(enable bool) {
	if enable {
		d.irq.Enable()
		if nil != d.tim {
			d.tim.irq.Enable()
		}
	} else {
		d.irq.Disable()
		if nil != d.tim {
			d.tim.irq.Disable()
		}
	}
}

func (d *dhw) setDeviceAddress(addr uint16) {
	d.address = uint32(addr) & (stm32.USB_DCFG_DAD_Msk >> stm32.USB_DCFG_DAD_Pos)
	d.dev.DCFG.ClearBits(stm32.USB_DCFG_DAD_Msk)
	if addr > 0 {
		// Transition to "addressed" peripheral state if we received a non-default
		// (non-zero) address.
		d.state = dhwStateAddress
		d.dev.DCFG.SetBits(d.address << stm32.USB_DCFG_DAD_Pos)
	} else {
		if dhwStateInit == d.state {
			// At this point, we transition to the USB-specified "Default" state,
			// representing a device that has successfully initialized, enumerated on
			// the bus, and received a USB reset. It has NOT received a SET_ADDRESS
			// command yet, and is therefore not yet ready to be used.
			d.state = dhwStateDefault
		}
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

// controlComplete handles control transfer completion on endpoint 0 based on
// current stage.
func (d *dhw) controlComplete() {
	switch d.stage {
	case dcdStageDataIn:
		// data IN transfer complete, status stage, read ACK from host
		d.stage = dcdStageStatusIn
		d.controlReceive(0, 0, false)
	case dcdStageDataOut:
		// data OUT transfer complete, status stage, send ACK to host
		d.stage = dcdStageStatusOut
		d.controlTransmit(0, 0, false)
	case dcdStageStatusIn, dcdStageStatusOut:
		// status ACK already received, no action needed
	default:
		// invalid endpoint state, abort any transfers and stall
		d.controlStall()
	}
}

// controlReceive receives (Rx, OUT) data on control endpoint 0.
func (d *dhw) controlReceive(data uintptr, size uint32, notify bool) {

	xfer := &dhwTransfer{data: data, len: size}
	if notify {
		xfer.complete = d.controlComplete
	}
	d.endpointReceive(rxEndpoint(0), xfer)
}

// controlTransmit transmits (Tx, IN) data on control endpoint 0.
func (d *dhw) controlTransmit(data uintptr, size uint32, notify bool) {

	xfer := &dhwTransfer{data: data, len: size}
	if notify {
		xfer.complete = d.controlComplete
	}
	d.endpointTransmit(txEndpoint(0), xfer)
}

// =============================================================================
//  Endpoint Descriptor
// =============================================================================

// dhwEndpoint defines a USB standard endpoint, used as the general channel of
// communication between host and device.
type dhwEndpoint struct {
	num   uint8        // Endpoint number
	isTx  bool         // Endpoint direction (is IN endpoint)
	stall bool         // Endpoint stall condition
	kind  uint8        // Endpoint type
	fifo  uint8        // Transmission FIFO number
	resv  uint32       // Reserved size in FIFO
	size  uint32       // Endpoint max packet size (bytes)
	zfer  *dhwTransfer // Current transfer descriptor
}

type dhwTransfer struct {
	data     uintptr      // Pointer to transfer buffer
	len      uint32       // Total number of bytes in transfer
	count    uint32       // Number of bytes transferred
	active   bool         // Transfer is currently being processed by scheduler
	complete func()       // Callback invoked when transfer completes
	next     *dhwTransfer // Next transfer to perform after completion
}

// endpointAlloc allocates FIFO buffer space and initializes the endpoint state
// buffer for the given endpoint.
func (d *dhw) endpointAlloc(endpoint, kind uint8, reserved, size uint32) {

	var fifo uint8
	var isTx bool
	num, _ := unpackEndpoint(endpoint)

	switch endpoint {
	case rxEndpoint(endpoint):
		// There is only a single Rx FIFO shared by all endpoints, regardless of the
		// current device class configuration. This FIFO is managed exclusively
		// through control endpoint 0, which is merely a convention to simplify
		// system design. For example:
		// * This "control FIFO" must account for space required to receive data on
		//   all other Rx endpoints used for a given device class configuration.
		// * Furthermore, the "control FIFO" must be reconfigured to account for all
		//   incoming Rx endpoints if/when the device class or class configuration
		//   is changed at runtime.
		// If any Rx endpoint other than 0 is received, the FIFO allocation request
		// is ignored.
		if 0 == num {
			words := reserved / 4
			d.glo.GRXFSIZ.Set(words)
		}
	case txEndpoint(endpoint):
		txOffset := d.glo.GRXFSIZ.Get()
		words := reserved / 4
		fifo = num
		isTx = true
		if 0 == num {
			d.glo.DIEPTXF0.Set((words << 16) | txOffset)
		} else {
			txOffset += d.glo.DIEPTXF0.Get() >> 16
			for i := uint8(0); i < num-1; i++ {
				txOffset += d.glo.DIEPTXF[i].Get() >> 16
			}
			d.glo.DIEPTXF[num-1].Set((words << 16) | txOffset)
		}
	}

	// Initialize endpoint state info buffer
	*d.endpointInfo(endpoint) = dhwEndpoint{
		num:   num,            // Endpoint number
		isTx:  isTx,           // Endpoint direction (is IN endpoint)
		stall: false,          // Endpoint stall condition
		kind:  kind,           // Endpoint type
		fifo:  fifo,           // Transmission FIFO number
		resv:  reserved,       // Reserved size in FIFO
		size:  size,           // Endpoint max packet size (bytes)
		zfer:  &dhwTransfer{}, // Current transfer descriptor
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

// endpointRxBuffer returns a pointer into the given endpoint's receive buffer
// at the given index offset. Offset is treated as a circular index, such that
// negative values offset from the end of the buffer (wrap on underflow), and
// positive values offset from the beginning (wrap on overflow).
func (d *dhw) endpointRxBuffer(endpoint uint8, offset int) uintptr {

	switch d.cc.id {
	// CDC-ACM (single)
	case classDeviceCDCACM:
		return d.uartRxBuffer(offset)

	// HID
	case classDeviceHID:
		//hid := &descHID[d.cc.config-1]
		//return uintptr(unsafe.Pointer(&hid.cx[wrap(offset, len(hid.cx))]))

	// Unhandled device class
	default:
	}

	return 0
}

// endpointTxBuffer returns a pointer into the given endpoint's transmit buffer
// at the given index offset. Offset is treated as a circular index, such that
// negative values offset from the end of the buffer (wrap on underflow), and
// positive values offset from the beginning (wrap on overflow).
func (d *dhw) endpointTxBuffer(endpoint uint8, offset int) uintptr {

	switch d.cc.id {
	// CDC-ACM (single)
	case classDeviceCDCACM:
		return d.uartTxBuffer(offset)

	// HID
	case classDeviceHID:
		//hid := &descHID[d.cc.config-1]
		//return uintptr(unsafe.Pointer(&hid.cx[wrap(offset, len(hid.cx))]))

	// Unhandled device class
	default:
	}

	return 0
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
		d.controlReceive(d.controlBuffer(0), 3*dcdSetupSize, true)

		// Enable both Rx and Tx for control endpoint 0. These are recursive calls
		// to the base case (control == false).
		d.endpointEnable(rxEndpoint(0), false, 0)
		d.endpointEnable(txEndpoint(0), false, 0)

	} else {
		num, _ := unpackEndpoint(endpoint)
		ep := d.endpointInfo(endpoint)

		switch endpoint {
		case rxEndpoint(endpoint): // OUT
			e := d.glo.OutEp(int(num))
			d.dev.DAINTMSK.SetBits(stm32.USB_DAINTMSK_OEPM & ((1 << num) << 16))
			if !e.DOEPCTL.HasBits(stm32.USB_DOEPCTL_USBAEP) {
				e.DOEPCTL.SetBits((ep.size & stm32.USB_DOEPCTL_MPSIZ) |
					(uint32(ep.kind) << stm32.USB_DOEPCTL_EPTYP_Pos) |
					stm32.USB_DIEPCTL_SD0PID_SEVNFRM | stm32.USB_DOEPCTL_USBAEP)
			}
		case txEndpoint(endpoint): // IN
			e := d.glo.InEp(int(num))
			d.dev.DAINTMSK.SetBits(stm32.USB_DAINTMSK_IEPM & (1 << num))
			if !e.DIEPCTL.HasBits(stm32.USB_DIEPCTL_USBAEP) {
				e.DIEPCTL.SetBits((ep.size & stm32.USB_DIEPCTL_MPSIZ) |
					(uint32(ep.kind) << stm32.USB_DIEPCTL_EPTYP_Pos) |
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
		e := d.glo.OutEp(int(num))
		if e.DOEPCTL.HasBits(stm32.USB_DOEPCTL_EPENA) {
			e.DOEPCTL.SetBits(stm32.USB_DOEPCTL_SNAK | stm32.USB_DOEPCTL_EPDIS)
		}
		d.dev.DEACHMSK.ClearBits(stm32.USB_DAINTMSK_OEPM & ((1 << num) << 16))
		d.dev.DAINTMSK.ClearBits(stm32.USB_DAINTMSK_OEPM & ((1 << num) << 16))
		e.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_USBAEP | stm32.USB_DOEPCTL_MPSIZ |
			stm32.USB_DOEPCTL_SD0PID_SEVNFRM | stm32.USB_DOEPCTL_EPTYP)

	case txEndpoint(endpoint): // IN
		e := d.glo.InEp(int(num))
		if e.DIEPCTL.HasBits(stm32.USB_DIEPCTL_EPENA) {
			e.DIEPCTL.SetBits(stm32.USB_DIEPCTL_SNAK | stm32.USB_DIEPCTL_EPDIS)
		}
		d.dev.DEACHMSK.ClearBits(stm32.USB_DAINTMSK_IEPM & (1 << num))
		d.dev.DAINTMSK.ClearBits(stm32.USB_DAINTMSK_IEPM & (1 << num))
		e.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_USBAEP |
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
		e := d.glo.OutEp(int(num))
		if !e.DOEPCTL.HasBits(stm32.USB_DOEPCTL_EPENA) && 0 != num {
			e.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_EPDIS)
		}
		e.DOEPCTL.SetBits(stm32.USB_DOEPCTL_STALL)

	case txEndpoint(endpoint): // IN
		e := d.glo.InEp(int(num))
		if !e.DIEPCTL.HasBits(stm32.USB_DIEPCTL_EPENA) && 0 != num {
			e.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_EPDIS)
		}
		e.DIEPCTL.SetBits(stm32.USB_DIEPCTL_STALL)
	}

	if 0 == endpoint {
		// Prepare endpoint 0 for setup packet transactions
		d.controlReceive(d.controlBuffer(0), 3*dcdSetupSize, true)
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
		e := d.glo.OutEp(int(num))
		e.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_STALL)
		if isBulkOrInt {
			e.DOEPCTL.SetBits(stm32.USB_DOEPCTL_SD0PID_SEVNFRM)
		}

	case txEndpoint(endpoint): // IN
		e := d.glo.InEp(int(num))
		e.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_STALL)
		if isBulkOrInt {
			e.DIEPCTL.SetBits(stm32.USB_DIEPCTL_SD0PID_SEVNFRM)
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

// controlReceive receives (Rx, OUT) data on control endpoint 0.
func (d *dhw) endpointReceive(endpoint uint8, xfer *dhwTransfer) {

	ep := d.endpointInfo(rxEndpoint(uint8(endpoint)))
	// if nil != ep.zfer && ep.zfer.active {
	// 	return
	// }
	ep.zfer = xfer
	ep.zfer.active = true

	// For Rx (OUT) endpoints, we do not want the extra packet introduced when
	// transfer size is a multiple of max packet size (i.e., size % mps = 0).
	// The transfer size must be strictly greater than max packet size in order
	// to request an additional packet; using (size - 1) ensures this strict
	// inequality holds before performing the integer division.
	ep.zfer.len &= stm32.USB_DOEPTSIZ_XFRSIZ_Msk >> stm32.USB_DOEPTSIZ_XFRSIZ_Pos
	mps := d.endpointInfo(endpoint).size
	pkt := uint32(1) + (ep.zfer.len-1)/mps

	// For Rx (OUT) packets, always prime the endpoint size register (DOEPTSIZ)
	// with packets whose sizes are a multiple of max packet size, because we
	// are receiving a full packet over the USB anyway. The number of ~packets~
	// to receive, however, is based on the originally-requested transfer size.
	ep.zfer.len = pkt * mps

	// transfer size (XFRSIZ) represents the entire transfer size, not just the
	// short packet remaining after 0 or more max-packet-sized packets (PKTCNT).
	e := d.glo.OutEp(int(endpoint))
	e.DOEPTSIZ.Set((pkt << stm32.USB_DOEPTSIZ_PKTCNT_Pos) |
		(ep.zfer.len << stm32.USB_DOEPTSIZ_XFRSIZ_Pos) | stm32.USB_DOEPTSIZ_STUPCNT)

	e.DOEPCTL.ClearBits(stm32.USB_DOEPCTL_STALL)
	e.DOEPCTL.SetBits(stm32.USB_DOEPCTL_CNAK | stm32.USB_DOEPCTL_EPENA)
}

// controlTransmit transmits (Tx, IN) data on control endpoint 0.
func (d *dhw) endpointTransmit(endpoint uint8, xfer *dhwTransfer) {

	ep := d.endpointInfo(txEndpoint(uint8(endpoint)))
	// if nil != ep.zfer && ep.zfer.active {
	// 	return
	// }
	ep.zfer = xfer
	ep.zfer.active = true

	// If transfer size is a non-zero multiple of max packet size, the STM32
	// core requires a separate zero-length packet be scheduled for transmission
	// to complete the handshaking protocol. This extra packet is scheduled by
	// always using pkt = 1 + size/mps (specifically when size % mps = 0).
	ep.zfer.len &= stm32.USB_DIEPTSIZ_XFRSIZ_Msk >> stm32.USB_DIEPTSIZ_XFRSIZ_Pos
	mps := ep.size
	pkt := uint32(1) + ep.zfer.len/mps

	// transfer size (XFRSIZ) represents the entire transfer size, not just the
	// short packet remaining after 0 or more max-packet-sized packets (PKTCNT).
	e := d.glo.InEp(int(endpoint))
	e.DIEPTSIZ.Set((pkt << stm32.USB_DIEPTSIZ_PKTCNT_Pos) |
		(ep.zfer.len << stm32.USB_DIEPTSIZ_XFRSIZ_Pos))

	e.DIEPCTL.ClearBits(stm32.USB_DIEPCTL_STALL)
	e.DIEPCTL.SetBits(stm32.USB_DIEPCTL_CNAK | stm32.USB_DIEPCTL_EPENA)

	// Enable the FIFO empty interrupt for this endpoint if we are requesting a
	// non-ZLP transfer.
	if ep.zfer.len > 0 {
		d.dev.DIEPEMPMSK.SetBits(1 << (endpoint & descEndptAddrNumberMsk))
	}
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
	ep := d.endpointInfo(txEndpoint(endpoint))
	if nil == ep.zfer {
		return 0
	}
	size := ep.zfer.len - ep.zfer.count
	if size > ep.size {
		size = ep.size
	}
	return size
}

// endpointRead reads a packet from the given endpoint's Rx FIFO into the buffer
// referenced by the given data pointer.
func (d *dhw) endpointRead(endpoint uint8, data uintptr, size uint32) {
	// Normalize endpoint in case we only received an endpoint address number
	n := rxEndpoint(endpoint)
	if nil == d.endpointInfo(n).zfer {
		return
	}
	fifo := d.glo.EpFifo(int(d.endpointInfo(n).fifo))
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
	d.endpointInfo(n).zfer.data = end
	d.endpointInfo(n).zfer.count += size
}

// endpointWrite writes a packet from the buffer referenced by the given data
// pointer to the given endpoint's Tx FIFO.
func (d *dhw) endpointWrite(endpoint uint8, data uintptr, size uint32) {
	// Normalize endpoint in case we only received an endpoint address number
	n := txEndpoint(endpoint)
	if nil == d.endpointInfo(n).zfer {
		return
	}
	fifo := d.glo.EpFifo(int(d.endpointInfo(n).fifo))
	for i := uint32(0); i < (size+3)>>2; i++ {
		fifo.Set(*(*uint32)(unsafe.Pointer(data)))
		data += 4
	}
	d.endpointInfo(n).zfer.data += uintptr(size)
	d.endpointInfo(n).zfer.count += size
}

// =============================================================================
//  [CDC-ACM] Serial UART (Virtual COM Port)
// =============================================================================

// dhwTimer defines a general-purpose timer for USB device classes.
type dhwTimer struct {
	*dhwTimerClass
	configured bool
	irq        interrupt.Interrupt
}

// dhwTimerConfig contains the configuration parameters used to initialize a
// device class timer.
type dhwTimerConfig struct {
	priority uint8  // interrupt priority (lower number => higher priority)
	period   uint32 // 1/period (microseconds) = interrupt frequency (Hz)
}

// dhwTimerPriority defines the priority for USB device class timers, such as
// the USB Serial (UART) Tx flush to FIFO poll.
const dhwTimerPriority = 0xD0

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

	d.endpointReceive(rxEndpoint(descCDCACMEndpointDataRx),
		&dhwTransfer{
			data:     d.uartRxBuffer(0),
			len:      uint32(acm.rxSize),
			complete: d.uartReceive,
		})

	// d.tim = uartTimer.configure(
	// 	dhwTimerConfig{
	// 		priority: dhwTimerPriority,
	// 		period:   0xFFFF, // interrupt every 1 ms
	// 	})
}

func (d *dhw) uartReady() bool {
	return d.state == dhwStateConfigure
}

// uartRxBuffer returns a pointer into the UART receive data (bulk OUT endpoint)
// buffer at the given index offset. Offset is treated as a circular index, such
// that negative values offset from the end of the buffer (wrap on underflow),
// and positive values offset from the beginning (wrap on overflow).
func (d *dhw) uartRxBuffer(offset int) uintptr {
	if d.state != dhwStateConfigure {
		return 0
	}
	acm := &descCDCACM[d.cc.config-1]
	pos := offset
	if 0 != pos {
		pos = wrap(pos, len(acm.rx))
	}
	return uintptr(unsafe.Pointer(&acm.rx[pos]))
}

// uartTxBuffer returns a pointer into the UART transmit data (bulk IN endpoint)
// buffer at the given index offset. Offset is treated as a circular index, such
// that negative values offset from the end of the buffer (wrap on underflow),
// and positive values offset from the beginning (wrap on overflow).
func (d *dhw) uartTxBuffer(offset int) uintptr {
	if d.state != dhwStateConfigure {
		return 0
	}
	acm := &descCDCACM[d.cc.config-1]
	pos := offset
	if 0 != pos {
		pos = wrap(pos, len(acm.tx))
	}
	return uintptr(unsafe.Pointer(&acm.tx[pos]))
}

func (d *dhw) uartSetLineState(dtr, rts bool) {
}

func (d *dhw) uartSetLineCoding(coding descCDCACMLineCoding) {
}

func (d *dhw) uartReceive() {
	if d.state != dhwStateConfigure {
		return
	}

	// Slice Rx buffer up to the number of bytes read from FIFO last transmission.
	// The .len field contains a multiple of the bulk data packet size - 64 or 512
	// bytes for full-speed or high-speed, respectively - but the .count field
	// contains the total number of bytes actually read from the Rx FIFO, which
	// can accumulate across multiple packets for a single transfer.
	ep := d.endpointInfo(rxEndpoint(descCDCACMEndpointDataRx))
	if nil == ep.zfer {
		return
	}

	acm := &descCDCACM[d.cc.config-1]

	// Now fill UART ring buffer with as many bytes as we received. If there is
	// insufficient space, we simply drop the extra bytes.
	_ = acm.rq.put(acm.rx[:ep.zfer.count])

	// Prepare for next packet reception
	d.endpointReceive(rxEndpoint(descCDCACMEndpointDataRx),
		&dhwTransfer{
			data:     d.uartRxBuffer(0),
			len:      uint32(acm.rxSize),
			complete: d.uartReceive,
		})
}

func (d *dhw) uartTransmit() {
	if d.state != dhwStateConfigure {
		return
	}
	ep := d.endpointInfo(rxEndpoint(descCDCACMEndpointDataRx))
	if nil == ep.zfer {
		return
	}
	s := 0
	if ep.zfer.active {
		s += int(ep.zfer.len)
	}
	acm := &descCDCACM[d.cc.config-1]
	if c := acm.tq.get(acm.tx[s:]); c > 0 {
		d.endpointTransmit(txEndpoint(descCDCACMEndpointDataTx),
			&dhwTransfer{
				data:     d.uartTxBuffer(s),
				len:      uint32(c),
				complete: nil,
			})
	}
}

func (d *dhw) uartAvailable() int {
	if d.state != dhwStateConfigure {
		return 0
	}
	return descCDCACM[d.cc.config-1].rq.available()
}

func (d *dhw) uartReadByte() (uint8, bool) {
	if d.state != dhwStateConfigure {
		return 0, false
	}
	b := []uint8{0}
	ok := d.uartRead(b) > 0
	return b[0], ok
}

func (d *dhw) uartRead(data []uint8) int {
	if d.state != dhwStateConfigure {
		return 0
	}
	return descCDCACM[d.cc.config-1].rq.get(data)
}

func (d *dhw) uartWriteByte(c uint8) bool {
	if d.state != dhwStateConfigure {
		return false
	}
	return 1 == d.uartWrite([]uint8{c})
}

func (d *dhw) uartWrite(data []uint8) int {
	if d.state != dhwStateConfigure {
		return 0
	}
	if n := descCDCACM[d.cc.config-1].tq.put(data); n > 0 {
		d.uartTransmit()
		return n
	}
	return 0
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

// =============================================================================
//  Data Queue (circular FIFO)
// =============================================================================

type dhwDataQueue struct {
	fifo *[dhwDataQueueSize]uint8
	head *volatile.Register32
	tail *volatile.Register32
}

// flush discards all buffered data.
func (q *dhwDataQueue) flush() {
	for i := range q.fifo {
		q.fifo[i] = 0
	}
	q.head.Set(0)
	q.tail.Set(0)
}

func (q *dhwDataQueue) available() int {
	return int(q.head.Get() - q.tail.Get())
}

func (q *dhwDataQueue) get(data []uint8) int {

	less := uint32(len(data))
	if less == 0 {
		return 0 // nothing to get
	}

	head := q.head.Get()
	tail := q.tail.Get()
	used := head - tail

	if used == 0 {
		return 0 // empty queue
	}

	if less > used {
		less = used // only get from used space
	}

	for i := uint32(0); i < less; i++ {
		tail++
		data[i] = q.fifo[tail%dhwDataQueueSize]
	}
	q.tail.Set(tail)

	return int(less)
}

func (q *dhwDataQueue) put(data []uint8) int {

	more := uint32(len(data))
	if more == 0 {
		return 0 // nothing to put
	}

	head := q.head.Get()
	tail := q.tail.Get()
	used := head - tail

	if used == dhwDataQueueSize {
		return 0 // full queue
	}

	if used+more > dhwDataQueueSize {
		more = dhwDataQueueSize - used // only put to unused space
	}

	for i := uint32(0); i < more; i++ {
		head++
		q.fifo[head%dhwDataQueueSize] = data[i]
	}
	q.head.Set(head)

	return int(more)
}

func (q *dhwDataQueue) peek() (uint8, bool) {

	head := q.head.Get()
	tail := q.tail.Get()
	used := head - tail

	if used == 0 {
		return 0, false // empty queue
	}

	return q.fifo[tail%dhwDataQueueSize], true
}

// =============================================================================
//  Transfer Queue (circular FIFO)
// =============================================================================

//type dhwTransferQueue struct {
//	fifo *[dhwTransferQueueSize]*dhwTransfer
//	head *volatile.Register32
//	tail *volatile.Register32
//}
//
//// flush discards all buffered data.
//func (q *dhwTransferQueue) flush() {
//	for i := range q.fifo {
//		q.fifo[i] = nil
//	}
//	q.head.Set(0)
//	q.tail.Set(0)
//}
//
//func (q *dhwTransferQueue) available() int {
//	return int(q.head.Get() - q.tail.Get())
//}
//
//func (q *dhwTransferQueue) get() (*dhwTransfer, bool) {
//
//	head := q.head.Get()
//	tail := q.tail.Get()
//	used := head - tail
//
//	if used == 0 {
//		return nil, false // empty queue
//	}
//
//	tail++
//	x := q.fifo[tail%dhwTransferQueueSize]
//	q.tail.Set(tail)
//
//	return x, true
//}
//
//func (q *dhwTransferQueue) put(x *dhwTransfer) bool {
//
//	head := q.head.Get()
//	tail := q.tail.Get()
//	used := head - tail
//
//	if used == dhwTransferQueueSize {
//		return false // full queue
//	}
//
//	head++
//	q.fifo[head%dhwTransferQueueSize] = x
//	q.head.Set(head)
//
//	return true
//}
//
//func (q *dhwTransferQueue) peek() (*dhwTransfer, bool) {
//
//	head := q.head.Get()
//	tail := q.tail.Get()
//	used := head - tail
//
//	if used == 0 {
//		return nil, false // empty queue
//	}
//
//	return q.fifo[tail%dhwTransferQueueSize], true
//}
