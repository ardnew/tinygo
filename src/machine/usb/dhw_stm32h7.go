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

// microTickTimer provides an abstraction for using one of the STM32 xTIM timer
// peripherals as a general purpose ticker (separate from the core's SysTick
// timer). The timer frequency is fixed at 1 MHz (1 tick/microsec).
type microTickTimer struct {
	*microTickBus
	irq        interrupt.Interrupt
	tick       func()
	configured bool
}

type microTickConfig struct {
	period   uint32
	priority uint8
	tick     func()
}

// microTickBus defines the hardware peripheral type used by microTickTimer.
//
// We use one of the low-power timers (LPTIMx), because we need only minimal
// capabilities: input clock selection (with prescalar), 16-bit counter, and
// interrupt on match/overflow. This type of timer can also be used to wake the
// system from various low-power modes, which might prove useful.
//
// Since the overflow counter is 16-bit, the longest tick period that can be
// generated is 65.536 millisecs.
type microTickBus stm32.LPTIM_Type

var microTick = &microTickTimer{microTickBus: (*microTickBus)(stm32.LPTIM1)}

func (t *microTickTimer) valid() bool { return nil != t && nil != t.microTickBus }
func (t *microTickTimer) ready() bool { return t.valid() && t.configured }

func (t *microTickTimer) configure(config microTickConfig) *microTickTimer {

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

	if config.period > 0xFFFF {
		config.period = 0xFFFF
	}
	t.ARR.Set(config.period - 1)

	t.tick = config.tick

	switch t {
	case microTick:
		t.irq = interrupt.New(stm32.IRQ_LPTIM1,
			func(interrupt.Interrupt) {
				if nil != microTick.tick {
					microTick.tick()
				}
			})
	}

	t.irq.SetPriority(config.priority)
	// Reset counter and enable interrupt
	t.reset()
	// Enable continuous mode to actually start counting
	t.CR.SetBits(stm32.LPTIM_CR_CNTSTRT)

	t.configured = true
	return t
}

func (t *microTickTimer) clearInterrupts() {
	if !t.valid() {
		return
	}
	t.ICR.Set(stm32.LPTIM_ICR_ARROKCF | stm32.LPTIM_ICR_CMPOKCF |
		stm32.LPTIM_ICR_ARRMCF | stm32.LPTIM_ICR_CMPMCF)
}

func (t *microTickTimer) reset() {
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
