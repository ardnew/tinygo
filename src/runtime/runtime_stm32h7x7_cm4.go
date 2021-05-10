// +build stm32h7x7_cm4

package runtime

import (
	"device/stm32"
	"runtime/volatile"
)

func preinitCore() {

	// Initialize VTOR with vectors in flash
	vtor := uintptr(unsafe.Pointer(&_svectors))
	stm32.SCB.VTOR.Set(uint32(vtor))

	// TODO: copy vectors to SRAM? I don't believe the M4 core can access any of
	//       the TCM regions, so VTOR cannot point to ITCM/DTCM.
}

func initCore() {

	initCache()
	initSemaphore()

	allocFlash()

	if !stm32.IsBootM4() {
		stm32.HSEM_CORE2.IER.SetBits(1 << semStopMode)
		// MODIFY_REG(PWR->CR1, PWR_CR1_LPDS, LL_PWR_REGU_DSMODE_MAIN);
		stm32.PWR.CR1.ReplaceBits(LL_PWR_REGU_DSMODE_MAIN, PWR_CR1_LPDS, 0)
		// MODIFY_REG(PWR->CPUCR, PWR_CPUCR_PDDS_D2, LL_PWR_CPU_MODE_D2STOP);
		stm32.PWR.CPUCR.ReplaceBits(LL_PWR_CPU_MODE_D2STOP, PWR_CPUCR_PDDS_D2, 0)
		// MODIFY_REG(PWR->CPU2CR, PWR_CPU2CR_PDDS_D2, LL_PWR_CPU2_MODE_D2STOP);
		stm32.PWR.CPU2CR.ReplaceBits(LL_PWR_CPU2_MODE_D2STOP, PWR_CPU2CR_PDDS_D2, 0)
		// SET_BIT(SCB->SCR, SCB_SCR_SLEEPDEEP_Msk);
		stm32.SCB.SCR.SetBits(stm32.SCB_SCR_SLEEPDEEP_Msk)
		// __DSB(); __ISB(); __WFE();
		arm.AsmFull(`
	dsb 0xF
	isb 0xF
	wfe
`, nil)
		// CLEAR_BIT(SCB->SCR, SCB_SCR_SLEEPDEEP_Msk);
		stm32.SCB.SCR.ClearBits(stm32.SCB_SCR_SLEEPDEEP_Msk)
		// CLEAR_BIT(HSEMx->C2IER, 1 << semStopMode);
		stm32.HSEM_CORE2.IER.ClearBits(1 << semStopMode)
		// WRITE_REG(HSEMx->C2ICR, 1 << semStopMode);
		stm32.HSEM_CORE2.ICR.Set(1 << semStopMode)
	}
}

func initCoreClock() {

}

func setCoreFreq(d1, d2 uint32) { coreD1FreqHz, coreD2FreqHz = d2, d2 }

var flashAllocated volatile.Register32

func allocFlash() {
	stm32.RCC_CORE2.AHB3ENR.SetBits(stm32.RCC_AHB3ENR_FLASHEN)
	if stm32.RCC_CORE2.AHB3ENR.HasBits(stm32.RCC_AHB3ENR_FLASHEN) {
		flashAllocated.Set(1)
	} else {
		flashAllocated.Set(0)
	}
}
