// +build stm32h7x7

package runtime

import (
	"device/stm32"
	"runtime/volatile"
	"unsafe"
)

// ----------+-----------
//   SYSCLK  |  480 MHz
//   AHBCLK  |  240 MHz
//  APB1CLK  |  120 MHz
//  APB2CLK  |  120 MHz
//  APB3CLK  |  120 MHz
//  APB4CLK  |  120 MHz
// ----------+-----------
//      HSE  |   25 MHz
//      CSI  |    4 MHz
//      HSI  |   64 MHz
// ----------+-----------
//      USB  |   48 MHz
// ----------+-----------

// Hardware semaphore index
type peripheralLock uint8

const (
	lockRNG peripheralLock = iota
	lockPKA
	lockFLASH
	lockRCC
	lockStopMode
	lockGPIO
)

func initSystem() {

	// SEVONPEND enabled so that an interrupt coming from the CPU(n) interrupt
	// signal is detectable by the CPU after a WFI/WFE instruction.
	stm32.SCB.SCR.SetBits(stm32.SCB_SCR_SEVEONPEND)

	// Prepare current core, reset bus clocks
	resetCore()

	// Initialize active core, RAM, VTOR, HSEM
	initCore()
}

func initClocks() {

	// Use the driver core (M7) to initialize core bus frequencies
	freq := initCoreFreq()

	// Initialize SysTick with new core frequency
	initSysTick(freq)
}

const (
	tickPriority = 16   // NVIC priority number
	tickFreqHz   = 1000 // 1 kHz
	nsecsPerTick = 1000000000 / tickFreqHz
)

var (
	tickCount  volatile.Register64
	cycleCount volatile.Register32
)

var (
	// We use the ARMv7-M Debug facilities for counting CPU cycles, specifically
	// the Data Watchpoint and Trace Unit (DWT), register CYCCNT.
	// Usage of these capabilities is specified in the ARMv7-M Architecture
	// Reference Manual (https://developer.arm.com/documentation/ddi0403/latest/),
	// in the chapters indicated below.

	// C1.6.5 Debug Exception and Monitor Control Register, DEMCR
	DEM_CR = (*volatile.Register32)(unsafe.Pointer(uintptr(0xE000EDFC)))
	// C1.8.7 Control register, DWT_CTRL
	DWT_CR = (*volatile.Register32)(unsafe.Pointer(uintptr(0xE0001000)))
	// C1.8.8 Cycle Count register, DWT_CYCCNT
	DWT_CYCCNT = (*volatile.Register32)(unsafe.Pointer(uintptr(0xE0001004)))
)

func initSysTick(coreFreqHz uint32) {

	// Determine counter top which will cause rollover when the source clock (our
	// MCU core frequency) has cycled as many times as desired SysTick frequency.
	top := coreFreqHz/tickFreqHz - 1

	if top > stm32.STK_RVR_RELOAD_Msk {
		return // invalid tick count
	}

	// Disable SysTick before reconfiguring.
	stm32.STK.CSR.ClearBits(stm32.STK_CSR_ENABLE)

	tickCount.Set(0)
	stm32.STK.RVR.Set(top)
	stm32.STK.CVR.Set(0)
	// Enable SysTick IRQ and SysTick Timer, use internal (core) clock source
	stm32.STK.CSR.Set(stm32.STK_CSR_CLKSOURCE_Msk |
		stm32.STK_CSR_TICKINT_Msk | stm32.STK_CSR_ENABLE_Msk)

	// set SysTick and PendSV priority to 32
	stm32.SCB.SHPR3.Set((0x20 << stm32.SCB_SHPR3_PRI_15_Pos) |
		(0x20 << stm32.SCB_SHPR3_PRI_14_Pos))

	// turn on cycle counter
	DEM_CR.SetBits(0x01000000) // enable debugging & monitoring blocks
	DWT_CR.SetBits(0x00000001) // cycle count register
	cycleCount.Set(DWT_CYCCNT.Get())
}

//go:export SysTick_Handler
func tick() {
	tickCount.Set(tickCount.Get() + 1)
	cycleCount.Set(DWT_CYCCNT.Get())
}

func ticksToNanoseconds(t timeUnit) int64 { return int64(t) * nsecsPerTick }
func nanosecondsToTicks(n int64) timeUnit { return timeUnit(n / nsecsPerTick) }

// number of ticks (microseconds) since start.
//go:linkname ticks runtime.ticks
func ticks() timeUnit { return timeUnit(tickCount.Get()) }

// current CPU cycle count reported by Cortex-M DWT unit
//go:linkname ticks runtime.cycles
func cycles() uint32 { return cycleCount.Get() }

func sleepTicks(d timeUnit) {}
