// Hand created file. DO NOT DELETE.
// Methods for controlling the clock gate of each core of the STM32H7x7 family
// of dual-core MCUs.
// These methods are available for use from the context of both Cortex-M7 and
// Cortex-M4 cores.

// +build stm32
// +build stm32h7x7_cm4 stm32h7x7_cm7

package stm32

import (
	"runtime/volatile"
	"unsafe"
)

// Boot Cortex-M4 core (if held by option byte BCM4 = 0)
func BootM4() {
	RCC.GCR.SetBits(RCC_GCR_BOOT_C2)
}

// Check if Cortex-M4 core has booted
func IsBootM4() bool {
	return RCC.GCR.HasBits(RCC_GCR_BOOT_C2)
}

// Boot Cortex-M7 core (if held by option byte BCM7 = 0)
func BootM7() {
	RCC.GCR.SetBits(RCC_GCR_BOOT_C1)
}

// Check if Cortex-M7 core has booted
func IsBootM7() bool {
	return RCC.GCR.HasBits(RCC_GCR_BOOT_C1)
}

const (
	RCC_AHB3ENR_FLASHEN_Pos = 8
	RCC_AHB3ENR_FLASHEN_Msk = 0x1 << RCC_AHB3ENR_FLASHEN_Pos // 0x00000100
	RCC_AHB3ENR_FLASHEN     = RCC_AHB3ENR_FLASHEN_Msk
)

var (
	RCC_CORE1 = (*RCC_CORE_Type)(unsafe.Pointer(uintptr(0x58024530)))
	RCC_CORE2 = (*RCC_CORE_Type)(unsafe.Pointer(uintptr(0x58024590)))
)

// RCC_CORE
type RCC_CORE_Type struct {
	RSR        volatile.Register32 // RCC Reset status register                            Address offset: 0x00
	AHB3ENR    volatile.Register32 // RCC AHB3 peripheral clock  register                  Address offset: 0x04
	AHB1ENR    volatile.Register32 // RCC AHB1 peripheral clock  register                  Address offset: 0x08
	AHB2ENR    volatile.Register32 // RCC AHB2 peripheral clock  register                  Address offset: 0x0C
	AHB4ENR    volatile.Register32 // RCC AHB4 peripheral clock  register                  Address offset: 0x10
	APB3ENR    volatile.Register32 // RCC APB3 peripheral clock  register                  Address offset: 0x14
	APB1LENR   volatile.Register32 // RCC APB1 peripheral clock  Low Word register         Address offset: 0x18
	APB1HENR   volatile.Register32 // RCC APB1 peripheral clock  High Word register        Address offset: 0x1C
	APB2ENR    volatile.Register32 // RCC APB2 peripheral clock  register                  Address offset: 0x20
	APB4ENR    volatile.Register32 // RCC APB4 peripheral clock  register                  Address offset: 0x24
	_          [4]byte             // Reserved                                             Address offset: 0x28
	AHB3LPENR  volatile.Register32 // RCC AHB3 peripheral sleep clock  register            Address offset: 0x3C
	AHB1LPENR  volatile.Register32 // RCC AHB1 peripheral sleep clock  register            Address offset: 0x40
	AHB2LPENR  volatile.Register32 // RCC AHB2 peripheral sleep clock  register            Address offset: 0x44
	AHB4LPENR  volatile.Register32 // RCC AHB4 peripheral sleep clock  register            Address offset: 0x48
	APB3LPENR  volatile.Register32 // RCC APB3 peripheral sleep clock  register            Address offset: 0x4C
	APB1LLPENR volatile.Register32 // RCC APB1 peripheral sleep clock  Low Word register   Address offset: 0x50
	APB1HLPENR volatile.Register32 // RCC APB1 peripheral sleep clock  High Word register  Address offset: 0x54
	APB2LPENR  volatile.Register32 // RCC APB2 peripheral sleep clock  register            Address offset: 0x58
	APB4LPENR  volatile.Register32 // RCC APB4 peripheral sleep clock  register            Address offset: 0x5C
	_          [16]byte            // Reserved, 0x60-0x6C                                  Address offset: 0x60
}
