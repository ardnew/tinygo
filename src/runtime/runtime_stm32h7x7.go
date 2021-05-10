// +build stm32h7x7

package runtime

import (
	"device/arm"
	"device/stm32"
	"runtime/interrupt"
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
	hseFreqHz uint32 = 25000000 // External oscillator (Hz)
	csiFreqHz uint32 = 4000000  // Internal oscillator (Hz)
	hsiFreqHz uint32 = 64000000 // Internal oscillator (Hz)
)

const (
	pllNone       = 1 << iota
	pllHSI        // Use HSI internal clock
	pllHSECrystal // Use external xtal (x3 on-board)
	pllHSEClock   // Use external clock
)

var (
	coreD1FreqHz uint32 = 64000000 // System core frequency, depending on target
	coreD2FreqHz uint32 = 64000000 // Second core frequency

	// System core frequency prescalar table
	coreD1FreqPresc = [...]uint8{
		0, 0, 0, 0, 1, 2, 3, 4, 1, 2, 3, 4, 6, 7, 8, 9,
	}
)

const (
	semStopMode = 4 // Index of the semaphore used to manage the entry Stop Mode procedure
	semRCC      = 3 // Index of the semaphore used to access the RCC
	semFLASH    = 2 // Index of the semaphore used to access the FLASH
	semPKA      = 1 // Index of the semaphore used to access the PKA
	semRNG      = 0 // Index of the semaphore used to access the RNG
	semGPIO     = 5 // Index of the semaphore used to access GPIO
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

	d1 := sysClockFreq() >>
		((coreD1FreqPresc[(stm32.RCC.D1CFGR.Get()&stm32.RCC_D1CFGR_D1CPRE_Msk)>>
			stm32.RCC_D1CFGR_D1CPRE_Pos]) & 0x1F)
	d2 := (d1 >>
		((coreD1FreqPresc[(stm32.RCC.D1CFGR.Get()&stm32.RCC_D1CFGR_HPRE_Msk)>>
			stm32.RCC_D1CFGR_HPRE_Pos]) & 0x1F))

	setCoreFreq(d1, d2)

	initSysTick(d1)

	// use PLL (with external HSE) as system clock source

	// iniitialize core clock
	initCoreClock()
}

func initCache() {
	enableICache(true)
	enableDCache(true)
}

var semaphoreEnabled volatile.Register32

func initSemaphore() {
	// Enable hardware semaphore (HSEM) peripheral clock
	stm32.RCC.AHB4ENR.SetBits(stm32.RCC_AHB4ENR_HSEMEN)
	if stm32.RCC.AHB4ENR.HasBits(stm32.RCC_AHB4ENR_HSEMEN) {
		semaphoreEnabled.Set(1)
	} else {
		semaphoreEnabled.Set(0)
	}
}

func initPeripherals() {

}

const (
	icacheEnablePos   = 17                     // SCB CCR: IC Position
	icacheEnableMsk   = (1 << icacheEnablePos) // SCB CCR: IC Mask
	dcacheEnablePos   = 16                     // SCB CCR: DC Position
	dcacheEnableMsk   = (1 << dcacheEnablePos) // SCB CCR: DC Mask
	dcacheWayPos      = 30                     // SCB DCISW: WAY Position
	dcacheWayMsk      = 3 << dcacheWayPos      // SCB DCISW: WAY Mask
	dcacheSetPos      = 5                      // SCB DCISW: SET Position
	dcacheSetMsk      = 0x1FF << dcacheSetPos  // SCB DCISW: SET Mask
	dccacheWayPos     = 30                     // SCB DCCISW: WAY Position
	dccacheWayMsk     = 3 << dccacheWayPos     // SCB DCCISW: WAY Mask
	dccacheSetPos     = 5                      // SCB DCCISW: SET Position
	dccacheSetMsk     = 0x1FF << dccacheSetPos // SCB DCCISW: SET Mask
	dccacheAssocPos   = 0x3                    // SCB DCCISW: ASSOCIATIVITY Position
	dccacheAssocMsk   = 0x1FF8                 // SCB DCCISW: ASSOCIATIVITY Mask
	dccacheNumSetsPos = 0xD                    // SCB DCCISW: NUMSETS Position
	dccacheNumSetsMsk = 0xFFFE000              // SCB DCCISW: NUMSETS Mask
)

var (
	dccache    volatile.Register32
	dcacheSets volatile.Register32
	dcacheWays volatile.Register32

	// Offset: 0x080 (R/ )  Cache Size ID Register
	CCSIDR = (*volatile.Register32)(unsafe.Pointer((uintptr(unsafe.Pointer(stm32.SCB)) + 0x080)))
	// Offset: 0x084 (R/W)  Cache Size Selection Register
	CSSELR = (*volatile.Register32)(unsafe.Pointer((uintptr(unsafe.Pointer(stm32.SCB)) + 0x084)))
	// Offset: 0x250 ( /W)  I-Cache Invalidate All to PoU
	ICIALLU = (*volatile.Register32)(unsafe.Pointer((uintptr(unsafe.Pointer(stm32.SCB)) + 0x250)))
	// Offset: 0x260 ( /W)  D-Cache Invalidate by Set-way
	DCISW = (*volatile.Register32)(unsafe.Pointer((uintptr(unsafe.Pointer(stm32.SCB)) + 0x260)))
	// Offset: 0x274 ( /W)  D-Cache Clean and Invalidate by Set-way
	DCCISW = (*volatile.Register32)(unsafe.Pointer((uintptr(unsafe.Pointer(stm32.SCB)) + 0x274)))
)

func enableICache(enable bool) {
	if enable == stm32.SCB.CCR.HasBits(icacheEnableMsk) {
		return
	}
	if enable {
		arm.AsmFull(`
			dsb 0xF
			isb 0xF
		`, nil)
		ICIALLU.Set(0)
		arm.AsmFull(`
			dsb 0xF
			isb 0xF
		`, nil)
		stm32.SCB.CCR.SetBits(icacheEnableMsk)
		arm.AsmFull(`
			dsb 0xF
			isb 0xF
		`, nil)
	} else {
		arm.AsmFull(`
			dsb 0xF
			isb 0xF
		`, nil)
		stm32.SCB.CCR.ClearBits(icacheEnableMsk)
		ICIALLU.Set(0)
		arm.AsmFull(`
			dsb 0xF
			isb 0xF
		`, nil)
	}
}

func enableDCache(enable bool) {
	if enable == stm32.SCB.CCR.HasBits(dcacheEnableMsk) {
		return
	}
	if enable {
		CSSELR.Set(0)
		arm.AsmFull(`
			dsb 0xF
		`, nil)
		dccache := CCSIDR.Get()
		dcacheSets := (dccache & dccacheNumSetsMsk) >> dccacheNumSetsPos
		for dcacheSets != 0 {
			dcacheWays := (dccache & dccacheAssocMsk) >> dccacheAssocPos
			for dcacheWays != 0 {
				DCISW.Set(
					((dcacheSets << dcacheSetPos) & dcacheSetMsk) |
						((dcacheWays << dcacheWayPos) & dcacheWayMsk))
				dcacheWays--
			}
			dcacheSets--
		}
		arm.AsmFull(`
			dsb 0xF
		`, nil)
		stm32.SCB.CCR.SetBits(dcacheEnableMsk)
		arm.AsmFull(`
			dsb 0xF
			isb 0xF
		`, nil)
	} else {
		var ()
		CSSELR.Set(0)
		arm.AsmFull(`
			dsb 0xF
		`, nil)
		stm32.SCB.CCR.ClearBits(dcacheEnableMsk)
		arm.AsmFull(`
			dsb 0xF
		`, nil)
		dccache.Set(CCSIDR.Get())
		dcacheSets.Set((dccache.Get() & dccacheNumSetsMsk) >> dccacheNumSetsPos)
		for dcacheSets.Get() != 0 {
			dcacheWays.Set((dccache.Get() & dccacheAssocMsk) >> dccacheAssocPos)
			for dcacheWays.Get() != 0 {
				DCCISW.Set(
					((dcacheSets.Get() << dccacheSetPos) & dccacheSetMsk) |
						((dcacheWays.Get() << dccacheWayPos) & dccacheWayMsk))
				dcacheWays.Set(dcacheWays.Get() - 1)
			}
			dcacheSets.Set(dcacheSets.Get() - 1)
		}
		arm.AsmFull(`
			dsb 0xF
			isb 0xF
		`, nil)
	}
}

const (
	IRQ_SysTick     = -1
	sysTickPriority = 16
)

var (
	_          = interrupt.Register(IRQ_SysTick, "SysTick_Handler")
	irqSysTick = interrupt.New(IRQ_SysTick, handleSysTick)
	tickCount  volatile.Register32
)

func initSysTick(freq uint32) {
	// Disable SysTick if already running
	irqSysTick.Disable()
	resetSysTick(freq / 1000)
	irqSysTick.SetPriority(sysTickPriority)
	irqSysTick.Enable()
}

func resetSysTick(ticks uint32) {
	if (ticks - 1) > stm32.STK_RVR_RELOAD_Msk {
		return // invalid tick count
	}
	stm32.STK.RVR.Set(ticks - 1)
	stm32.STK.CVR.Set(0)
	// Enable SysTick IRQ and SysTick Timer
	stm32.STK.CSR.Set(stm32.STK_CSR_CLKSOURCE_Msk |
		stm32.STK_CSR_TICKINT_Msk | stm32.STK_CSR_ENABLE_Msk)
}

func handleSysTick(interrupt.Interrupt) {
	tickCount.Set(tickCount.Get() + 1)
}

func ticksToNanoseconds(ticks timeUnit) int64 {
	return 0
}

func nanosecondsToTicks(ns int64) timeUnit {
	return 0
}

// number of ticks (microseconds) since start.
//go:linkname ticks runtime.ticks
func ticks() timeUnit {
	return timeUnit(tickCount.Get())
}

func sleepTicks(d timeUnit) {}

// pllParam represents PLL frequency parameters.
type pllParam struct {
	m   uint32 // (1-63) division factor for PLL VCO input clock
	n   uint32 // (4-512) Multiplication factor for PLL VCO output clock
	p   uint32 // (2-128, must be even) division factor for system clock
	q   uint32 // (1-128) division factor for peripheral clocks
	r   uint32 // (1-128) division factor for peripheral clocks
	rng uint32 // PLL clock input range
	sel uint32 // PLL clock output range
	fra uint32 // (0-8191) fractional part of multiplication factor for PLL VCO
}

// pllConfig defines an RCC PLL configuration.
type pllConfig struct {
	pll uint32 // PLL state
	src uint32 // clock source
	par pllParam
}

// oscConfig defines an RCC oscillator (HSE/HSI/CSI/LSE/LSI) configuration.
type oscConfig struct {
	osc     uint32    // oscillator to be configured
	hse     uint32    // HSE state
	lse     uint32    // LSE state
	hsi     uint32    // HSI state
	hsiTrim uint32    // (RevY:0-63, RevB+:0-127) calibration trim
	lsi     uint32    // LSI state
	hsi48   uint32    // HSI48 state
	csi     uint32    // CSI state
	csiTrim uint32    // (RevY:0-31, RevB+:0-63) calibration trim
	pll     pllConfig // PLL structure parameters
}

// clkConfig defines an RCC SYS/AHB/APB bus clock configuration.
type clkConfig struct {
	clk     uint32 // clock to be configured
	sysSrc  uint32 // clock source (SYSCLKS) used as system clock
	sysDiv  uint32 // system clock divider
	ahbDiv  uint32 // AHB clock (HCLK) divider (derived from SYSCLK)
	apb3Div uint32 // APB3 clock (D1PCLK1) divider (derived from HCLK)
	apb1Div uint32 // APB1 clock (PCLK1) divider (derived from HCLK)
	apb2Div uint32 // APB2 clock (PCLK2) divider (derived from HCLK)
	apb4Div uint32 // APB4 clock (D3PCLK1) divider (derived from HCLK)
}

func (c clkConfig) apply(latency uint32) bool {
	if latency > stm32.FLASH.ACR.Get()&stm32.FLASH_ACR_LATENCY_Msk {
		const retryMax = 5
		ok := false
		for retry := 0; !ok && retry < retryMax; retry++ {
			stm32.FLASH.ACR.ReplaceBits(latency, stm32.FLASH_ACR_LATENCY_Msk, 0)
			ok = latency == stm32.FLASH.ACR.Get()&stm32.FLASH_ACR_LATENCY_Msk
		}
		if !ok {
			return false // exceeded max attempts, bail out
		}
	}

	if c.clk&rccCLKTypeD1PCLK1 == rccCLKTypeD1PCLK1 {
		if c.apb3Div > stm32.RCC.D1CFGR.Get()&stm32.RCC_D1CFGR_D1PPRE_Msk {
			stm32.RCC.D1CFGR.ReplaceBits(c.apb3Div, stm32.RCC_D1CFGR_D1PPRE_Msk, 0)
		}
	}
	if c.clk&rccCLKTypePCLK1 == rccCLKTypePCLK1 {
		if c.apb1Div > stm32.RCC.D2CFGR.Get()&stm32.RCC_D2CFGR_D2PPRE1_Msk {
			stm32.RCC.D2CFGR.ReplaceBits(c.apb1Div, stm32.RCC_D2CFGR_D2PPRE1_Msk, 0)
		}
	}
	if c.clk&rccCLKTypePCLK2 == rccCLKTypePCLK2 {
		if c.apb2Div > stm32.RCC.D2CFGR.Get()&stm32.RCC_D2CFGR_D2PPRE2_Msk {
			stm32.RCC.D2CFGR.ReplaceBits(c.apb2Div, stm32.RCC_D2CFGR_D2PPRE2_Msk, 0)
		}
	}
	if c.clk&rccCLKTypeD3PCLK1 == rccCLKTypeD3PCLK1 {
		// See notes below regarding TinyGo source and MCU table "RCC_D3CFGR".
		if c.apb4Div > stm32.RCC.D3CFGR.Get()&RCC_D3CFGR_D3PPRE {
			stm32.RCC.D3CFGR.ReplaceBits(c.apb4Div, RCC_D3CFGR_D3PPRE, 0)
		}
	}
	if c.clk&rccCLKTypeHCLK == rccCLKTypeHCLK {
		if c.ahbDiv > stm32.RCC.D1CFGR.Get()&stm32.RCC_D1CFGR_HPRE_Msk {
			stm32.RCC.D1CFGR.ReplaceBits(c.ahbDiv, stm32.RCC_D1CFGR_HPRE_Msk, 0)
		}
	}
	if c.clk&rccCLKTypeSYSCLK == rccCLKTypeSYSCLK {
		stm32.RCC.D1CFGR.ReplaceBits(c.sysDiv, stm32.RCC_D1CFGR_D1CPRE_Msk, 0)
		var flag rccFlag
		switch c.sysSrc {
		case rccSYSCLKSourceHSI:
			flag = rccFlagHSIRDY
		case rccSYSCLKSourceCSI:
			flag = rccFlagCSIRDY
		case rccSYSCLKSourceHSE:
			flag = rccFlagHSERDY
		case rccSYSCLKSourcePLL:
			flag = rccFlagHSERDY
		default:
			return false
		}
		if !flag.get() {
			return false
		}
		stm32.RCC.CFGR.ReplaceBits(c.sysSrc, stm32.RCC_CFGR_SW_Msk, 0)
		// TODO ---
	}
	return true
}

type rccFlag uint8

const (
	rccFlagMASK     rccFlag = 0x1F
	rccFlagHSIRDY   rccFlag = 0x22
	rccFlagHSIDIV   rccFlag = 0x25
	rccFlagCSIRDY   rccFlag = 0x28
	rccFlagHSI48RDY rccFlag = 0x2D
	rccFlagD1CKRDY  rccFlag = 0x2E
	rccFlagCPUCKRDY rccFlag = 0x2E
	rccFlagD2CKRDY  rccFlag = 0x2F
	rccFlagCDCKRDY  rccFlag = 0x2F
	rccFlagHSERDY   rccFlag = 0x31
	rccFlagPLLRDY   rccFlag = 0x39
	rccFlagPLL2RDY  rccFlag = 0x3B
	rccFlagPLL3RDY  rccFlag = 0x3D
	rccFlagLSERDY   rccFlag = 0x41
	rccFlagLSIRDY   rccFlag = 0x61
	rccFlagCPURST   rccFlag = 0x91
	rccFlagD1RST    rccFlag = 0x93
	rccFlagCDRST    rccFlag = 0x93
	rccFlagD2RST    rccFlag = 0x94
	rccFlagBORRST   rccFlag = 0x95
	rccFlagPINRST   rccFlag = 0x96
	rccFlagPORRST   rccFlag = 0x97
	rccFlagSFTRST   rccFlag = 0x98
	rccFlagIWDG1RST rccFlag = 0x9A
	rccFlagWWDG1RST rccFlag = 0x9C
	rccFlagLPWR1RST rccFlag = 0x9E
	rccFlagLPWR2RST rccFlag = 0x9F
	rccFlagC1RST            = rccFlagCPURST
	rccFlagC2RST    rccFlag = 0x92
	rccFlagSFTR1ST          = rccFlagSFTRST
	rccFlagSFTR2ST  rccFlag = 0x99
	rccFlagWWDG2RST rccFlag = 0x9D
	rccFlagIWDG2RST rccFlag = 0x9B
)

func (f rccFlag) get() bool {
	// Derived from the following obnoxious C macro:
	//	 (((((((__FLAG__) >> 5U) == 1U) ?
	//	 	RCC->CR : ((((__FLAG__) >> 5U) == 2U) ?
	//	 		RCC->BDCR : ((((__FLAG__) >> 5U) == 3U) ?
	//	 			RCC->CSR : ((((__FLAG__) >> 5U) == 4U) ?
	//	 				RCC->RSR : RCC->CIFR)))) &
	//	 	(1U << ((__FLAG__) & RCC_FLAG_MASK))) != 0U) ? 1U : 0U)
	var r uint32
	switch f >> 5 {
	case 1:
		r = stm32.RCC.CR.Get()
	case 2:
		r = stm32.RCC.BDCR.Get()
	case 3:
		r = stm32.RCC.CSR.Get()
	case 4:
		r = stm32.RCC.RSR.Get()
	default:
		r = stm32.RCC.CIFR.Get()
	}
	return 0 != r&(1<<(f&rccFlagMASK))
}

const (
	rccCLKTypeSYSCLK  = 0x00000001
	rccCLKTypeHCLK    = 0x00000002
	rccCLKTypeD1PCLK1 = 0x00000004
	rccCLKTypePCLK1   = 0x00000008
	rccCLKTypePCLK2   = 0x00000010
	rccCLKTypeD3PCLK1 = 0x00000020

	rccSYSCLKSourceHSI = 0x00000000
	rccSYSCLKSourceCSI = 0x00000001
	rccSYSCLKSourceHSE = 0x00000002
	rccSYSCLKSourcePLL = 0x00000003

	rccPLLSourceHSI  = 0x00000000
	rccPLLSourceCSI  = 0x00000001
	rccPLLSourceHSE  = 0x00000002
	rccPLLSourceNONE = 0x00000003
)

const (
	// The auto-generated TinyGo source contains bitmasks for the wrong table
	// (RCC_AHB3ENR), and doesn't have those required for RCC_D3CFGR (which has
	// been recreated from STM32 HAL SDK below). Need to determine if this is a
	// bug with the SVD itself or the parser/generator.

	/********************  Bit definition for RCC_D3CFGR register  ******************/
	/*!< D3PPRE configuration */
	RCC_D3CFGR_D3PPRE_Pos       = 4                                  //
	RCC_D3CFGR_D3PPRE_Msk       = 0x7 << RCC_D3CFGR_D3PPRE_Pos       // 0x00000070
	RCC_D3CFGR_D3PPRE           = RCC_D3CFGR_D3PPRE_Msk              // D3PPRE1[2:0] bits (APB4 prescaler)
	RCC_D3CFGR_D3PPRE_0         = 0x1 << RCC_D3CFGR_D3PPRE_Pos       // 0x00000010
	RCC_D3CFGR_D3PPRE_1         = 0x2 << RCC_D3CFGR_D3PPRE_Pos       // 0x00000020
	RCC_D3CFGR_D3PPRE_2         = 0x4 << RCC_D3CFGR_D3PPRE_Pos       // 0x00000040
	RCC_D3CFGR_D3PPRE_DIV1      = 0x00000000                         // APB4 clock not divided
	RCC_D3CFGR_D3PPRE_DIV2_Pos  = 6                                  //
	RCC_D3CFGR_D3PPRE_DIV2_Msk  = 0x1 << RCC_D3CFGR_D3PPRE_DIV2_Pos  // 0x00000040
	RCC_D3CFGR_D3PPRE_DIV2      = RCC_D3CFGR_D3PPRE_DIV2_Msk         // APB4 clock divided by 2
	RCC_D3CFGR_D3PPRE_DIV4_Pos  = 4                                  //
	RCC_D3CFGR_D3PPRE_DIV4_Msk  = 0x5 << RCC_D3CFGR_D3PPRE_DIV4_Pos  // 0x00000050
	RCC_D3CFGR_D3PPRE_DIV4      = RCC_D3CFGR_D3PPRE_DIV4_Msk         // APB4 clock divided by 4
	RCC_D3CFGR_D3PPRE_DIV8_Pos  = 5                                  //
	RCC_D3CFGR_D3PPRE_DIV8_Msk  = 0x3 << RCC_D3CFGR_D3PPRE_DIV8_Pos  // 0x00000060
	RCC_D3CFGR_D3PPRE_DIV8      = RCC_D3CFGR_D3PPRE_DIV8_Msk         // APB4 clock divided by 8
	RCC_D3CFGR_D3PPRE_DIV16_Pos = 4                                  //
	RCC_D3CFGR_D3PPRE_DIV16_Msk = 0x7 << RCC_D3CFGR_D3PPRE_DIV16_Pos // 0x00000070
	RCC_D3CFGR_D3PPRE_DIV16     = RCC_D3CFGR_D3PPRE_DIV16_Msk        // APB4 clock divided by 16
)

const (
	RCC_CFGR_SWS_HSI  = 0x00000000 // HSI used as system clock
	RCC_CFGR_SWS_CSI  = 0x00000008 // CSI used as system clock
	RCC_CFGR_SWS_HSE  = 0x00000010 // HSE used as system clock
	RCC_CFGR_SWS_PLL1 = 0x00000018 // PLL1 used as system clock
)

func sysClockFreq() uint32 {

	switch (stm32.RCC.CFGR.Get() & stm32.RCC_CFGR_SWS_Msk) >> stm32.RCC_CFGR_SWS_Pos {

	case rccSYSCLKSourceHSI:
		hsi := hsiFreqHz
		if rccFlagHSIDIV.get() {
			hsi >>= ((stm32.RCC.CR.Get() & stm32.RCC_CR_HSIDIV_Msk) >>
				stm32.RCC_CR_HSIDIV_Pos)
		}
		return hsi

	case rccSYSCLKSourceCSI:
		return csiFreqHz

	case rccSYSCLKSourceHSE:
		return hseFreqHz

	case rccSYSCLKSourcePLL:
		// We are trying to find H, where:
		//   S := HSE or HSI or CSI (based on PLL source)
		//   H := (S/M * (N + F/8192)) / P
		// Rearranged:
		//   H := (S * (F + N*8192)) / (8192*M*P)
		// It's also given that 0<=M<=63 and 0<=P<=127.
		// Therefore, we pick a scalar X>=8192*64*128 to ensure our ratio is always
		// greater than 1.
		const scale uint64 = 0x04000000

		m := (stm32.RCC.PLLCKSELR.Get() & stm32.RCC_PLLCKSELR_DIVM1_Msk) >>
			stm32.RCC_PLLCKSELR_DIVM1_Pos
		if 0 == m {
			return 0
		}

		var src uint32
		switch (stm32.RCC.PLLCKSELR.Get() & stm32.RCC_PLLCKSELR_PLLSRC_Msk) >>
			stm32.RCC_PLLCKSELR_PLLSRC_Pos {
		case rccPLLSourceHSI:
			src = hsiFreqHz
			if rccFlagHSIDIV.get() {
				src >>= ((stm32.RCC.CR.Get() & stm32.RCC_CR_HSIDIV_Msk) >>
					stm32.RCC_CR_HSIDIV_Pos)
			}
		case rccPLLSourceCSI:
			src = csiFreqHz
		case rccPLLSourceHSE:
			src = hseFreqHz
		default:
			src = csiFreqHz
		}

		var f uint32
		if stm32.RCC.PLLCFGR.HasBits(stm32.RCC_PLLCFGR_PLL1FRACEN) {
			f = (stm32.RCC.PLL1FRACR.Get() & stm32.RCC_PLL1FRACR_FRACN1_Msk) >>
				stm32.RCC_PLL1FRACR_FRACN1_Pos
		}
		n := ((stm32.RCC.PLL1DIVR.Get() & stm32.RCC_PLL1DIVR_DIVN1_Msk) >>
			stm32.RCC_PLL1DIVR_DIVN1_Pos) + 1
		p := ((stm32.RCC.PLL1DIVR.Get() & stm32.RCC_PLL1DIVR_DIVP1_Msk) >>
			stm32.RCC_PLL1DIVR_DIVP1_Pos) + 1

		//return uint32(((scale * uint64(src*f+0x2000*n)) /
		//	uint64(0x2000*m*p)) / scale)

		return uint32(((float32(src) / float32(m) *
			(float32(n) + (float32(f) / 0x2000))) / float32(p)))

	default:
		// CSI is default source for SYSCLK
		return csiFreqHz
	}
}

func initSysClockPLL(bypassHSE, lowSpeed bool) bool {

	if rccPLLSourceHSE == (stm32.RCC.PLLCKSELR.Get()&
		stm32.RCC_PLLCKSELR_PLLSRC_Msk)>>stm32.RCC_PLLCKSELR_PLLSRC_Pos {
		if !(clkConfig{
			clk:    rccCLKTypeSYSCLK,
			sysSrc: rccSYSCLKSourceCSI,
		}).apply(stm32.FLASH_ACR_LATENCY_1WS) {
			return false
		}
	}
	// TODO --
	return true
}
