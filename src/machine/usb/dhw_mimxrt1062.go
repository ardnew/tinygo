// +build mimxrt1062

package usb

// Implementation of USB device controller hardware abstraction (dhw) for NXP
// iMXRT1062.

import (
	"device/arm"
	"device/nxp"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

// dcdInterruptPriority defines the priority for all USB device interrupts.
const dcdInterruptPriority = 3

// dhw implements USB device controller hardware abstraction for iMXRT1062.
type dhw struct {
	drv *dcd                // USB device controller driver
	bus *nxp.USB_Type       // USB core register
	phy *nxp.USBPHY_Type    // USB PHY register
	irq interrupt.Interrupt // USB IRQ, only a single interrupt on iMXRT1062

	timerInterrupt [2]func()
	sofUsage       uint8
	rebootTimer    uint8
	speed          uint8 // bus speed (0=full, 1=low, 2=high, 4=super)
}

// cycleCount uses the ARM debug cycle counter available on iMXRT1062 (enabled
// in runtime_mimxrt1062_time.go) to return the number of CPU cycles since boot.
//go:inline
func cycleCount() uint32 {
	return (*volatile.Register32)(unsafe.Pointer(uintptr(0xe0001004))).Get()
}

//go:linkname ticks runtime.ticks
func ticks() int64

// allocDHW returns a reference to the USB hardware abstraction for the given
// device controller driver. Should be called only one time and during device
// controller initialization.
func allocDHW(d *dcd) *dhw {
	switch d.port {
	case 0:
		dhwInstance[d.id].drv = d
		dhwInstance[d.id].bus = nxp.USB1
		dhwInstance[d.id].phy = nxp.USBPHY1
		dhwInstance[d.id].irq =
			interrupt.New(nxp.IRQ_USB_OTG1,
				func(interrupt.Interrupt) {
					coreInstance[0].dc.hw.event()
				})

	case 1:
		dhwInstance[d.id].drv = d
		dhwInstance[d.id].bus = nxp.USB2
		dhwInstance[d.id].phy = nxp.USBPHY2
		dhwInstance[d.id].irq =
			interrupt.New(nxp.IRQ_USB_OTG2,
				func(interrupt.Interrupt) {
					//coreInstance[1].dc.hw.event()
				})
	}

	// Clear installed timer callbacks
	dhwInstance[d.id].timerInterrupt[0] = nil
	dhwInstance[d.id].timerInterrupt[1] = nil

	return &dhwInstance[d.id]
}

// init configures the USB hardware for device mode operation by resetting the
// USB PHY, enabling power on the bus, and initializing core registers and
// interrupts.
func (d *dhw) init() status {

	// Reset the controller
	d.phy.CTRL_SET.Set(nxp.USBPHY_CTRL_SFTRST)
	d.bus.USBCMD.SetBits(nxp.USB_USBCMD_RST)
	for d.bus.USBCMD.HasBits(nxp.USB_USBCMD_RST) {
	}
	// Clear interrupts
	d.interruptsClearUSB()
	d.phy.CTRL_CLR.Set(nxp.USBPHY_CTRL_CLKGATE | nxp.USBPHY_CTRL_SFTRST)
	d.phy.PWD.Set(0)

	// Clear the controller mode field and set to device mode:
	//   Controller mode (CM) 0x0=idle, 0x2=device-only, 0x3=host-only
	d.bus.USBMODE.ReplaceBits(nxp.USB_USBMODE_CM_CM_2,
		nxp.USB_USBMODE_CM_Msk>>nxp.USB_USBMODE_CM_Pos, nxp.USB_USBMODE_CM_Pos)

	d.bus.USBCMD.ClearBits(nxp.USB_USBCMD_ITC_Msk)  // No interrupt threshold
	d.bus.USBMODE.SetBits(nxp.USB_USBMODE_SLOM_Msk) // Disable setup lockout
	d.bus.USBMODE.ClearBits(nxp.USB_USBMODE_ES_Msk) // Use little-endianness

	// Install base address of endpoints
	d.bus.ASYNCLISTADDR.Set(uint32(uintptr(unsafe.Pointer(d.drv.stat))))

	// Enable interrupts in USB core
	d.bus.USBINTR.Set(
		nxp.USB_USBINTR_UE_Msk | // bus enable
			nxp.USB_USBINTR_UEE_Msk | // bus error
			nxp.USB_USBINTR_PCE_Msk | // port change detect
			nxp.USB_USBINTR_URE_Msk | // bus reset
			nxp.USB_USBINTR_SLE) // sleep enable

	// Ensure D+ pulled down long enough for host to detect previous disconnect
	udelay(5000)

	return statusOK
}

// enable causes the USB core to enter (or exit) the normal run state.
func (d *dhw) enable(enable bool) {
	if enable {
		d.bus.USBCMD.SetBits(nxp.USB_USBCMD_RS)
	} else {
		d.bus.USBCMD.ClearBits(nxp.USB_USBCMD_RS)
	}
}

// event handles the USB hardware interrupts events and notifies the device
// controller driver using a common "virtual interrupt" code.
func (d *dhw) event() {

	// read and clear the interrupts that fired
	status := d.bus.USBSTS.Get() & d.bus.USBINTR.Get()
	d.bus.USBSTS.Set(status)

	// USB Interrupt (USBINT) - R/WC
	// This bit is set by the Host/Device Controller when the cause of an
	// interrupt is a completion of a USB transaction where the Transfer
	// Descriptor (TD) has an interrupt on complete (IOC) bit set.
	// This bit is also set by the Host/Device Controller when a short packet is
	// detected. A short packet is when the actual number of bytes received was
	// less than the expected number of bytes.
	if 0 != status&nxp.USB_USBSTS_UI {

		setupStatus := d.bus.ENDPTSETUPSTAT.Get()
		for 0 != setupStatus {
			d.bus.ENDPTSETUPSTAT.Set(setupStatus)
			var setup dcdSetup
			ready := false
			for !ready {
				d.bus.USBCMD.SetBits(nxp.USB_USBCMD_SUTW)
				setup = d.drv.stat.setup
				ready = d.bus.USBCMD.HasBits(nxp.USB_USBCMD_SUTW)
			}
			d.bus.USBCMD.ClearBits(nxp.USB_USBCMD_SUTW)
			// flush endpoint 0 (bit 0=Rx, 16=Tx)
			d.bus.ENDPTFLUSH.Set(0x00010001)
			// wait for flush to complete
			for d.bus.ENDPTFLUSH.HasBits(0x00010001) {
			}
			// Notify device controller driver
			d.drv.event(dcdEvent{
				id:    dcdEventControlSetup,
				setup: setup,
			})
			setupStatus = d.bus.ENDPTSETUPSTAT.Get()
		}

		completeStatus := d.bus.ENDPTCOMPLETE.Get()
		if 0 != completeStatus {
			d.bus.ENDPTCOMPLETE.Set(completeStatus)
			// Notify device controller driver
			d.drv.event(dcdEvent{
				id:   dcdEventTransactComplete,
				mask: completeStatus,
			})
		}
	}

	// USB Reset Received - R/WC
	// When the device controller detects a USB Reset and enters the default
	// state, this bit will be set to a one.
	// Software can write a 1 to this bit to clear the USB Reset Received status
	// bit.
	// Only used in device operation mode.
	if 0 != status&nxp.USB_USBSTS_URI {
		// clear all setup tokens
		d.bus.ENDPTSETUPSTAT.Set(d.bus.ENDPTSETUPSTAT.Get())
		// clear all endpoint complete status
		d.bus.ENDPTCOMPLETE.Set(d.bus.ENDPTCOMPLETE.Get())
		// wait on any endpoint priming
		for 0 != d.bus.ENDPTPRIME.Get() {
		}
		d.bus.ENDPTFLUSH.Set(0xFFFFFFFF)
		d.drv.event(dcdEvent{id: dcdEventStatusReset})
	}

	// General Purpose Timer Interrupt 0(GPTINT0) - R/WC
	// This bit is set when the counter in the GPTIMER0CTRL register transitions
	// to zero, writing a one to this bit clears it.
	if 0 != status&nxp.USB_USBSTS_TI0 {
		if nil != d.timerInterrupt[0] {
			d.timerInterrupt[0]()
		}
		d.drv.event(dcdEvent{id: dcdEventTimer, mask: 0})
	}

	// General Purpose Timer Interrupt 1(GPTINT1) - R/WC
	// This bit is set when the counter in the GPTIMER1CTRL register transitions
	// to zero, writing a one to this bit will clear it.
	if 0 != status&nxp.USB_USBSTS_TI1 {
		if nil != d.timerInterrupt[1] {
			d.timerInterrupt[1]()
		}
		d.drv.event(dcdEvent{id: dcdEventTimer, mask: 1})
	}

	// Port Change Detect - R/WC
	// The Host Controller sets this bit to a one when on any port a Connect
	// Status occurs, a Port Enable/Disable Change occurs, or the Force Port
	// Resume bit is set as the result of a J-K transition on the suspended port.
	// The Device Controller sets this bit to a one when the port controller
	// enters the full or high-speed operational state. When the port controller
	// exits the full or high-speed operation states due to Reset or Suspend
	// events, the notification mechanisms are the USB Reset Received bit and the
	// DCSuspend bits respectively.
	if 0 != status&nxp.USB_USBSTS_PCI {
		if d.bus.PORTSC1.HasBits(nxp.USB_PORTSC1_HSP) {
			d.speed = descDeviceSpeedHigh // 480 Mbit/sec
		} else {
			d.speed = descDeviceSpeedFull // 12 Mbit/sec
		}
		d.drv.event(dcdEvent{id: dcdEventStatusRun})
	}

	// DCSuspend - R/WC
	// When a controller enters a suspend state from an active state, this bit
	// will be set to a one. The device controller clears the bit upon exiting
	// from a suspend state. Only used in device operation mode.
	if 0 != status&nxp.USB_USBSTS_SLI {
		d.drv.event(dcdEvent{id: dcdEventStatusSuspend})
	}

	// USB Error Interrupt (USBERRINT) - R/WC
	// When completion of a USB transaction results in an error condition, this
	// bit is set by the Host/Device Controller. This bit is set along with the
	// USBINT bit, if the TD on which the error interrupt occurred also had its
	// interrupt on complete (IOC) bit set.
	// The device controller detects resume signaling only.
	if 0 != status&nxp.USB_USBSTS_UEI {
		d.drv.event(dcdEvent{id: dcdEventStatusError})
	}

	// SOF Received - R/WC
	// When the device controller detects a Start Of (micro) Frame, this bit will
	// be set to a one. When a SOF is extremely late, the device controller will
	// automatically set this bit to indicate that an SOF was expected.
	// Therefore, this bit will be set roughly every 1ms in device FS mode and
	// every 125us in HS mode and will be synchronized to the actual SOF that is
	// received.
	// Because the device controller is initialized to FS before connect, this bit
	// will be set at an interval of 1ms during the prelude to connect and chirp.
	// In host mode, this bit will be set every 125us and can be used by host
	// controller driver as a time base. Software writes a 1 to this bit to clear
	// it.
	if d.bus.USBINTR.HasBits(nxp.USB_USBINTR_SRE) &&
		0 != status&nxp.USB_USBSTS_SRI {
		if 0 != d.rebootTimer {
			d.rebootTimer -= 1
			if 0 == d.rebootTimer {
				d.interruptsEnableSOF(false, descCDCACMInterfaceCount)
			}
		}
	}
}

func (d *dhw) deleteCache(addr, size uintptr) {
	nxp.FlushDeleteDcache(addr, size)
}

func (d *dhw) flushCache(addr, size uintptr) {
	nxp.FlushDeleteDcache(addr, size)
}

func (d *dhw) controlSpeed() uint8 { return d.speed }

func (d *dhw) controlDeviceAddress(addr uint16) {
	d.bus.DEVICEADDR.Set(nxp.USB_DEVICEADDR_USBADRA |
		((uint32(addr) << nxp.USB_DEVICEADDR_USBADR_Pos) &
			nxp.USB_DEVICEADDR_USBADR_Msk))
}

func (d *dhw) controlLineState(coding *descCDCACMLineCoding, dtr, rts bool) {
	// TBD: does the PHY need to handle on iMXRT1062 (e.g., Teensyduino Loader)?
}

func (d *dhw) controlLineCoding(coding *descCDCACMLineCoding) {
	if 134 == coding.baud {
		d.interruptsEnableSOF(true, descCDCACMInterfaceCount)
	}
}

// controlStatus transitions transfers on control endpoint 0 into status stage.
func (d *dhw) controlStatus() {
	// Not used on iMXRT1062
}

// controlStall stalls a transfer on control endpoint 0. To stall a transfer on
// any other endpoint, use method endpointStall().
func (d *dhw) controlStall() {
	d.endpointStall(0)
}

func (d *dhw) endpointControlRegister(endpoint uint8) *volatile.Register32 {
	num, _ := unpackEndpoint(endpoint)
	switch num {
	case 0:
		return &d.bus.ENDPTCTRL0
	case 1:
		return &d.bus.ENDPTCTRL1
	case 2:
		return &d.bus.ENDPTCTRL2
	case 3:
		return &d.bus.ENDPTCTRL3
	case 4:
		return &d.bus.ENDPTCTRL4
	case 5:
		return &d.bus.ENDPTCTRL5
	case 6:
		return &d.bus.ENDPTCTRL6
	case 7:
		return &d.bus.ENDPTCTRL7
	}
	return nil
}

func (d *dhw) endpointEnable(endpoint uint8, control bool, config uint32) {
	// control endpoint 0 configured in dcd init
	if !control || 0 != endpoint&descEndptAddrNumberMsk {
		d.endpointControlRegister(endpoint & descEndptAddrNumberMsk).Set(config)
	}
}

func (d *dhw) endpointStatus(endpoint uint8) uint16 {
	status := uint16(0)
	switch endpoint {
	case rxEndpoint(endpoint):
		status |= uint16((d.endpointControlRegister(endpoint).Get() &
			nxp.USB_ENDPTCTRL0_RXS_Msk) >> nxp.USB_ENDPTCTRL0_RXS_Pos)
	case txEndpoint(endpoint):
		status |= uint16((d.endpointControlRegister(endpoint).Get() &
			nxp.USB_ENDPTCTRL0_TXS_Msk) >> nxp.USB_ENDPTCTRL0_TXS_Pos)
	}
	return status
}

// endpointStall stalls a transfer on the given endpoint.
func (d *dhw) endpointStall(endpoint uint8) {
	// RXS and TXS bits at same position in all endpoint control registers.
	d.endpointControlRegister(endpoint).SetBits(
		nxp.USB_ENDPTCTRL0_RXS | nxp.USB_ENDPTCTRL0_TXS,
	)
}

func (d *dhw) endpointPrime(mask uint32, transfer *dcdTransfer) {
	d.bus.ENDPTPRIME.Set(mask)
}

func (d *dhw) endpointPrimed() uint32 {
	return d.bus.ENDPTPRIME.Get()
}

func (d *dhw) endpointUnprime(mask uint32) {
	d.bus.ENDPTCOMPLETE.Set(mask)
}

func (d *dhw) timerConfigure(timer int, usec uint32, fn func()) {
	if timer < 0 || timer >= len(d.timerInterrupt) {
		return
	}
	d.timerInterrupt[timer] = fn
	switch timer {
	case 0:
		d.bus.GPTIMER0CTRL.Set(0)
		d.bus.GPTIMER0LD.Set(usec - 1)
		d.bus.USBINTR.SetBits(nxp.USB_USBINTR_TIE0)
	case 1:
		d.bus.GPTIMER1CTRL.Set(0)
		d.bus.GPTIMER1LD.Set(usec - 1)
		d.bus.USBINTR.SetBits(nxp.USB_USBINTR_TIE1)
	}
}

func (d *dhw) timerOneShot(timer int) {
	switch timer {
	case 0:
		d.bus.GPTIMER0CTRL.Set(
			nxp.USB_GPTIMER0CTRL_GPTRUN | nxp.USB_GPTIMER0CTRL_GPTRST)
	case 1:
		d.bus.GPTIMER1CTRL.Set(
			nxp.USB_GPTIMER1CTRL_GPTRUN | nxp.USB_GPTIMER1CTRL_GPTRST)
	}
}

func (d *dhw) timerStop(timer int) {
	switch timer {
	case 0:
		d.bus.GPTIMER0CTRL.Set(0)
	case 1:
		d.bus.GPTIMER1CTRL.Set(0)
	}
}

// interruptsClearUSB clears all interrupts associated with the receiver's USB
// port.
//go:inline
func (d *dhw) interruptsClearUSB() {
	// iMXRT1062 has only a single interrupt vector per USB core.
	m := d.interruptsDisable()
	switch d.drv.port {
	case 0:
		d.interruptsEnable(m & ^uintptr(nxp.IRQ_USB_OTG1))
	case 1:
		d.interruptsEnable(m & ^uintptr(nxp.IRQ_USB_OTG2))
	}
}

// interruptsEnableUSB enables or disables all interrupts associated with the
// receiver's USB port.
//go:inline
func (d *dhw) interruptsEnableUSB(enable bool) {
	// iMXRT1062 has only a single interrupt vector per USB core.
	if enable {
		d.irq.SetPriority(dcdInterruptPriority)
		d.irq.Enable()
	} else {
		d.irq.Disable()
	}
}

func (d *dhw) interruptsEnableSOF(enable bool, iface uint8) {
	if enable {
		ivm := arm.DisableInterrupts()
		d.sofUsage |= 1 << iface
		if !d.bus.USBINTR.HasBits(nxp.USB_USBINTR_SRE) {
			d.bus.USBSTS.Set(nxp.USB_USBSTS_SRI)
			d.bus.USBINTR.SetBits(nxp.USB_USBINTR_SRE)
		}
		d.rebootTimer = 80
		arm.EnableInterrupts(ivm)
	} else {
		d.sofUsage &^= 1 << iface
		if 0 == d.sofUsage {
			d.bus.USBINTR.ClearBits(nxp.USB_USBINTR_SRE)
		}
	}
}

// interruptsDisable disables all interrupts and returns the old system
// interrupt state mask.
//go:inline
func (d *dhw) interruptsDisable() uintptr { return arm.DisableInterrupts() }

// interruptsEnable enables all interrupts using the old system interrupt state
// mask.
//go:inline
func (d *dhw) interruptsEnable(mask uintptr) { arm.EnableInterrupts(mask) }
