// +build stm32h7

package usb

// Implementation of USB device controller hardware abstraction (dhw) for
// STM32H7.

import (
	"device/arm"
	"device/stm32"
	"runtime/interrupt"
)

//   +-- [ NOTE ] --------------------------------------------------------+
//   |                                                                    |
//   |  Often you will see suffixes "HS" and "FS" when referring to core  |
//   |  registers and bitmasks. These are automatically-generated naming  |
//   |  conventions used to distinguish the two USB cores and are not     |
//   |  actually dependent on configured bus speed.                       |
//   |                                                                    |
//   |  "HS" refers to USB PHY/core 1, since it is the only core that is  |
//   |  capable of high-speed mode. "FS" refers to USB PHY/core 2.        |
//   |                                                                    |
//   +--------------------------------------------------------------------+

// deleteCache deletes cached data without touching physical memory. Useful for
// receiving data via DMA, which writes directly to memory, as this will force
// subsequent reads to ignore cache and access physical memory.
func deleteCache(addr, size uintptr) { /* stm32.DeleteDCache(addr, size) */ }

// flushCache immediately flushes cached data to physical memory. Useful for
// transmitting data via DMA, which reads directly from memory, as this will
// immediately flush data currently in cache to physical memory. This also
// purges the data from cache, since we no longer need to access it after
// priming DMA for transmission.
func flushCache(addr, size uintptr) { /* stm32.FlushDeleteDCache(addr, size) */ }

// makeInterrupt returns a new Interrupt using the IRQ number associated with
// the given USB port.
func makeInterrupt(port int) interrupt.Interrupt {
	switch port {
	case 0:
		return interrupt.New(stm32.IRQ_OTG_HS,
			func(interrupt.Interrupt) {
				coreInstance[0].dc.interrupt()
			})
	case 1:
		return interrupt.New(stm32.IRQ_OTG_FS,
			func(interrupt.Interrupt) {
				coreInstance[1].dc.interrupt()
			})
	}
	panic("invalid USB port")
}

// resetInterrupt clears and enables the USB system interrupt.
func (d *dhw) resetInterrupt() {
	m := arm.DisableInterrupts()
	switch d.port {
	case 0:
		m &^= uintptr(stm32.IRQ_OTG_HS)
	case 1:
		m &^= uintptr(stm32.IRQ_OTG_FS)
	}
	arm.EnableInterrupts(m)
}

// enableClocks enables or disables the USB peripheral's core clocks for the
// receiver's configured USB port number and bus speed.
//
// Each USB OTG port clock is gated independently via RCC. Additionally, there
// are separate clock sources/gates for normal and low-power (LP) modes.
// This method enables or disables both the normal and low-power (LP) clock
// gates; it cannot enable or disable one but not the other.
func (d *dhw) enableClocks(enable bool) {

	var resetMask uint32

	switch d.port {

	// USB OTG port 1 supports both high- and full-speed modes
	case 0:

		resetMask = stm32.RCC_AHB1RSTR_USB1OTGRST

		switch d.speed {

		// Configured as either full-speed (FS) or low-speed (LS)
		case LowSpeed, FullSpeed:

			// Always disable ULPI/ULPILP clocks in FS mode
			stm32.RCC.AHB1ENR.ClearBits(stm32.RCC_AHB1ENR_USB1ULPIEN)
			stm32.RCC.AHB1LPENR.ClearBits(stm32.RCC_AHB1LPENR_USB1ULPILPEN)

			// Configure OTG/OTGLP FS clocks
			if enable {
				stm32.RCC.AHB1ENR.SetBits(stm32.RCC_AHB1ENR_USB1OTGEN)
				stm32.RCC.AHB1LPENR.SetBits(stm32.RCC_AHB1LPENR_USB1OTGLPEN)
			} else {
				stm32.RCC.AHB1ENR.ClearBits(stm32.RCC_AHB1ENR_USB1OTGEN)
				stm32.RCC.AHB1LPENR.ClearBits(stm32.RCC_AHB1LPENR_USB1OTGLPEN)
			}

		// Configured as either high-speed (HS) or super-speed (SS [unsupported])
		case HighSpeed, SuperSpeed, DualSuperSpeed:

			// Configure OTG/OTGLP HS and ULPI/ULPILP clocks
			if enable {
				stm32.RCC.AHB1ENR.SetBits(stm32.RCC_AHB1ENR_USB1OTGEN |
					stm32.RCC_AHB1ENR_USB1ULPIEN)
				stm32.RCC.AHB1LPENR.SetBits(stm32.RCC_AHB1LPENR_USB1OTGLPEN |
					stm32.RCC_AHB1LPENR_USB1ULPILPEN)
			} else {
				stm32.RCC.AHB1ENR.ClearBits(stm32.RCC_AHB1ENR_USB1OTGEN |
					stm32.RCC_AHB1ENR_USB1ULPIEN)
				stm32.RCC.AHB1LPENR.ClearBits(stm32.RCC_AHB1LPENR_USB1OTGLPEN |
					stm32.RCC_AHB1LPENR_USB1ULPILPEN)
			}
		}

	// USB OTG port 2 supports full-speed (FS) mode ONLY
	case 1:

		resetMask = stm32.RCC_AHB1RSTR_USB2OTGRST

		// Always disable ULPI/ULPILP clocks in FS mode
		stm32.RCC.AHB1ENR.ClearBits(stm32.RCC_AHB1ENR_USB2ULPIEN)
		stm32.RCC.AHB1LPENR.ClearBits(stm32.RCC_AHB1LPENR_USB2ULPILPEN)

		// Configure OTG/OTGLP FS clocks
		if enable {
			stm32.RCC.AHB1ENR.SetBits(stm32.RCC_AHB1ENR_USB2OTGEN)
			stm32.RCC.AHB1LPENR.SetBits(stm32.RCC_AHB1LPENR_USB2OTGLPEN)
		} else {
			stm32.RCC.AHB1ENR.ClearBits(stm32.RCC_AHB1ENR_USB2OTGEN)
			stm32.RCC.AHB1LPENR.ClearBits(stm32.RCC_AHB1LPENR_USB2OTGLPEN)
		}
	}

	// Reset OTG core clock if enabled
	if enable {
		stm32.RCC.AHB1RSTR.SetBits(resetMask)
		stm32.RCC.AHB1RSTR.ClearBits(resetMask)
		udelay(1000) // Delay 1 ms for clock reset
	}
}
