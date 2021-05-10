// +build stm32h7x7_cm7

package runtime

import (
	"device/stm32"
	"runtime/volatile"
	"unsafe"
)

func preinitCore() {

	// Increasing CPU frequency
	if stm32.FLASH_LATENCY_DEFAULT > stm32.FLASH.ACR.Get()&stm32.FLASH_ACR_LATENCY_Msk {
		stm32.FLASH.ACR.ReplaceBits(stm32.FLASH_LATENCY_DEFAULT, stm32.FLASH_ACR_LATENCY_Msk, 0)
	}

	stm32.RCC.CR.SetBits(stm32.RCC_CR_HSION)
	stm32.RCC.CFGR.Set(0)
	stm32.RCC.CR.ClearBits(stm32.RCC_CR_HSEON | stm32.RCC_CR_HSECSSON |
		stm32.RCC_CR_CSION | stm32.RCC_CR_RC48ON | stm32.RCC_CR_CSIKERON |
		stm32.RCC_CR_PLL1ON | stm32.RCC_CR_PLL2ON | stm32.RCC_CR_PLL3ON)

	// Decreasing the number of wait states because of lower CPU frequency
	if stm32.FLASH_LATENCY_DEFAULT < stm32.FLASH.ACR.Get()&stm32.FLASH_ACR_LATENCY_Msk {
		stm32.FLASH.ACR.ReplaceBits(stm32.FLASH_LATENCY_DEFAULT, stm32.FLASH_ACR_LATENCY_Msk, 0)
	}

	stm32.RCC.D1CFGR.Set(0)
	stm32.RCC.D2CFGR.Set(0)
	stm32.RCC.D3CFGR.Set(0)
	stm32.RCC.PLLCKSELR.Set(0x02020200)
	stm32.RCC.PLLCFGR.Set(0x01FF0000)
	stm32.RCC.PLL1DIVR.Set(0x01010280)
	stm32.RCC.PLL1FRACR.Set(0)
	stm32.RCC.PLL2DIVR.Set(0x01010280)
	stm32.RCC.PLL2FRACR.Set(0)
	stm32.RCC.PLL3DIVR.Set(0x01010280)
	stm32.RCC.PLL3FRACR.Set(0)
	stm32.RCC.CR.ClearBits(stm32.RCC_CR_HSEBYP)

	stm32.RCC.CIER.Set(0) // Disable all interrupts

	// Enable Cortex-M7 HSEM EXTI line (line 78)
	stm32.EXTI_CORE2.EMR3.SetBits(0x4000)

	// Check if STM32H7 RevY
	if (stm32.DBGMCU.IDCODE.Get() & stm32.DBGMCU_IDCODE_REV_ID_Msk) < 0x20000000 {
		// Change the switch matrix read issuing capability to 1 for the AXI SRAM
		// target (Target 7)
		((*volatile.Register32)(unsafe.Pointer(uintptr(0x51008108)))).Set(1)
	}

	// Disable FMC bank 1 (enabled after reset).
	// This prevents CPU speculation access on this bank, which blocks the use of
	// FMC during 24us. During this time, the other FMC masters (such as LTDC)
	// cannot use it!
	stm32.FMC.FMC_BCR1.Set(0x000030D2)

	if true {
		// Initialize VTOR with vectors in flash
		src := unsafe.Pointer(&_svectors)
		stm32.SCB.VTOR.Set(uint32(uintptr(src)))
	} else {
		// Copy vectors from flash to ITCM
		src := unsafe.Pointer(&_svectors)
		dst := unsafe.Pointer(&_svtor)
		for src != unsafe.Pointer(&_evectors) {
			*(*uint32)(dst) = *(*uint32)(src)
			src = unsafe.Pointer(uintptr(src) + 4)
			dst = unsafe.Pointer(uintptr(dst) + 4)
		}
		// Initialize VTOR with vectors in ITCM
		stm32.SCB.VTOR.Set(uint32(uintptr(unsafe.Pointer(&_svtor))))
	}
}

func initCore() {

	initCache()
	initSemaphore()

	for rccFlagD2CKRDY.get() {
	} // Wait until Cortex-M4 enters stop mode
}

func initCoreClock() {

}

func setCoreFreq(d1, d2 uint32) { coreD1FreqHz, coreD2FreqHz = d1, d2 }
