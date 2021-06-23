// +build stm32

package usb

// Common definitions for the USB host controller hardware abstraction (hhw) for
// all STM32 devices.

import (
	"device/stm32"
	"runtime/interrupt"
	"runtime/volatile"
)

// hhwInterruptPriority defines the priority for all USB host interrupts.
const hhwInterruptPriority = 3

// hhw implements USB host controller hardware abstraction interface.
type hhw struct {
	*hcd // USB host controller driver

	glo *stm32.USB_GLOBAL_Type // USB global registers
	dev *stm32.USB_DEVICE_Type // USB device registers
	pcc *volatile.Register32   // USB PWR/CLKCTL register

	irq interrupt.Interrupt

	speed Speed
}

// allocHHW returns a reference to the USB hardware abstraction for the given
// host controller driver. Should be called only one time and during host
// controller initialization.
func allocHHW(port, instance int, speed Speed, hc *hcd) *hhw {
	switch port {
	case 0:
		hhwInstance[instance].hcd = hc
		hhwInstance[instance].glo = stm32.USB_GLOBAL1
		hhwInstance[instance].dev = stm32.USB_DEVICE1
		hhwInstance[instance].pcc = stm32.USB_PCCTRL1

	case 1:
		hhwInstance[instance].hcd = hc
		hhwInstance[instance].glo = stm32.USB_GLOBAL2
		hhwInstance[instance].dev = stm32.USB_DEVICE2
		hhwInstance[instance].pcc = stm32.USB_PCCTRL2
	}

	// All ports default to full-speed during initialization. An interrupt event
	// is raised if/when a USB port connect/status change occurs in which the
	// host signals a different speed is to be used. At that point, this speed
	// field will be updated accordingly.
	if 0 == speed {
		speed = FullSpeed
	}
	hhwInstance[instance].speed = speed

	return &hhwInstance[instance]
}

// init configures the USB port for host mode operation by initializing all
// endpoint and transfer descriptor data structures, initializing core registers
// and interrupts, resetting the USB PHY, and enabling power on the bus.
func (h *hhw) init() status {

	return statusOK
}

// enable causes the USB core to enter (or exit) the normal run state and
// enables/disables all interrupts on the receiver's USB port.
func (h *hhw) enable(enable bool) {
	if enable {
		h.irq.Enable() // Enable USB interrupts
	} else {
		h.irq.Disable() // Disable USB interrupts
	}
}
