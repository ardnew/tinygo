// +build stm32h7x5

package machine

import (
	"device/stm32"
	"runtime/volatile"
	"unsafe"
)

// EnableClock enables or disables the clock gate(s) for the given peripheral.
// If the peripheral depends on multiple clock gates for normal operation, then
// each of those gates are enabled/disabled.
func EnableClock(bus unsafe.Pointer, enable bool) bool {

	regMask := map[*volatile.Register32]uint32{}

	switch bus {

	// AHB1/AHB1LP
	case unsafe.Pointer(stm32.ART):
		regMask[&stm32.RCC.AHB1ENR] = stm32.RCC_AHB1ENR_ARTEN
	case unsafe.Pointer(stm32.OTG1_HS_DEVICE):
		// Configure both OTG PHY and ULPI clocks for USB 1
		regMask[&stm32.RCC.AHB1ENR] =
			stm32.RCC_AHB1ENR_USB1OTGEN | stm32.RCC_AHB1ENR_USB1ULPIEN
		// Also configure the corresponding low-power clocks for USB 1
		regMask[&stm32.RCC.AHB1LPENR] =
			stm32.RCC_AHB1LPENR_USB1OTGLPEN | stm32.RCC_AHB1LPENR_USB1ULPILPEN
	case unsafe.Pointer(stm32.OTG2_HS_DEVICE):
		// Configure both OTG PHY and ULPI clocks for USB 2
		regMask[&stm32.RCC.AHB1ENR] =
			stm32.RCC_AHB1ENR_USB2OTGEN | stm32.RCC_AHB1ENR_USB2ULPIEN
		// Also configure the corresponding low-power clocks for USB 2
		regMask[&stm32.RCC.AHB1LPENR] =
			stm32.RCC_AHB1LPENR_USB2OTGLPEN | stm32.RCC_AHB1LPENR_USB2ULPILPEN

	// APB1/APB1L
	case unsafe.Pointer(stm32.LPTIM1):
		regMask[&stm32.RCC.APB1LENR] = stm32.RCC_APB1LENR_LPTIM1EN

	// APB4
	case unsafe.Pointer(stm32.SYSCFG):
		regMask[&stm32.RCC.APB4ENR] = stm32.RCC_APB4ENR_SYSCFGEN

	// AHB4
	case unsafe.Pointer(stm32.HSEM):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_HSEMEN
	case unsafe.Pointer(stm32.GPIOA):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOAEN
	case unsafe.Pointer(stm32.GPIOB):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOBEN
	case unsafe.Pointer(stm32.GPIOC):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOCEN
	case unsafe.Pointer(stm32.GPIOD):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIODEN
	case unsafe.Pointer(stm32.GPIOE):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOEEN
	case unsafe.Pointer(stm32.GPIOF):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOFEN
	case unsafe.Pointer(stm32.GPIOG):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOGEN
	case unsafe.Pointer(stm32.GPIOH):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOHEN
	case unsafe.Pointer(stm32.GPIOI):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOIEN
	case unsafe.Pointer(stm32.GPIOJ):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOJEN
	case unsafe.Pointer(stm32.GPIOK):
		regMask[&stm32.RCC.AHB4ENR] = stm32.RCC_AHB4ENR_GPIOKEN
	}

	// if regMask is empty, we received an unhandled peripheral
	ok := len(regMask) > 0

	for reg, mask := range regMask {
		for !SemRCC.Lock(CoreID) {
		} // wait until we have exclusive access to RCC
		if enable {
			reg.SetBits(mask)
		} else {
			reg.ClearBits(mask)
		}
		// Reading the register back out gives RCC a few cycles to enable the clock,
		// and lets us verify the change was accepted.
		ok = ok && (enable == reg.HasBits(mask))
		// Release the semaphore once we've performed all accesses
		SemRCC.Unlock(CoreID)
	}

	return ok
}
