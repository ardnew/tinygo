// +build nrf52840

package usb

// Implementation of USB device controller hardware abstraction (dhw) for Nordic
// nRF52840.

import (
	"device/arm"
	"device/nrf"
	"math/bits"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

// dcdInterruptPriority defines the priority for all USB device interrupts.
const dcdInterruptPriority = 3

// dhw implements USB device controller hardware abstraction for nRF52840.
type dhw struct {
	drv *dcd                // USB device controller driver
	bus *nrf.USBD_Type      // USB core register
	irq interrupt.Interrupt // USB IRQ, only a single interrupt on nRF52840

	speed uint8 // bus speed (0=full, 1=low, 2=high, 4=super)

	critical volatile.Register32
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
		dhwInstance[d.id].bus = nrf.USBD
		dhwInstance[d.id].irq =
			interrupt.New(nrf.IRQ_USBD,
				func(interrupt.Interrupt) {
					coreInstance[0].dc.hw.event()
				})
	default:
		return nil // Invalid USB port number
	}

	// nRF52840 supports full-speed mode (12 Mbit/sec) only
	dhwInstance[d.id].speed = descDeviceSpeedFull

	return &dhwInstance[d.id]
}

// init configures the USB hardware for device mode operation by resetting the
// USB PHY, enabling power on the bus, and initializing core registers and
// interrupts.
func (d *dhw) init() status {

	// Ensure USB pull-up is disabled prior to enabling PHY power supply.
	d.bus.USBPULLUP.ClearBits(nrf.USBD_USBPULLUP_CONNECT)

	// Source VBUS (if available) to USB PHY. Note that this only enables the PHY
	// power supply, but it does not yet begin USB enumeration, which occurs when
	// VBUS supply is stable (per USBEVENT) and pull-up is enabled via software.
	// See:                                      -- 6.35.4: USBD power-up sequence
	//                                       nRF52840 Product Specification (v1.0)
	d.bus.ENABLE.SetBits(nrf.USBD_ENABLE_ENABLE)

	// Enable interrupts in USB core
	d.bus.INTENSET.Set(
		nrf.USBD_INTENSET_EPDATA |
			nrf.USBD_INTENSET_EP0DATADONE |
			nrf.USBD_INTENSET_USBEVENT |
			nrf.USBD_INTENSET_SOF |
			nrf.USBD_INTENSET_EP0SETUP,
	)

	// Ensure D+ pulled down long enough for host to detect previous disconnect
	udelay(5000)

	return statusOK
}

// enable causes the USB core to enter (or exit) the normal run state.
func (d *dhw) enable(enable bool) {
	// On nRF52840, there isn't a concept of "run state", and using USBD.ENABLE
	// will reset all registers and clear the current device configuration.
	//
	// Instead we emulate a physical device disconnect via USBD.USBPULLUP, which
	// will halt all traffic until the pull-up is re-enabled. Upon re-enabling,
	// the device will perform bus enumeration again as if a new connection is
	// being established, but also retains current device configuration and
	// endpoints.
	if enable {
		d.bus.USBPULLUP.SetBits(nrf.USBD_USBPULLUP_CONNECT)
	} else {
		d.bus.USBPULLUP.ClearBits(nrf.USBD_USBPULLUP_CONNECT)
	}
}

// event handles the USB hardware interrupts events and notifies the device
// controller driver using a common "virtual interrupt" code.
func (d *dhw) event() {

	// Handle start of frame (SOF) interrupt
	sof := d.bus.EVENTS_SOF.Get()
	d.bus.EVENTS_SOF.Set(sof) // Clear SOF interrupt
	if 0 != sof&nrf.USBD_EVENTS_SOF_EVENTS_SOF {
		// TODO: flush writes
		// TODO: Blink LED to show USB traffic
	}

	// Handle events and errors not reserved by specific event registers. The
	// source of this interrupt must be obtained from EVENTCAUSE register.
	event := d.bus.EVENTS_USBEVENT.Get()
	d.bus.EVENTS_USBEVENT.Set(event) // Clear all event/error interrupts
	if 0 != event&nrf.USBD_EVENTS_USBEVENT_EVENTS_USBEVENT {
		cause := d.bus.EVENTCAUSE.Get() // Determine cause for event interrupt
		d.bus.EVENTCAUSE.Set(cause)     // Clear all flagged event causes
		for 0 != cause {
			// Handle each flagged cause in order (LSB first, MSB last)
			b := uint32(1) << bits.TrailingZeros32(cause)
			switch b {
			// USB device is ready for normal operations
			case nrf.USBD_EVENTCAUSE_READY:
				// Notify device controller driver the USB PHY is up and running, and we
				// are now ready to begin normal operation.
				d.drv.event(dcdEvent{
					id: dcdEventPeripheralReady,
				})
			// USB resume
			case nrf.USBD_EVENTCAUSE_RESUME:
			// USB suspend
			case nrf.USBD_EVENTCAUSE_SUSPEND:
			// USB wake-up allowed
			case nrf.USBD_EVENTCAUSE_USBWUALLOWED:
			}
			cause &^= b
		}
	}

	// Handle setup requests on control endpoint 0.
	setup := d.bus.EVENTS_EP0SETUP.Get()
	d.bus.EVENTS_EP0SETUP.Set(setup) // Clear setup interrupt
	if 0 != setup&nrf.USBD_EVENTS_EP0SETUP_EVENTS_EP0SETUP {
		// Notify device controller driver using the setup data constructed from the
		// USBD peripheral's dedicated registers.
		d.drv.event(dcdEvent{
			id: dcdEventControlSetup,
			setup: dcdSetup{
				bmRequestType: uint8(nrf.USBD.BMREQUESTTYPE.Get()),
				bRequest:      uint8(nrf.USBD.BREQUEST.Get()),
				wValue:        uint16((nrf.USBD.WVALUEH.Get() << 8) | nrf.USBD.WVALUEL.Get()),
				wIndex:        uint16((nrf.USBD.WINDEXH.Get() << 8) | nrf.USBD.WINDEXL.Get()),
				wLength:       uint16((nrf.USBD.WLENGTHH.Get() << 8) | nrf.USBD.WLENGTHL.Get()),
			},
		})
	}

	done0 := d.bus.EVENTS_EP0DATADONE.Get()
	d.bus.EVENTS_EP0DATADONE.Set(done0)

	start := d.bus.EVENTS_STARTED.Get()
	d.bus.EVENTS_STARTED.Set(start)
	// endIn := d.bus.EVENTS_ENDEPIN.Get()
	// d.bus.EVENTS_ENDEPIN.Set(endIn)
	// endOut := d.bus.EVENTS_ENDEPOUT.Get()
	// d.bus.EVENTS_ENDEPOUT.Set(endOut)

	data := d.bus.EVENTS_EPDATA.Get()
	d.bus.EVENTS_EPDATA.Set(data)

}

func (d *dhw) criticalSection(enter bool) {
	if enter {
		for 0 != d.critical.Get() {
			arm.Asm("wfi")
		}
		d.critical.Set(1)
	} else {
		d.critical.Set(0)
	}
}

func (d *dhw) deleteCache(addr, size uintptr) {
	// TBD
}

func (d *dhw) flushCache(addr, size uintptr) {
	// TBD
}

func (d *dhw) controlSpeed() uint8 { return d.speed }

func (d *dhw) controlDeviceAddress(addr uint16) {
	// Handled internally on nRF52840 by USBD peripheral:
	//   | The USBD peripheral handles the SetAddress transfer by itself. As a
	//   | consequence, the software shall not process this command other than
	//   | updating its state machine (see Device state diagram), nor initiate a
	//   | status stage. If necessary, the address assigned by the host can be
	//   | read out from the USBADDR register after the command has been
	//   | processed.
	//                                                -- 6.35.9: Control transfers
	//                                       nRF52840 Product Specification (v1.0)
	return
}

func (d *dhw) controlLineState(coding *descCDCACMLineCoding, dtr, rts bool) {

}

func (d *dhw) controlLineCoding(coding *descCDCACMLineCoding) {

}

// controlStatus transitions transfers on control endpoint 0 into status stage.
func (d *dhw) controlStatus() {
	d.bus.TASKS_EP0STATUS.Set(nrf.USBD_TASKS_EP0STATUS_TASKS_EP0STATUS)
}

// controlStall stalls a transfer on control endpoint 0. To stall a transfer on
// any other endpoint, use method endpointStall().
func (d *dhw) controlStall() {
	d.bus.TASKS_EP0STALL.Set(nrf.USBD_TASKS_EP0STALL_TASKS_EP0STALL)
}

func (d *dhw) endpointEnable(endpoint uint8, control bool, config uint32) {
	num, _ := unpackEndpoint(endpoint)
	if control {
		d.bus.EPINEN.SetBits(nrf.USBD_EPINEN_IN0 << num)
		d.bus.EPOUTEN.SetBits(nrf.USBD_EPOUTEN_OUT0 << num)
		d.bus.INTENSET.Set(nrf.USBD_INTENSET_ENDEPOUT0 << num)
	} else {
		switch config & descEndptConfigAttrRxMsk {
		case descEndptConfigAttrRxIsochronous:
			d.bus.SIZE.ISOOUT.Set(0)
			d.bus.EPOUTEN.SetBits(nrf.USBD_EPOUTEN_ISOOUT)
			d.bus.INTENSET.Set(nrf.USBD_INTENSET_ENDISOOUT)
		case descEndptConfigAttrRxBulk, descEndptConfigAttrRxInterrupt:
			d.bus.SIZE.EPOUT[num].Set(0)
			d.bus.EPOUTEN.SetBits(nrf.USBD_EPOUTEN_OUT0 << num)
			d.bus.INTENSET.Set(nrf.USBD_INTENSET_ENDEPOUT0 << num)
		}
		switch config & descEndptConfigAttrTxMsk {
		case descEndptConfigAttrTxIsochronous:
			d.bus.EPINEN.SetBits(nrf.USBD_EPINEN_ISOIN)
		case descEndptConfigAttrTxBulk, descEndptConfigAttrTxInterrupt:
			d.bus.EPINEN.SetBits(nrf.USBD_EPINEN_IN0 << num)
		}
	}
	// Endpoints are only enabled in response to a setup request. So, always
	// trigger a transition into the status stage.
	d.bus.TASKS_EP0STATUS.Set(nrf.USBD_TASKS_EP0STATUS_TASKS_EP0STATUS)
}

func (d *dhw) endpointStatus(endpoint uint8) uint16 {
	// The HALTED registers can be returned verbatim (no masking/shifting) for a
	// given endpoint:
	//   | The halted (or not) state of a given endpoint can be read back from
	//   | register HALTED.EPIN[n] or HALTED.EPOUT[n]. The format of the returned
	//   | 16-bit value can be copied as is as response to a GetStatusEndpoint
	//   | request from the host.
	//   |   [...]
	//   | The control endpoint 0 IN and OUT can also be enabled and/or halted
	//   | using the same mechanisms, but due to USB specification, receiving a
	//   | SETUP will override its state.
	//                                 -- 6.35.10: Bulk and interrupt transactions
	//                                       nRF52840 Product Specification (v1.0)
	switch num, dir := unpackEndpoint(endpoint); dir {
	case nrf.USBD_EPSTALL_IO_Out:
		return uint16(d.bus.HALTED.EPOUT[num].Get() &
			nrf.USBD_HALTED_EPOUT_GETSTATUS_Msk)
	case nrf.USBD_EPSTALL_IO_In:
		return uint16(d.bus.HALTED.EPIN[num].Get() &
			nrf.USBD_HALTED_EPIN_GETSTATUS_Msk)
	}
	return 0
}

// endpointStall stalls a transfer on the given endpoint. To stall transfers on
// control endpoint 0, use method controlStall().
func (d *dhw) endpointStall(endpoint uint8) {
	// TBD: Control endpoint 0 has a special, dedicated TASK register for stall,
	//      USBD.TASKS_EP0STALL. Do we need to use this? Or can we use the general
	//      EPSTALL register used by bulk/interrupt endpoints?
	//
	// The controlStall() method stalls transfers on control endpoint 0 (only)
	// using the dedicated TASK register.
	d.bus.EPSTALL.Set(uint32(endpoint) | nrf.USBD_EPSTALL_STALL)
}

func (d *dhw) endpointPrime(mask uint32, transfer *dcdTransfer) {

	rm := uint16(mask >> descEndptConfigAttrRxPos)
	tm := uint16(mask >> descEndptConfigAttrTxPos)
	if 0 != rm {
		// ep := bits.TrailingZeros16(rm)
	} else if 0 != tm {
		ep := bits.TrailingZeros16(tm)
		sz := (transfer.token >> 16) & 0x7FFF
		d.bus.EPIN[ep].PTR.Set(
			uint32(uintptr(unsafe.Pointer(transfer.pointer[0]))),
		)
		d.bus.EPIN[ep].MAXCNT.Set(sz)
		d.bus.TASKS_STARTEPIN[ep].Set(nrf.USBD_TASKS_STARTEPIN_TASKS_STARTEPIN)
	}
}

func (d *dhw) endpointPrimed() uint32 {
	return d.bus.EPSTATUS.Get()
}

func (d *dhw) endpointUnprime(mask uint32) {
	// TBD
}

// interruptsEnableUSB enables or disables all interrupts associated with the
// receiver's USB port.
//go:inline
func (d *dhw) interruptsEnableUSB(enable bool) {
	// nRF52840 has only a single interrupt vector per USB core.
	if enable {
		d.irq.SetPriority(dcdInterruptPriority)
		d.irq.Enable()
	} else {
		d.irq.Disable()
	}
}
