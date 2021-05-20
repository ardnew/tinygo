// +build stm32h7x7_cm7

package runtime

import (
	"device/stm32"
	"runtime/volatile"
	"unsafe"
)

func resetCore() {

	if stm32.FLASH_LATENCY_DEFAULT >
		stm32.FLASH.ACR.Get()&stm32.FLASH_ACR_LATENCY_Msk {
		stm32.FLASH.ACR.ReplaceBits(
			stm32.FLASH_LATENCY_DEFAULT, stm32.FLASH_ACR_LATENCY_Msk, 0)
	}

	stm32.RCC.CR.SetBits(stm32.RCC_CR_HSION)
	stm32.RCC.CFGR.Set(0)
	stm32.RCC.CR.ClearBits(stm32.RCC_CR_HSEON | stm32.RCC_CR_HSECSSON |
		stm32.RCC_CR_CSION | stm32.RCC_CR_RC48ON | stm32.RCC_CR_CSIKERON |
		stm32.RCC_CR_PLL1ON | stm32.RCC_CR_PLL2ON | stm32.RCC_CR_PLL3ON)

	if stm32.FLASH_LATENCY_DEFAULT <
		stm32.FLASH.ACR.Get()&stm32.FLASH_ACR_LATENCY_Msk {
		stm32.FLASH.ACR.ReplaceBits(
			stm32.FLASH_LATENCY_DEFAULT, stm32.FLASH_ACR_LATENCY_Msk, 0)
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
}

func initCore() {

	// Enable Cortex-M7 HSEM EXTI line (line 78)
	stm32.EXTI_CORE2.EMR3.SetBits(0x4000)

	// Check if STM32H7 revision prior to revision B
	if stm32.DBG.MCURevision() < stm32.DBG_MCU_REVISION_B {
		// Change the switch matrix read issuing capability to 1 for the AXI SRAM
		// target (Target 7)
		((*volatile.Register32)(unsafe.Pointer(uintptr(0x51008108)))).Set(1)
	}

	// Disable FMC bank 1 (enabled after reset).
	// This prevents CPU speculation access on this bank, which blocks the use of
	// FMC during 24us. During this time, the other FMC masters (such as LTDC)
	// cannot use it!
	stm32.FMC.FMC_BCR1.Set(0x000030D2)

	if false {
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

	// Disable MPU for now
	stm32.MPU.Enable(false)

	// Enable data/instruction cache
	stm32.SCB.EnableICache(true)
	stm32.SCB.EnableDCache(true)

	// Enable hardware semaphore
	stm32.HSEM.Enable(true)

	//for rccFlagD2CKRDY.get() {
	//} // Wait until Cortex-M4 enters stop mode
}

func initCoreFreq() uint32 {

	stm32.RCC.D1CFGR.ClearBits(stm32.RCC_D1CFGR_D1CPRE_Msk)
	stm32.RCC.CFGR.ReplaceBits(stm32.RCC_CFGR_SW_CSI, stm32.RCC_CFGR_SW_Msk, 0)
	for stm32.RCC.CFGR.Get()&stm32.RCC_CFGR_SWS_Msk != stm32.RCC_CFGR_SWS_CSI {
	}

	stm32.FLASH.SetLatency(stm32.FLASH_ACR_LATENCY_1WS) // 1 wait states

	stm32.PWR.Configure(
		stm32.PWR_SMPS_1V8_SUPPLIES_LDO, stm32.PWR_REGULATOR_VOLTAGE_SCALE0)

	for stm32.PWR.PWR_D3CR.Get()&stm32.PWR_PWR_D3CR_VOSRDY != stm32.PWR_PWR_D3CR_VOSRDY {
	}

	stm32.RCC.CR.SetBits(stm32.RCC_CR_HSEBYP)
	stm32.RCC.CR.SetBits(stm32.RCC_CR_HSEON)

	for stm32.RCC.CR.Get()&stm32.RCC_CR_HSERDY != stm32.RCC_CR_HSERDY {
	}

	stm32.RCC.PLLCKSELR.ReplaceBits(
		stm32.RCC_PLLCKSELR_PLLSRC_HSE, stm32.RCC_PLLCKSELR_PLLSRC_Msk, 0)
	stm32.RCC.PLLCFGR.SetBits(stm32.RCC_PLLCFGR_DIVP1EN)

	stm32.RCC.PLLCFGR.ReplaceBits(
		stm32.RCC_PLLCFGR_PLL1RGE_2, stm32.RCC_PLLCFGR_PLL1RGE_Msk, 0)
	stm32.RCC.PLLCFGR.ReplaceBits(
		stm32.RCC_PLLCFGR_PLL1VCOSEL_WIDE, stm32.RCC_PLLCFGR_PLL1VCOSEL_Msk, 0)

	// M=1, N=192, P=2, Q=2, R=2
	stm32.RCC.PLLCKSELR.ReplaceBits(5, stm32.RCC_PLLCKSELR_DIVM1_Msk, 0)
	stm32.RCC.PLL1DIVR.ReplaceBits(192-1, stm32.RCC_PLL1DIVR_N1_Msk, 0)
	stm32.RCC.PLL1DIVR.ReplaceBits(2-1, stm32.RCC_PLL1DIVR_P1_Msk, 0)
	stm32.RCC.PLL1DIVR.ReplaceBits(2-1, stm32.RCC_PLL1DIVR_Q1_Msk, 0)
	stm32.RCC.PLL1DIVR.ReplaceBits(2-1, stm32.RCC_PLL1DIVR_R1_Msk, 0)
	stm32.RCC.CR.SetBits(stm32.RCC_CR_PLL1ON)

	for stm32.RCC.CR.Get()&stm32.RCC_CR_PLL1RDY != stm32.RCC_CR_PLL1RDY {
	}

	stm32.FLASH.SetLatency(stm32.FLASH_ACR_LATENCY_4WS) // 4 wait states

	// (DIV) SYS=1, AHB=2, APB1=2, APB2=2, APB3=2, APB4=2
	stm32.RCC.D1CFGR.ReplaceBits(
		stm32.RCC_D1CFGR_HPRE_DIV2, stm32.RCC_D1CFGR_HPRE_Msk, 0)
	stm32.RCC.CFGR.ReplaceBits(
		stm32.RCC_CFGR_SW_PLL, stm32.RCC_CFGR_SW_Msk, 0)
	stm32.RCC.D1CFGR.ReplaceBits(
		stm32.RCC_D1CFGR_D1CPRE_DIV1, stm32.RCC_D1CFGR_D1CPRE_Msk, 0)
	stm32.RCC.D1CFGR.ReplaceBits(
		stm32.RCC_D1CFGR_HPRE_DIV2, stm32.RCC_D1CFGR_HPRE_Msk, 0)
	stm32.RCC.D2CFGR.ReplaceBits(
		stm32.RCC_D2CFGR_D2PPRE1_DIV2, stm32.RCC_D2CFGR_D2PPRE1_Msk, 0)
	stm32.RCC.D2CFGR.ReplaceBits(
		stm32.RCC_D2CFGR_D2PPRE2_DIV2, stm32.RCC_D2CFGR_D2PPRE2_Msk, 0)
	stm32.RCC.D1CFGR.ReplaceBits(
		stm32.RCC_D1CFGR_D1PPRE_DIV2, stm32.RCC_D1CFGR_D1PPRE_Msk, 0)
	stm32.RCC.D3CFGR.ReplaceBits(
		stm32.RCC_D3CFGR_D3PPRE_DIV2, stm32.RCC_D3CFGR_D3PPRE_Msk, 0)

	// M7 core runs at the faster SYSCLK frequency (480 MHz)
	return stm32.RCC.ClockFreq().SYSCLK
}
