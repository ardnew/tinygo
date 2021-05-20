// +build stm32h7x7

package runtime

import (
	"device/stm32"
	"machine"
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

const (
	semStopMode = 4 // semaphore index: Stop Mode
	semRCC      = 3 // semaphore index: RCC
	semFLASH    = 2 // semaphore index: FLASH
	semPKA      = 1 // semaphore index: PKA
	semRNG      = 0 // semaphore index: RNG
	semGPIO     = 5 // semaphore index: GPIO
)

func initSync() {

	// SEVONPEND enabled so that an interrupt coming from the CPU(n) interrupt
	// signal is detectable by the CPU after a WFI/WFE instruction.
	stm32.SCB.SCR.SetBits(stm32.SCB_SCR_SEVEONPEND)

	// Prepare core, install vector table
	preinitCore()

	// Copy data/bss sections from flash to RAM
	preinit()

	// Initialize active core
	initCore()
}

func initClocks() {

	// Initialize SysTick with timebase out of reset
	freq := stm32.RCC.ClockFreq()
	setCoreFreq(freq)
	initSysTick(freq.SYSCLK)

	_ = stm32.RCC.Enable(unsafe.Pointer(stm32.SYSCFG), true)
	_ = stm32.RCC.EnableGPIO()

	initLowSpeedCrystal(lseDriveLow)

	// initialize SYSCLK (PLL1 with HSE)
	initSYSCLK(true, false)

	// re-initialize SysTick with increased frequencies
	freq = stm32.RCC.ClockFreq()
	setCoreFreq(freq)
	initSysTick(freq.SYSCLK)

	// iniitialize core clock
	initCoreClock()
}

func initPeripherals() {

}

const (
	lseDriveLow = uint32(iota) << stm32.RCC_BDCR_LSEDRV_Pos
	lseDriveMedLow
	lseDriveMedHigh
	lseDriveHigh
)

func initLowSpeedCrystal(drive uint32) {
	if !stm32.RCC.BDCR.HasBits(stm32.RCC_BDCR_LSERDY) {
		// LSE is in the backup domain (DBP) and write access is denied to DBP after
		// reset, so we first have to enable write access for DBP before configuring.
		stm32.PWR.PWR_CR1.SetBits(stm32.PWR_PWR_CR1_DBP)
		if (lseDriveMedLow <= drive && drive <= lseDriveMedHigh) &&
			stm32.DBG.MCURevision() <= stm32.DBG_MCU_REVISION_Y {
			drive = ^drive & stm32.RCC_BDCR_LSEDRV_Msk
		}
		stm32.RCC.BDCR.ReplaceBits(drive, stm32.RCC_BDCR_LSEDRV_Msk, 0)
	}
}

func initSYSCLK(bypass, lowSpeed bool) bool {

	// First need to change SYSCLK source to CSI before modifying the main PLL
	if stm32.RCC_PLL_SRC_HSE == (stm32.RCC.PLLCKSELR.Get()&
		stm32.RCC_PLLCKSELR_PLLSRC_Msk)>>stm32.RCC_PLLCKSELR_PLLSRC_Pos {
		if !stm32.RCC.ConfigClk(stm32.RCC_CLK_CONFIG{
			CLK:    stm32.RCC_CLK_SYSCLK,
			SYSSrc: stm32.RCC_SYSCLK_SRC_CSI,
		}, stm32.FLASH_ACR_LATENCY_1WS) {
			return false
		}
	}

	// Enable oscillator pin
	oscPin := machine.PH01
	oscPin.Configure(machine.PinConfig{
		Mode: machine.PinOutputPushPull | machine.PinPullUp | machine.PinLowSpeed,
	})
	oscPin.Set(true)

	// Configure main internal regulator voltage for increased CPU frequencies.
	scale := uint32(stm32.PWR_REGULATOR_VOLTAGE_SCALE1)
	if lowSpeed {
		scale = stm32.PWR_REGULATOR_VOLTAGE_SCALE3
	}
	if !stm32.PWR.Configure(stm32.PWR_SMPS_1V8_SUPPLIES_LDO, scale) {
		return false
	}

	for !stm32.PWR_FLAG_VOSRDY.Get() {
	} // wait for voltage to stabilize

	// Enable HSE oscillator and activate PLL with HSE as source
	osc := stm32.RCC_OSC_CONFIG{
		OSC:   stm32.RCC_OSC_HSE | stm32.RCC_OSC_HSI48,
		HSE:   stm32.RCC_HSE_ON,
		HSI48: stm32.RCC_HSI48_ON,
		PLL: stm32.RCC_PLL_CONFIG{
			State: stm32.RCC_PLL_ON,
			Src:   stm32.RCC_PLL_SRC_HSE,
			Param: stm32.RCC_PLL_PARAM{
				M:    5,
				N:    160,
				Frac: 0,
				P:    2,
				Q:    10,
				R:    2,
				In:   stm32.RCC_PLL1_VCI_RANGE_2,
				Out:  stm32.RCC_PLL1_VCO_WIDE,
			},
		},
	}
	if bypass {
		osc.HSE = stm32.RCC_HSE_BYPASS
	}
	if lowSpeed {
		osc.PLL.Param.N = 40
	}

	if !stm32.RCC.ConfigOsc(osc) {
		return false
	}

	latency := uint32(stm32.FLASH_ACR_LATENCY_4WS)
	if lowSpeed {
		latency = stm32.FLASH_ACR_LATENCY_0WS
	}

	if !stm32.RCC.ConfigClk(
		stm32.RCC_CLK_CONFIG{
			CLK: stm32.RCC_CLK_SYSCLK | stm32.RCC_CLK_HCLK |
				stm32.RCC_CLK_PCLK1 | stm32.RCC_CLK_PCLK2 |
				stm32.RCC_CLK_D1PCLK1 | stm32.RCC_CLK_D3PCLK1,
			SYSSrc:  stm32.RCC_SYSCLK_SRC_PLL,
			SYSDiv:  stm32.RCC_SYSCLK_DIV1,
			AHBDiv:  stm32.RCC_HCLK_DIV2,
			APB1Div: stm32.RCC_APB1_DIV2,
			APB2Div: stm32.RCC_APB2_DIV2,
			APB3Div: stm32.RCC_APB3_DIV2,
			APB4Div: stm32.RCC_APB4_DIV2,
		}, latency) {
		return false
	}

	// TODO: enable USB regulator, VBUS detection

	return true
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
