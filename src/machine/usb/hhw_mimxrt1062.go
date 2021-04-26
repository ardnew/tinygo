// +build mimxrt1062

package usb

// Implementation of USB host controller driver (hcd) for NXP iMXRT1062.

import (
	"device/nxp"
	"runtime/interrupt"
)

// hcdInterruptPriority defines the priority for all USB host interrupts.
const hcdInterruptPriority = 3

// hcd implements USB host controller driver (hcd) interface.
type hhw struct {
	drv *hcd                // USB host controller driver
	bus *nxp.USB_Type       // USB core register
	phy *nxp.USBPHY_Type    // USB PHY register
	irq interrupt.Interrupt // USB IRQ, only a single interrupt on iMXRT1062
}

// allocHHW returns a reference to the USB hardware abstraction for the given
// host controller driver. Should be called only one time and during host
// controller initialization.
func allocHHW(h *hcd) *hhw {
	switch h.port {
	case 0:
		hhwInstance[h.id].drv = h
		hhwInstance[h.id].bus = nxp.USB1
		hhwInstance[h.id].phy = nxp.USBPHY1
		//		hhwInstance[h.id].irq =
		//			interrupt.New(nxp.IRQ_USB_OTG1,
		//				func(interrupt.Interrupt) {
		//					coreInstance[0].dc.hw.event()
		//				})

	case 1:
		hhwInstance[h.id].drv = h
		hhwInstance[h.id].bus = nxp.USB2
		hhwInstance[h.id].phy = nxp.USBPHY2
		//		hhwInstance[h.id].irq =
		//			interrupt.New(nxp.IRQ_USB_OTG2,
		//				func(interrupt.Interrupt) {
		//					//coreInstance[1].dc.hw.event()
		//				})
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
