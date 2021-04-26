// +build nrf52840

package usb

// Implementation of USB host controller driver (hcd) for Nordic nRF52840.

import (
	"device/nrf"
	"runtime/interrupt"
)

// hcdInterruptPriority defines the priority for all USB host interrupts.
const hcdInterruptPriority = 3

// hcd implements USB host controller driver (hcd) interface.
type hhw struct {
	drv *hcd                // USB host controller driver
	bus *nrf.USBD_Type      // USB core register
	irq interrupt.Interrupt // USB IRQ
}

// allocHHW returns a reference to the USB hardware abstraction for the given
// host controller driver. Should be called only one time and during host
// controller initialization.
func allocHHW(h *hcd) *hhw {
	switch h.port {
	case 0:
		hhwInstance[h.id].drv = h
		hhwInstance[h.id].bus = nrf.USBD

	case 1:
		hhwInstance[h.id].drv = h
		hhwInstance[h.id].bus = nrf.USBD
	}

	return &hhwInstance[h.id]
}

// init configures the USB hardware for host mode operation by resetting the
// USB PHY, enabling power on the bus, and initializing core registers and
// interrupts.
func (h *hhw) init() status {

	return statusOK
}

// enable causes the USB core to enter (or exit) the normal run state.
func (h *hhw) enable(enable bool) {

}
