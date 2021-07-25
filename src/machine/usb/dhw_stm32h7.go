// +build stm32h7

package usb

// Implementation of USB device controller hardware abstraction (dhw) for
// STM32H7.

import (
	"device/arm"
	"device/stm32"
	"runtime/interrupt"
)

const (
	dhwDataQueueSize     = 1024 // Must be a power of 2
	dhwTransferQueueSize = 64   // Must be a power of 2
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

// dhwTimerClass provides an abstraction for using one of the STM32 xTIM timer
// peripherals with the device class timer type dhwTimer.
//
// We use one of the low-power timers (LPTIMx) for class polling, since we need
// only minimal capabilities: input clock selection (with prescalar), 16-bit
// counter, and interrupt on match/overflow. This type of timer can also be used
// to wake the system from various low-power modes, which might prove useful.
type dhwTimerClass stm32.LPTIM_Type

// uartTimer defines the specific timer peripheral (LPTIM1) used by the CDC-ACM
// (single) device class driver. It is embedded in an instance of dhwTimer, our
// timer peripheral abstraction/container.
//
// The purpose of this timer is to flush UART transmit (Tx) data buffered in the
// application's circular queue into the respective USB bulk data endpoint's
// transmit FIFO at regular intervals. This timer-based flushing enables the
// device class driver to better schedule transfers and ensure FIFO overrun
// never occurs.
var uartTimer = &dhwTimer{
	dhwTimerClass: (*dhwTimerClass)(stm32.LPTIM1),
}

func (t *dhwTimer) valid() bool { return nil != t && nil != t.dhwTimerClass }
func (t *dhwTimer) ready() bool { return t.valid() && t.configured }

func (t *dhwTimer) configure(config dhwTimerConfig) *dhwTimer {

	if !t.valid() {
		return nil
	}
	if t.ready() {
		t.reset()
		return t
	}

	// LPTIM1 timer settings:
	//   clock source    = PER[HSI] (64 MHz)
	//   clock prescaler = DIV64 (-> 1 MHz timer)
	//   trigger source  = software
	//   update mode     = immediate
	//   counter source  = internal
	t.CR.Set(0)
	t.CFGR.Set(stm32.LPTIM_CFGR_PRESC_Div64 << stm32.LPTIM_CFGR_PRESC_Pos)
	t.CFGR2.Set(0)
	// interrupt on counter match value in auto-reload register (ARR)
	t.IER.Set(stm32.LPTIM_IER_ARRMIE)
	// enable LPTIM module
	t.CR.Set(stm32.LPTIM_CR_ENABLE)
	// LPTIM is only a 16-bit timer. Since we have scaled the timer frequency to
	// 1 MHz (1 tick/us), the reload register (ARR) conveniently specifies the
	// overflow/interrupt period in terms of microseconds. However, the 16-bit
	// limit means the maximum period is 65.536 milliseconds.
	if config.period > 0xFFFF {
		config.period = 0xFFFF
	}
	t.ARR.Set(config.period - 1)

	// We must manually test which device controllers to invoke interrupt for,
	// because it isn't possible to create different interrupt handlers with the
	// same IRQ. All interrupt handlers defined on the IRQ anywhere in source
	// code, regardless if interrupt.New was actually called or not, will in fact
	// be called when any one of them is called. This is an unintuitive TinyGo
	// design constraint.
	switch t {
	case uartTimer:
		t.irq = interrupt.New(stm32.IRQ_LPTIM1,
			func(interrupt.Interrupt) {
				for _, core := range coreInstance {
					if nil != core.dc {
						core.dc.interruptEnable(false)
						core.dc.tim.clearInterrupts()

						switch core.dc.cc.id {
						case classDeviceCDCACM:
							core.dc.uartTransmit()
						case classDeviceHID:
						}

						core.dc.interruptEnable(true)
					}
				}
			})
	}

	// Initialize LPTIM interrupt with lower priority than USB interrupt.
	t.irq.SetPriority(dhwInterruptPriority + 1)
	// Reset counter and enable interrupt
	t.reset()
	// Enable continuous mode to actually start counting
	t.CR.SetBits(stm32.LPTIM_CR_CNTSTRT)

	t.configured = true
	return t
}

func (t *dhwTimer) clearInterrupts() {
	if !t.valid() {
		return
	}
	t.ICR.Set(stm32.LPTIM_ICR_ARROKCF | stm32.LPTIM_ICR_CMPOKCF |
		stm32.LPTIM_ICR_ARRMCF | stm32.LPTIM_ICR_CMPMCF)
}

func (t *dhwTimer) reset() {
	if !t.valid() {
		return
	}
	t.irq.Disable()
	// From user manual:
	//  | Caution: COUNTRST must never be set to '1' by software before it is
	//  | already cleared to '0' by hardware. Software should consequently check
	//  | that COUNTRST bit is already cleared to '0' before attempting to set it
	//  | to '1'.
	cr := t.CR.Get()
	if cr&stm32.LPTIM_CR_COUNTRST == 0 {
		t.CR.Set(cr | stm32.LPTIM_CR_COUNTRST)
		for t.CR.HasBits(stm32.LPTIM_CR_COUNTRST) {
		} // wait for reset to complete
	}
	t.clearInterrupts()
	t.irq.Enable()
}
