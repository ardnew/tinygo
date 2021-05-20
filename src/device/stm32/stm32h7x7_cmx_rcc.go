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

//go:linkname ticks runtime.ticks
func ticks() int64

// Force Cortex-M4 core to boot (if held by option byte BCM4 = 0)
func BootM4() {
	RCC.GCR.SetBits(RCC_GCR_BOOT_C2)
}

// Check if Cortex-M4 core was booted by force
func IsBootM4() bool {
	return RCC.GCR.HasBits(RCC_GCR_BOOT_C2)
}

// Force Cortex-M7 core to boot (if held by option byte BCM7 = 0)
func BootM7() {
	RCC.GCR.SetBits(RCC_GCR_BOOT_C1)
}

// Check if Cortex-M7 core was booted by force
func IsBootM7() bool {
	return RCC.GCR.HasBits(RCC_GCR_BOOT_C1)
}

const (
	RCC_D1CFGR_HPRE   = RCC_D1CFGR_HPRE_Msk        // HPRE[3:0] bits (AHB3 prescaler)
	RCC_D1CFGR_HPRE_0 = 0x1 << RCC_D1CFGR_HPRE_Pos // 0x00000001
	RCC_D1CFGR_HPRE_1 = 0x2 << RCC_D1CFGR_HPRE_Pos // 0x00000002
	RCC_D1CFGR_HPRE_2 = 0x4 << RCC_D1CFGR_HPRE_Pos // 0x00000004
	RCC_D1CFGR_HPRE_3 = 0x8 << RCC_D1CFGR_HPRE_Pos // 0x00000008

	RCC_D1CFGR_HPRE_DIV1       = 0                                 // AHB3 Clock not divided
	RCC_D1CFGR_HPRE_DIV2_Pos   = 3                                 // //
	RCC_D1CFGR_HPRE_DIV2_Msk   = 0x1 << RCC_D1CFGR_HPRE_DIV2_Pos   // 0x00000008
	RCC_D1CFGR_HPRE_DIV2       = RCC_D1CFGR_HPRE_DIV2_Msk          // AHB3 Clock divided by 2
	RCC_D1CFGR_HPRE_DIV4_Pos   = 0                                 //
	RCC_D1CFGR_HPRE_DIV4_Msk   = 0x9 << RCC_D1CFGR_HPRE_DIV4_Pos   // 0x00000009
	RCC_D1CFGR_HPRE_DIV4       = RCC_D1CFGR_HPRE_DIV4_Msk          // AHB3 Clock divided by 4
	RCC_D1CFGR_HPRE_DIV8_Pos   = 1                                 //
	RCC_D1CFGR_HPRE_DIV8_Msk   = 0x5 << RCC_D1CFGR_HPRE_DIV8_Pos   // 0x0000000A
	RCC_D1CFGR_HPRE_DIV8       = RCC_D1CFGR_HPRE_DIV8_Msk          // AHB3 Clock divided by 8
	RCC_D1CFGR_HPRE_DIV16_Pos  = 0                                 //
	RCC_D1CFGR_HPRE_DIV16_Msk  = 0xB << RCC_D1CFGR_HPRE_DIV16_Pos  // 0x0000000B
	RCC_D1CFGR_HPRE_DIV16      = RCC_D1CFGR_HPRE_DIV16_Msk         // AHB3 Clock divided by 16
	RCC_D1CFGR_HPRE_DIV64_Pos  = 2                                 //
	RCC_D1CFGR_HPRE_DIV64_Msk  = 0x3 << RCC_D1CFGR_HPRE_DIV64_Pos  // 0x0000000C
	RCC_D1CFGR_HPRE_DIV64      = RCC_D1CFGR_HPRE_DIV64_Msk         // AHB3 Clock divided by 64
	RCC_D1CFGR_HPRE_DIV128_Pos = 0                                 //
	RCC_D1CFGR_HPRE_DIV128_Msk = 0xD << RCC_D1CFGR_HPRE_DIV128_Pos // 0x0000000D
	RCC_D1CFGR_HPRE_DIV128     = RCC_D1CFGR_HPRE_DIV128_Msk        // AHB3 Clock divided by 128
	RCC_D1CFGR_HPRE_DIV256_Pos = 1                                 //
	RCC_D1CFGR_HPRE_DIV256_Msk = 0x7 << RCC_D1CFGR_HPRE_DIV256_Pos // 0x0000000E
	RCC_D1CFGR_HPRE_DIV256     = RCC_D1CFGR_HPRE_DIV256_Msk        // AHB3 Clock divided by 256
	RCC_D1CFGR_HPRE_DIV512_Pos = 0                                 //
	RCC_D1CFGR_HPRE_DIV512_Msk = 0xF << RCC_D1CFGR_HPRE_DIV512_Pos // 0x0000000F
	RCC_D1CFGR_HPRE_DIV512     = RCC_D1CFGR_HPRE_DIV512_Msk        // AHB3 Clock divided by 512

	RCC_D1CFGR_D1PPRE   = RCC_D1CFGR_D1PPRE_Msk        // D1PRE[2:0] bits (APB3 prescaler)
	RCC_D1CFGR_D1PPRE_0 = 0x1 << RCC_D1CFGR_D1PPRE_Pos // 0x00000010
	RCC_D1CFGR_D1PPRE_1 = 0x2 << RCC_D1CFGR_D1PPRE_Pos // 0x00000020
	RCC_D1CFGR_D1PPRE_2 = 0x4 << RCC_D1CFGR_D1PPRE_Pos // 0x00000040

	RCC_D1CFGR_D1PPRE_DIV1      = 0                                  // APB3 clock not divided
	RCC_D1CFGR_D1PPRE_DIV2_Pos  = 6                                  //
	RCC_D1CFGR_D1PPRE_DIV2_Msk  = 0x1 << RCC_D1CFGR_D1PPRE_DIV2_Pos  // 0x00000040
	RCC_D1CFGR_D1PPRE_DIV2      = RCC_D1CFGR_D1PPRE_DIV2_Msk         // APB3 clock divided by 2
	RCC_D1CFGR_D1PPRE_DIV4_Pos  = 4                                  //
	RCC_D1CFGR_D1PPRE_DIV4_Msk  = 0x5 << RCC_D1CFGR_D1PPRE_DIV4_Pos  // 0x00000050
	RCC_D1CFGR_D1PPRE_DIV4      = RCC_D1CFGR_D1PPRE_DIV4_Msk         // APB3 clock divided by 4
	RCC_D1CFGR_D1PPRE_DIV8_Pos  = 5                                  //
	RCC_D1CFGR_D1PPRE_DIV8_Msk  = 0x3 << RCC_D1CFGR_D1PPRE_DIV8_Pos  // 0x00000060
	RCC_D1CFGR_D1PPRE_DIV8      = RCC_D1CFGR_D1PPRE_DIV8_Msk         // APB3 clock divided by 8
	RCC_D1CFGR_D1PPRE_DIV16_Pos = 4                                  //
	RCC_D1CFGR_D1PPRE_DIV16_Msk = 0x7 << RCC_D1CFGR_D1PPRE_DIV16_Pos // 0x00000070
	RCC_D1CFGR_D1PPRE_DIV16     = RCC_D1CFGR_D1PPRE_DIV16_Msk        // APB3 clock divided by 16

	RCC_D1CFGR_D1CPRE   = RCC_D1CFGR_D1CPRE_Msk        // D1CPRE[2:0] bits (Domain 1 Core prescaler)
	RCC_D1CFGR_D1CPRE_0 = 0x1 << RCC_D1CFGR_D1CPRE_Pos // 0x00000100
	RCC_D1CFGR_D1CPRE_1 = 0x2 << RCC_D1CFGR_D1CPRE_Pos // 0x00000200
	RCC_D1CFGR_D1CPRE_2 = 0x4 << RCC_D1CFGR_D1CPRE_Pos // 0x00000400
	RCC_D1CFGR_D1CPRE_3 = 0x8 << RCC_D1CFGR_D1CPRE_Pos // 0x00000800

	RCC_D1CFGR_D1CPRE_DIV1       = 0                                   // Domain 1 Core clock not divided
	RCC_D1CFGR_D1CPRE_DIV2_Pos   = 11                                  //
	RCC_D1CFGR_D1CPRE_DIV2_Msk   = 0x1 << RCC_D1CFGR_D1CPRE_DIV2_Pos   // 0x00000800
	RCC_D1CFGR_D1CPRE_DIV2       = RCC_D1CFGR_D1CPRE_DIV2_Msk          // Domain 1 Core clock divided by 2
	RCC_D1CFGR_D1CPRE_DIV4_Pos   = 8                                   //
	RCC_D1CFGR_D1CPRE_DIV4_Msk   = 0x9 << RCC_D1CFGR_D1CPRE_DIV4_Pos   // 0x00000900
	RCC_D1CFGR_D1CPRE_DIV4       = RCC_D1CFGR_D1CPRE_DIV4_Msk          // Domain 1 Core clock divided by 4
	RCC_D1CFGR_D1CPRE_DIV8_Pos   = 9                                   //
	RCC_D1CFGR_D1CPRE_DIV8_Msk   = 0x5 << RCC_D1CFGR_D1CPRE_DIV8_Pos   // 0x00000A00
	RCC_D1CFGR_D1CPRE_DIV8       = RCC_D1CFGR_D1CPRE_DIV8_Msk          // Domain 1 Core clock divided by 8
	RCC_D1CFGR_D1CPRE_DIV16_Pos  = 8                                   //
	RCC_D1CFGR_D1CPRE_DIV16_Msk  = 0xB << RCC_D1CFGR_D1CPRE_DIV16_Pos  // 0x00000B00
	RCC_D1CFGR_D1CPRE_DIV16      = RCC_D1CFGR_D1CPRE_DIV16_Msk         // Domain 1 Core clock divided by 16
	RCC_D1CFGR_D1CPRE_DIV64_Pos  = 10                                  //
	RCC_D1CFGR_D1CPRE_DIV64_Msk  = 0x3 << RCC_D1CFGR_D1CPRE_DIV64_Pos  // 0x00000C00
	RCC_D1CFGR_D1CPRE_DIV64      = RCC_D1CFGR_D1CPRE_DIV64_Msk         // Domain 1 Core clock divided by 64
	RCC_D1CFGR_D1CPRE_DIV128_Pos = 8                                   //
	RCC_D1CFGR_D1CPRE_DIV128_Msk = 0xD << RCC_D1CFGR_D1CPRE_DIV128_Pos // 0x00000D00
	RCC_D1CFGR_D1CPRE_DIV128     = RCC_D1CFGR_D1CPRE_DIV128_Msk        // Domain 1 Core clock divided by 128
	RCC_D1CFGR_D1CPRE_DIV256_Pos = 9                                   //
	RCC_D1CFGR_D1CPRE_DIV256_Msk = 0x7 << RCC_D1CFGR_D1CPRE_DIV256_Pos // 0x00000E00
	RCC_D1CFGR_D1CPRE_DIV256     = RCC_D1CFGR_D1CPRE_DIV256_Msk        // Domain 1 Core clock divided by 256
	RCC_D1CFGR_D1CPRE_DIV512_Pos = 8                                   //
	RCC_D1CFGR_D1CPRE_DIV512_Msk = 0xF << RCC_D1CFGR_D1CPRE_DIV512_Pos // 0x00000F00
	RCC_D1CFGR_D1CPRE_DIV512     = RCC_D1CFGR_D1CPRE_DIV512_Msk        // Domain 1 Core clock divided by 512

	RCC_D2CFGR_D2PPRE1   = RCC_D2CFGR_D2PPRE1_Msk        // D1PPRE1[2:0] bits (APB1 prescaler)
	RCC_D2CFGR_D2PPRE1_0 = 0x1 << RCC_D2CFGR_D2PPRE1_Pos // 0x00000010
	RCC_D2CFGR_D2PPRE1_1 = 0x2 << RCC_D2CFGR_D2PPRE1_Pos // 0x00000020
	RCC_D2CFGR_D2PPRE1_2 = 0x4 << RCC_D2CFGR_D2PPRE1_Pos // 0x00000040

	RCC_D2CFGR_D2PPRE1_DIV1      = 0                                   // APB1 clock not divided
	RCC_D2CFGR_D2PPRE1_DIV2_Pos  = 6                                   //
	RCC_D2CFGR_D2PPRE1_DIV2_Msk  = 0x1 << RCC_D2CFGR_D2PPRE1_DIV2_Pos  // 0x00000040
	RCC_D2CFGR_D2PPRE1_DIV2      = RCC_D2CFGR_D2PPRE1_DIV2_Msk         // APB1 clock divided by 2
	RCC_D2CFGR_D2PPRE1_DIV4_Pos  = 4                                   //
	RCC_D2CFGR_D2PPRE1_DIV4_Msk  = 0x5 << RCC_D2CFGR_D2PPRE1_DIV4_Pos  // 0x00000050
	RCC_D2CFGR_D2PPRE1_DIV4      = RCC_D2CFGR_D2PPRE1_DIV4_Msk         // APB1 clock divided by 4
	RCC_D2CFGR_D2PPRE1_DIV8_Pos  = 5                                   //
	RCC_D2CFGR_D2PPRE1_DIV8_Msk  = 0x3 << RCC_D2CFGR_D2PPRE1_DIV8_Pos  // 0x00000060
	RCC_D2CFGR_D2PPRE1_DIV8      = RCC_D2CFGR_D2PPRE1_DIV8_Msk         // APB1 clock divided by 8
	RCC_D2CFGR_D2PPRE1_DIV16_Pos = 4                                   //
	RCC_D2CFGR_D2PPRE1_DIV16_Msk = 0x7 << RCC_D2CFGR_D2PPRE1_DIV16_Pos // 0x00000070
	RCC_D2CFGR_D2PPRE1_DIV16     = RCC_D2CFGR_D2PPRE1_DIV16_Msk        // APB1 clock divided by 16

	RCC_D2CFGR_D2PPRE2   = RCC_D2CFGR_D2PPRE2_Msk        // D2PPRE2[2:0] bits (APB2 prescaler)
	RCC_D2CFGR_D2PPRE2_0 = 0x1 << RCC_D2CFGR_D2PPRE2_Pos // 0x00000100
	RCC_D2CFGR_D2PPRE2_1 = 0x2 << RCC_D2CFGR_D2PPRE2_Pos // 0x00000200
	RCC_D2CFGR_D2PPRE2_2 = 0x4 << RCC_D2CFGR_D2PPRE2_Pos // 0x00000400

	RCC_D2CFGR_D2PPRE2_DIV1      = 0                                   // APB2 clock not divided
	RCC_D2CFGR_D2PPRE2_DIV2_Pos  = 10                                  //
	RCC_D2CFGR_D2PPRE2_DIV2_Msk  = 0x1 << RCC_D2CFGR_D2PPRE2_DIV2_Pos  // 0x00000400
	RCC_D2CFGR_D2PPRE2_DIV2      = RCC_D2CFGR_D2PPRE2_DIV2_Msk         // APB2 clock divided by 2
	RCC_D2CFGR_D2PPRE2_DIV4_Pos  = 8                                   //
	RCC_D2CFGR_D2PPRE2_DIV4_Msk  = 0x5 << RCC_D2CFGR_D2PPRE2_DIV4_Pos  // 0x00000500
	RCC_D2CFGR_D2PPRE2_DIV4      = RCC_D2CFGR_D2PPRE2_DIV4_Msk         // APB2 clock divided by 4
	RCC_D2CFGR_D2PPRE2_DIV8_Pos  = 9                                   //
	RCC_D2CFGR_D2PPRE2_DIV8_Msk  = 0x3 << RCC_D2CFGR_D2PPRE2_DIV8_Pos  // 0x00000600
	RCC_D2CFGR_D2PPRE2_DIV8      = RCC_D2CFGR_D2PPRE2_DIV8_Msk         // APB2 clock divided by 8
	RCC_D2CFGR_D2PPRE2_DIV16_Pos = 8                                   //
	RCC_D2CFGR_D2PPRE2_DIV16_Msk = 0x7 << RCC_D2CFGR_D2PPRE2_DIV16_Pos // 0x00000700
	RCC_D2CFGR_D2PPRE2_DIV16     = RCC_D2CFGR_D2PPRE2_DIV16_Msk        // APB2 clock divided by 16

	// The auto-generated TinyGo source contains bitmasks for the wrong table
	// (RCC_AHB3ENR), and doesn't have those required for RCC_D3CFGR (which has
	// been recreated from STM32 HAL SDK below). Need to determine if this is a
	// bug with the SVD itself or the parser/generator.

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
	RCC_PLL1DIVR_N1_Pos = 0
	RCC_PLL1DIVR_N1_Msk = 0x1FF << RCC_PLL1DIVR_N1_Pos // 0x000001FF
	RCC_PLL1DIVR_N1     = RCC_PLL1DIVR_N1_Msk
	RCC_PLL1DIVR_P1_Pos = 9
	RCC_PLL1DIVR_P1_Msk = 0x7F << RCC_PLL1DIVR_P1_Pos // 0x0000FE00
	RCC_PLL1DIVR_P1     = RCC_PLL1DIVR_P1_Msk
	RCC_PLL1DIVR_Q1_Pos = 16
	RCC_PLL1DIVR_Q1_Msk = 0x7F << RCC_PLL1DIVR_Q1_Pos // 0x007F0000
	RCC_PLL1DIVR_Q1     = RCC_PLL1DIVR_Q1_Msk
	RCC_PLL1DIVR_R1_Pos = 24
	RCC_PLL1DIVR_R1_Msk = 0x7F << RCC_PLL1DIVR_R1_Pos // 0x7F000000
	RCC_PLL1DIVR_R1     = RCC_PLL1DIVR_R1_Msk
)

// Clock type identifiers used by RCC_CLK_CONFIG field CLK to specify which
// clock to use in (*RCC_Type).Configure().
const (
	RCC_CLK_SYSCLK  = 0x00000001
	RCC_CLK_HCLK    = 0x00000002
	RCC_CLK_D1PCLK1 = 0x00000004
	RCC_CLK_PCLK1   = 0x00000008
	RCC_CLK_PCLK2   = 0x00000010
	RCC_CLK_D3PCLK1 = 0x00000020
)

const (
	RCC_OSC_NONE  = 0x00000000
	RCC_OSC_HSE   = 0x00000001
	RCC_OSC_HSI   = 0x00000002
	RCC_OSC_LSE   = 0x00000004
	RCC_OSC_LSI   = 0x00000008
	RCC_OSC_CSI   = 0x00000010
	RCC_OSC_HSI48 = 0x00000020
)

const (
	RCC_SYSCLK_SRC_HSI = 0x00000000
	RCC_SYSCLK_SRC_CSI = 0x00000001
	RCC_SYSCLK_SRC_HSE = 0x00000002
	RCC_SYSCLK_SRC_PLL = 0x00000003
)

const (
	RCC_PLL_SRC_HSI  = 0x00000000
	RCC_PLL_SRC_CSI  = 0x00000001
	RCC_PLL_SRC_HSE  = 0x00000002
	RCC_PLL_SRC_NONE = 0x00000003
)

// DO NOT USE THESE - they are pre-shifted into register (CFGR) position, but
// the values used internally in this source file are unshifted values.
// Instead, use the constant values with prefix "RCC_SYSCLK_SRC_" when comparing
// to any existing variables representing "SYSCLK source timebase" (such as
// member fields of RCC_CLK_CONFIG and RCC_OSC_CONFIG structs)
// const (
// 	RCC_CFGR_SWS_HSI  = 0x00000000 // HSI used as system clock
// 	RCC_CFGR_SWS_CSI  = 0x00000008 // CSI used as system clock
// 	RCC_CFGR_SWS_HSE  = 0x00000010 // HSE used as system clock
// 	RCC_CFGR_SWS_PLL1 = 0x00000018 // PLL1 used as system clock
// )

const (
	RCC_HSE_OFF    = 0
	RCC_HSE_ON     = RCC_CR_HSEON
	RCC_HSE_BYPASS = RCC_CR_HSEBYP | RCC_CR_HSEON

	RCC_LSE_OFF    = 0
	RCC_LSE_ON     = RCC_BDCR_LSEON
	RCC_LSE_BYPASS = RCC_BDCR_LSEBYP | RCC_BDCR_LSEON

	RCC_HSI_OFF = 0            // HSI clock deactivation
	RCC_HSI_ON  = RCC_CR_HSION // HSI clock activation

	RCC_HSI48_OFF = 0
	RCC_HSI48_ON  = 1

	RCC_LSI_OFF = 0
	RCC_LSI_ON  = RCC_CSR_LSION

	RCC_CSI_OFF = 0
	RCC_CSI_ON  = RCC_CR_CSION

	RCC_PLL_NONE = 0
	RCC_PLL_OFF  = 1
	RCC_PLL_ON   = 2
)

const (
	RCC_SYSCLK_DIV1   = RCC_D1CFGR_D1CPRE_DIV1
	RCC_SYSCLK_DIV2   = RCC_D1CFGR_D1CPRE_DIV2
	RCC_SYSCLK_DIV4   = RCC_D1CFGR_D1CPRE_DIV4
	RCC_SYSCLK_DIV8   = RCC_D1CFGR_D1CPRE_DIV8
	RCC_SYSCLK_DIV16  = RCC_D1CFGR_D1CPRE_DIV16
	RCC_SYSCLK_DIV64  = RCC_D1CFGR_D1CPRE_DIV64
	RCC_SYSCLK_DIV128 = RCC_D1CFGR_D1CPRE_DIV128
	RCC_SYSCLK_DIV256 = RCC_D1CFGR_D1CPRE_DIV256
	RCC_SYSCLK_DIV512 = RCC_D1CFGR_D1CPRE_DIV512

	RCC_HCLK_DIV1   = RCC_D1CFGR_HPRE_DIV1
	RCC_HCLK_DIV2   = RCC_D1CFGR_HPRE_DIV2
	RCC_HCLK_DIV4   = RCC_D1CFGR_HPRE_DIV4
	RCC_HCLK_DIV8   = RCC_D1CFGR_HPRE_DIV8
	RCC_HCLK_DIV16  = RCC_D1CFGR_HPRE_DIV16
	RCC_HCLK_DIV64  = RCC_D1CFGR_HPRE_DIV64
	RCC_HCLK_DIV128 = RCC_D1CFGR_HPRE_DIV128
	RCC_HCLK_DIV256 = RCC_D1CFGR_HPRE_DIV256
	RCC_HCLK_DIV512 = RCC_D1CFGR_HPRE_DIV512

	RCC_APB1_DIV1  = RCC_D2CFGR_D2PPRE1_DIV1
	RCC_APB1_DIV2  = RCC_D2CFGR_D2PPRE1_DIV2
	RCC_APB1_DIV4  = RCC_D2CFGR_D2PPRE1_DIV4
	RCC_APB1_DIV8  = RCC_D2CFGR_D2PPRE1_DIV8
	RCC_APB1_DIV16 = RCC_D2CFGR_D2PPRE1_DIV16

	RCC_APB2_DIV1  = RCC_D2CFGR_D2PPRE2_DIV1
	RCC_APB2_DIV2  = RCC_D2CFGR_D2PPRE2_DIV2
	RCC_APB2_DIV4  = RCC_D2CFGR_D2PPRE2_DIV4
	RCC_APB2_DIV8  = RCC_D2CFGR_D2PPRE2_DIV8
	RCC_APB2_DIV16 = RCC_D2CFGR_D2PPRE2_DIV16

	RCC_APB3_DIV1  = RCC_D1CFGR_D1PPRE_DIV1
	RCC_APB3_DIV2  = RCC_D1CFGR_D1PPRE_DIV2
	RCC_APB3_DIV4  = RCC_D1CFGR_D1PPRE_DIV4
	RCC_APB3_DIV8  = RCC_D1CFGR_D1PPRE_DIV8
	RCC_APB3_DIV16 = RCC_D1CFGR_D1PPRE_DIV16

	RCC_APB4_DIV1  = RCC_D3CFGR_D3PPRE_DIV1
	RCC_APB4_DIV2  = RCC_D3CFGR_D3PPRE_DIV2
	RCC_APB4_DIV4  = RCC_D3CFGR_D3PPRE_DIV4
	RCC_APB4_DIV8  = RCC_D3CFGR_D3PPRE_DIV8
	RCC_APB4_DIV16 = RCC_D3CFGR_D3PPRE_DIV16

	RCC_HSI_DIV1 = (0x0 << RCC_CR_HSIDIV_Pos) | RCC_CR_HSION // 0x00000000: HSI_DIV1 clock activation
	RCC_HSI_DIV2 = (0x1 << RCC_CR_HSIDIV_Pos) | RCC_CR_HSION // 0x00000008: HSI_DIV2 clock activation
	RCC_HSI_DIV4 = (0x2 << RCC_CR_HSIDIV_Pos) | RCC_CR_HSION // 0x00000010: HSI_DIV4 clock activation
	RCC_HSI_DIV8 = (0x3 << RCC_CR_HSIDIV_Pos) | RCC_CR_HSION // 0x00000018: HSI_DIV8 clock activation
)

const (
	RCC_CSI_TRIM_DEFAULT = 0x20
	RCC_HSI_TRIM_DEFAULT = 0x40

	HAL_RCC_REV_Y_HSITRIM_Pos = 12
	HAL_RCC_REV_Y_HSITRIM_Msk = 0x3F000
	HAL_RCC_REV_Y_CSITRIM_Pos = 26
	HAL_RCC_REV_Y_CSITRIM_Msk = 0x7C000000
)

const (
	RCC_PLL1_VCI_RANGE_0 = 0x0 << RCC_PLLCFGR_PLL1RGE_Pos // 0x00000000: Clock range frequency between 1 and 2 MHz
	RCC_PLL1_VCI_RANGE_1 = 0x1 << RCC_PLLCFGR_PLL1RGE_Pos // 0x00000004: Clock range frequency between 2 and 4 MHz
	RCC_PLL1_VCI_RANGE_2 = 0x2 << RCC_PLLCFGR_PLL1RGE_Pos // 0x00000008: Clock range frequency between 4 and 8 MHz
	RCC_PLL1_VCI_RANGE_3 = 0x3 << RCC_PLLCFGR_PLL1RGE_Pos // 0x0000000C: Clock range frequency between 8 and 16 MHz

	RCC_PLL1_VCO_WIDE   = 0
	RCC_PLL1_VCO_MEDIUM = RCC_PLLCFGR_PLL1VCOSEL
)

const (
	RCC_AHB3ENR_FLASHEN_Pos = 8
	RCC_AHB3ENR_FLASHEN_Msk = 0x1 << RCC_AHB3ENR_FLASHEN_Pos // 0x00000100
	RCC_AHB3ENR_FLASHEN     = RCC_AHB3ENR_FLASHEN_Msk
)

// Default oscillator frequencies
const (
	RCC_HSE_FREQ_HZ uint32 = 25000000
	RCC_CSI_FREQ_HZ uint32 = 4000000
	RCC_HSI_FREQ_HZ uint32 = 64000000
)

const (
	rccClockSwitchTimeoutMs = 5000 // 5 sec
	rccHSETimeoutMs         = rccClockSwitchTimeoutMs
	rccHSITimeoutMs         = 2 // 2 ms
	rccHSI48TimeoutMs       = 2 // 2 ms
	rccCSITimeoutMs         = 2 // 2 ms
	rccLSITimeoutMs         = 2 // 2 ms
	rccPLLTimeoutMs         = 2 // 2 ms
	rccDBPTimeoutMs         = 100
	rccLSETimeoutMs         = rccClockSwitchTimeoutMs
)

//const (
//	rccPLLNone       = 1 << iota
//	rccPLLHSI        // Use HSI internal clock
//	rccPLLHSECrystal // Use external xtal (x3 on-board)
//	rccPLLHSEClock   // Use external clock
//)

// System core frequency prescalar table
var rccFreqPrescalar = [...]uint8{
	0, 0, 0, 0, 1, 2, 3, 4, 1, 2, 3, 4, 6, 7, 8, 9,
}

// ConfigOsc configures the system oscillators using the given oscillator
// configuration struct.
func (rcc *RCC_Type) ConfigOsc(config RCC_OSC_CONFIG) bool {

	sysSrc, pllSrc :=
		(rcc.CFGR.Get()&RCC_CFGR_SWS_Msk)>>RCC_CFGR_SWS_Pos,
		(rcc.PLLCKSELR.Get()&RCC_PLLCKSELR_PLLSRC_Msk)>>RCC_PLLCKSELR_PLLSRC_Pos

	// Configure HSE
	if config.OSC&RCC_OSC_HSE == RCC_OSC_HSE {
		if (RCC_SYSCLK_SRC_HSE == sysSrc) ||
			(RCC_SYSCLK_SRC_PLL == sysSrc && RCC_PLL_SRC_HSE == pllSrc) {
			// HSE is currently source of SYSCLK.
			if RCC_FLAG_HSERDY.Get() && config.HSE == RCC_HSE_OFF {
				return false // cannot disable HSE while driving SYSCLK
			}
		} else {
			// HSE is currently NOT source of SYSCLK.
			// Configure HSE according to its state given in configuration struct.
			switch config.HSE {
			case RCC_HSE_ON:
				rcc.CR.SetBits(RCC_CR_HSEON)
			case RCC_HSE_OFF:
				rcc.CR.ClearBits(RCC_CR_HSEON)
				rcc.CR.ClearBits(RCC_CR_HSEBYP)
			case RCC_HSE_BYPASS:
				rcc.CR.SetBits(RCC_CR_HSEBYP)
				rcc.CR.SetBits(RCC_CR_HSEON)
			default:
				rcc.CR.ClearBits(RCC_CR_HSEON)
				rcc.CR.ClearBits(RCC_CR_HSEBYP)
			}
			// Wait for new oscillator configuration to take effect within a fixed
			// window of time; if time expires, return early with error condition.
			if config.HSE != RCC_HSE_OFF {
				start := ticks()
				for !RCC_FLAG_HSERDY.Get() {
					if ticks()-start > rccHSETimeoutMs {
						return false
					}
				}
			} else {
				start := ticks()
				for RCC_FLAG_HSERDY.Get() {
					if ticks()-start > rccHSETimeoutMs {
						return false
					}
				}
			}
		}
	}
	// Configure HSI
	if config.OSC&RCC_OSC_HSI == RCC_OSC_HSI {
		if (RCC_SYSCLK_SRC_HSI == sysSrc) ||
			(RCC_SYSCLK_SRC_PLL == sysSrc && RCC_PLL_SRC_HSI == pllSrc) {
			// HSI is currently source of SYSCLK
			if RCC_FLAG_HSIRDY.Get() && config.HSI == RCC_HSI_OFF {
				return false // cannot disable HSI while driving SYSCLK
			} else {
				// apply HSI calibration factor
				if DBG.MCURevision() <= DBG_MCU_REVISION_Y {
					if config.HSITrim == RCC_HSI_TRIM_DEFAULT {
						rcc.HSICFGR.ReplaceBits(
							0x20<<HAL_RCC_REV_Y_HSITRIM_Pos,
							HAL_RCC_REV_Y_HSITRIM_Msk, 0)
					} else {
						rcc.HSICFGR.ReplaceBits(
							config.HSITrim<<HAL_RCC_REV_Y_HSITRIM_Pos,
							HAL_RCC_REV_Y_HSITRIM_Msk, 0)
					}
				} else {
					rcc.HSICFGR.ReplaceBits(
						config.HSITrim<<RCC_HSICFGR_HSITRIM_Pos,
						RCC_HSICFGR_HSITRIM_Msk, 0)
				}
			}
		} else {
			// HSI is currently NOT source of SYSCLK.
			// Configure HSI according to its state given in configuration struct.
			if config.HSI != RCC_HSI_OFF {
				// Enable HSI
				rcc.CR.ReplaceBits(config.HSI, RCC_CR_HSION_Msk|RCC_CR_HSIDIV_Msk, 0)
				// Verify HSI was enabled
				start := ticks()
				for !RCC_FLAG_HSIRDY.Get() {
					if ticks()-start > rccHSITimeoutMs {
						return false
					}
				}
				// apply HSI calibration factor
				if DBG.MCURevision() <= DBG_MCU_REVISION_Y {
					if config.HSITrim == RCC_HSI_TRIM_DEFAULT {
						rcc.HSICFGR.ReplaceBits(
							0x20<<HAL_RCC_REV_Y_HSITRIM_Pos,
							HAL_RCC_REV_Y_HSITRIM_Msk, 0)
					} else {
						rcc.HSICFGR.ReplaceBits(
							config.HSITrim<<HAL_RCC_REV_Y_HSITRIM_Pos,
							HAL_RCC_REV_Y_HSITRIM_Msk, 0)
					}
				} else {
					rcc.HSICFGR.ReplaceBits(
						config.HSITrim<<RCC_HSICFGR_HSITRIM_Pos,
						RCC_HSICFGR_HSITRIM_Msk, 0)
				}
			} else {
				// Disable HSI
				rcc.CR.ClearBits(RCC_CR_HSION)
				// Verify HSI was disabled
				start := ticks()
				for RCC_FLAG_HSIRDY.Get() {
					if ticks()-start > rccHSITimeoutMs {
						return false
					}
				}
			}
		}
	}
	// Configure CSI
	if config.OSC&RCC_OSC_CSI == RCC_OSC_CSI {
		if (RCC_SYSCLK_SRC_CSI == sysSrc) ||
			(RCC_SYSCLK_SRC_PLL == sysSrc && RCC_PLL_SRC_CSI == pllSrc) {
			// CSI is currently source of SYSCLK
			if RCC_FLAG_CSIRDY.Get() && config.CSI == RCC_CSI_OFF {
				return false // cannot disable HSI while driving SYSCLK
			} else {
				// apply CSI calibration factor
				if DBG.MCURevision() <= DBG_MCU_REVISION_Y {
					if config.CSITrim == RCC_CSI_TRIM_DEFAULT {
						rcc.CSICFGR.ReplaceBits(
							0x10<<HAL_RCC_REV_Y_CSITRIM_Pos,
							HAL_RCC_REV_Y_CSITRIM_Msk, 0)
					} else {
						rcc.CSICFGR.ReplaceBits(
							config.CSITrim<<HAL_RCC_REV_Y_CSITRIM_Pos,
							HAL_RCC_REV_Y_CSITRIM_Msk, 0)
					}
				} else {
					rcc.CSICFGR.ReplaceBits(
						config.CSITrim<<RCC_CSICFGR_CSITRIM_Pos,
						RCC_CSICFGR_CSITRIM_Msk, 0)
				}
			}
		} else {
			// CSI is currently NOT source of SYSCLK.
			// Configure CSI according to its state given in configuration struct.
			if config.CSI != RCC_CSI_OFF {
				// Enable CSI
				rcc.CR.SetBits(RCC_CR_CSION)
				// Verify CSI was enabled
				start := ticks()
				for !RCC_FLAG_CSIRDY.Get() {
					if ticks()-start > rccCSITimeoutMs {
						return false
					}
				}
				// apply CSI calibration factor
				if DBG.MCURevision() <= DBG_MCU_REVISION_Y {
					if config.CSITrim == RCC_CSI_TRIM_DEFAULT {
						rcc.CSICFGR.ReplaceBits(
							0x10<<HAL_RCC_REV_Y_CSITRIM_Pos,
							HAL_RCC_REV_Y_CSITRIM_Msk, 0)
					} else {
						rcc.CSICFGR.ReplaceBits(
							config.CSITrim<<HAL_RCC_REV_Y_CSITRIM_Pos,
							HAL_RCC_REV_Y_CSITRIM_Msk, 0)
					}
				} else {
					rcc.CSICFGR.ReplaceBits(
						config.CSITrim<<RCC_CSICFGR_CSITRIM_Pos,
						RCC_CSICFGR_CSITRIM_Msk, 0)
				}
			} else {
				// Disable CSI
				rcc.CR.ClearBits(RCC_CR_CSION)
				// Verify CSI was disabled
				start := ticks()
				for RCC_FLAG_CSIRDY.Get() {
					if ticks()-start > rccCSITimeoutMs {
						return false
					}
				}
			}
		}
	}
	// Configure LSI
	if config.OSC&RCC_OSC_LSI == RCC_OSC_LSI {
		if config.LSI != RCC_LSI_OFF {
			// Enable LSI
			rcc.CSR.SetBits(RCC_CSR_LSION)
			// Verify LSI was enabled
			start := ticks()
			for !RCC_FLAG_LSIRDY.Get() {
				if ticks()-start > rccLSITimeoutMs {
					return false
				}
			}
		} else {
			// Disable LSI
			rcc.CSR.ClearBits(RCC_CSR_LSION)
			// Verify LSI was disabled
			start := ticks()
			for RCC_FLAG_LSIRDY.Get() {
				if ticks()-start > rccLSITimeoutMs {
					return false
				}
			}
		}
	}
	// Configure HSI48
	if config.OSC&RCC_OSC_HSI48 == RCC_OSC_HSI48 {
		if config.HSI48 != RCC_HSI48_OFF {
			// Enable HSI48
			rcc.CR.SetBits(RCC_CR_RC48ON)
			// Verify HSI48 was enabled
			start := ticks()
			for !RCC_FLAG_HSI48RDY.Get() {
				if ticks()-start > rccHSI48TimeoutMs {
					return false
				}
			}
		} else {
			// Disable HSI48
			rcc.CR.ClearBits(RCC_CR_RC48ON)
			// Verify HSI48 was disabled
			start := ticks()
			for RCC_FLAG_HSI48RDY.Get() {
				if ticks()-start > rccHSI48TimeoutMs {
					return false
				}
			}
		}
	}
	// Configure LSE
	if config.OSC&RCC_OSC_LSE == RCC_OSC_LSE {
		// ensure write access to backup domain
		PWR.PWR_CR1.SetBits(PWR_PWR_CR1_DBP)
		// bail out with error condition if write protection can't be removed.
		start := ticks()
		for !PWR.PWR_CR1.HasBits(PWR_PWR_CR1_DBP) {
			if ticks()-start > rccDBPTimeoutMs {
				return false
			}
		}
		// Configure LSE according to its state given in configuration struct.
		switch config.LSE {
		case RCC_LSE_ON:
			rcc.BDCR.SetBits(RCC_BDCR_LSEON)
		case RCC_LSE_OFF:
			rcc.BDCR.ClearBits(RCC_BDCR_LSEON)
			rcc.BDCR.ClearBits(RCC_BDCR_LSEBYP)
		case RCC_LSE_BYPASS:
			rcc.BDCR.SetBits(RCC_BDCR_LSEBYP)
			rcc.BDCR.SetBits(RCC_BDCR_LSEON)
		default:
			rcc.BDCR.ClearBits(RCC_BDCR_LSEON)
			rcc.BDCR.ClearBits(RCC_BDCR_LSEBYP)
		}
		// Wait for new oscillator configuration to take effect within a fixed
		// window of time; if time expires, return early with error condition.
		if config.LSE != RCC_LSE_OFF {
			start := ticks()
			for !RCC_FLAG_LSERDY.Get() {
				if ticks()-start > rccLSETimeoutMs {
					return false
				}
			}
		} else {
			// Verify LSE was disabled
			start := ticks()
			for RCC_FLAG_LSERDY.Get() {
				if ticks()-start > rccLSETimeoutMs {
					return false
				}
			}
		}
	}
	// Configure PLL
	if config.PLL.State != RCC_PLL_NONE {
		// Check if PLL is currently being used as source of SYSCLK.
		if sysSrc != RCC_SYSCLK_SRC_PLL {
			// PLL is currently NOT source of SYSCLK.
			// Check if we are requesting to enabled or disable PLL.
			if config.PLL.State == RCC_PLL_ON {
				// Enabling PLL ...
				// First, disable PLL prior to reconfiguring
				rcc.CR.ClearBits(RCC_CR_PLL1ON)
				// Wait for PLL to disable
				start := ticks()
				for RCC_FLAG_PLLRDY.Get() {
					if ticks()-start > rccPLLTimeoutMs {
						return false
					}
				}

				// Configure PLL clock source
				rcc.PLLCKSELR.ReplaceBits(
					(config.PLL.Src<<RCC_PLLCKSELR_PLLSRC_Pos)|
						(config.PLL.Param.M<<RCC_PLLCKSELR_DIVM1_Pos),
					(RCC_PLLCKSELR_PLLSRC_Msk | RCC_PLLCKSELR_DIVM1_Msk), 0)

				// Configure PLL multiplication and division factors
				n := ((config.PLL.Param.N - 1) << RCC_PLL1DIVR_DIVN1_Pos) & RCC_PLL1DIVR_DIVN1_Msk
				p := ((config.PLL.Param.P - 1) << RCC_PLL1DIVR_DIVP1_Pos) & RCC_PLL1DIVR_DIVP1_Msk
				q := ((config.PLL.Param.Q - 1) << RCC_PLL1DIVR_DIVQ1_Pos) & RCC_PLL1DIVR_DIVQ1_Msk
				r := ((config.PLL.Param.R - 1) << RCC_PLL1DIVR_DIVR1_Pos) & RCC_PLL1DIVR_DIVR1_Msk

				rcc.PLL1DIVR.Set(n | p | q | r)

				rcc.PLLCFGR.ClearBits(RCC_PLLCFGR_PLL1FRACEN)

				rcc.PLL1FRACR.ReplaceBits(
					config.PLL.Param.Frac<<RCC_PLL1FRACR_FRACN1_Pos,
					RCC_PLL1FRACR_FRACN1_Msk, 0)

				rcc.PLLCFGR.ReplaceBits(
					config.PLL.Param.In,
					RCC_PLLCFGR_PLL1RGE_Msk, 0)

				rcc.PLLCFGR.ReplaceBits(
					config.PLL.Param.Out,
					RCC_PLLCFGR_PLL1VCOSEL_Msk, 0)

				rcc.PLLCFGR.SetBits((RCC_PLLCFGR_DIVP1EN))
				rcc.PLLCFGR.SetBits((RCC_PLLCFGR_DIVQ1EN))
				rcc.PLLCFGR.SetBits((RCC_PLLCFGR_DIVR1EN))
				rcc.PLLCFGR.SetBits(RCC_PLLCFGR_PLL1FRACEN)
				rcc.CR.SetBits(RCC_CR_PLL1ON)
				start = ticks()
				for !RCC_FLAG_PLLRDY.Get() {
					if ticks()-start > rccPLLTimeoutMs {
						return false
					}
				}

			} else {
				// Disabling PLL ...
				// Clear PLL enable bits
				rcc.CR.ClearBits(RCC_CR_PLL1ON)
				// Verify PLL was disabled
				start := ticks()
				for RCC_FLAG_PLLRDY.Get() {
					if ticks()-start > rccPLLTimeoutMs {
						return false
					}
				}
			}
		} else {
			// PLL is currently source of SYSCLK
		}
	}
	return true
}

// ConfigClk configures the CPU, AHB, and APB bus clocks using the given clock
// configuration struct.
func (rcc *RCC_Type) ConfigClk(config RCC_CLK_CONFIG, latency uint32) bool {

	if latency > FLASH.ACR.Get()&FLASH_ACR_LATENCY_Msk {
		FLASH.ACR.ReplaceBits(latency, FLASH_ACR_LATENCY_Msk, 0)
		if latency != FLASH.ACR.Get()&FLASH_ACR_LATENCY_Msk {
			return false
		}
	}
	if config.CLK&RCC_CLK_D1PCLK1 == RCC_CLK_D1PCLK1 {
		if config.APB3Div > RCC.D1CFGR.Get()&RCC_D1CFGR_D1PPRE_Msk {
			RCC.D1CFGR.ReplaceBits(config.APB3Div, RCC_D1CFGR_D1PPRE_Msk, 0)
		}
	}
	if config.CLK&RCC_CLK_PCLK1 == RCC_CLK_PCLK1 {
		if config.APB1Div > RCC.D2CFGR.Get()&RCC_D2CFGR_D2PPRE1_Msk {
			RCC.D2CFGR.ReplaceBits(config.APB1Div, RCC_D2CFGR_D2PPRE1_Msk, 0)
		}
	}
	if config.CLK&RCC_CLK_PCLK2 == RCC_CLK_PCLK2 {
		if config.APB2Div > RCC.D2CFGR.Get()&RCC_D2CFGR_D2PPRE2_Msk {
			RCC.D2CFGR.ReplaceBits(config.APB2Div, RCC_D2CFGR_D2PPRE2_Msk, 0)
		}
	}
	if config.CLK&RCC_CLK_D3PCLK1 == RCC_CLK_D3PCLK1 {
		// See notes below regarding TinyGo source and MCU table "RCC_D3CFGR".
		if config.APB4Div > RCC.D3CFGR.Get()&RCC_D3CFGR_D3PPRE {
			RCC.D3CFGR.ReplaceBits(config.APB4Div, RCC_D3CFGR_D3PPRE, 0)
		}
	}
	if config.CLK&RCC_CLK_HCLK == RCC_CLK_HCLK {
		if config.AHBDiv > RCC.D1CFGR.Get()&RCC_D1CFGR_HPRE_Msk {
			RCC.D1CFGR.ReplaceBits(config.AHBDiv, RCC_D1CFGR_HPRE_Msk, 0)
		}
	}
	if config.CLK&RCC_CLK_SYSCLK == RCC_CLK_SYSCLK {
		RCC.D1CFGR.ReplaceBits(config.SYSDiv, RCC_D1CFGR_D1CPRE_Msk, 0)
		var flag RCC_FLAG
		switch config.SYSSrc {
		case RCC_SYSCLK_SRC_HSI:
			flag = RCC_FLAG_HSIRDY
		case RCC_SYSCLK_SRC_CSI:
			flag = RCC_FLAG_CSIRDY
		case RCC_SYSCLK_SRC_HSE:
			flag = RCC_FLAG_HSERDY
		case RCC_SYSCLK_SRC_PLL:
			flag = RCC_FLAG_HSERDY
		default:
			return false
		}
		if !flag.Get() {
			return false
		}
		RCC.CFGR.ReplaceBits(config.SYSSrc, RCC_CFGR_SW_Msk, 0)
		start := ticks()
		for (rcc.CFGR.Get()&RCC_CFGR_SWS_Msk)>>RCC_CFGR_SWS_Pos != config.SYSSrc {
			if ticks()-start > rccClockSwitchTimeoutMs {
				return false
			}
		}
	}
	if config.CLK&RCC_CLK_HCLK == RCC_CLK_HCLK {
		if config.AHBDiv < RCC.D1CFGR.Get()&RCC_D1CFGR_HPRE_Msk {
			RCC.D1CFGR.ReplaceBits(config.AHBDiv, RCC_D1CFGR_HPRE_Msk, 0)
		}
	}
	if latency < FLASH.ACR.Get()&FLASH_ACR_LATENCY_Msk {
		FLASH.ACR.ReplaceBits(latency, FLASH_ACR_LATENCY_Msk, 0)
		if latency != FLASH.ACR.Get()&FLASH_ACR_LATENCY_Msk {
			return false
		}
	}
	if config.CLK&RCC_CLK_D1PCLK1 == RCC_CLK_D1PCLK1 {
		if config.APB3Div < RCC.D1CFGR.Get()&RCC_D1CFGR_D1PPRE_Msk {
			RCC.D1CFGR.ReplaceBits(config.APB3Div, RCC_D1CFGR_D1PPRE_Msk, 0)
		}
	}
	if config.CLK&RCC_CLK_PCLK1 == RCC_CLK_PCLK1 {
		if config.APB1Div < RCC.D2CFGR.Get()&RCC_D2CFGR_D2PPRE1_Msk {
			RCC.D2CFGR.ReplaceBits(config.APB1Div, RCC_D2CFGR_D2PPRE1_Msk, 0)
		}
	}
	if config.CLK&RCC_CLK_PCLK2 == RCC_CLK_PCLK2 {
		if config.APB2Div < RCC.D2CFGR.Get()&RCC_D2CFGR_D2PPRE2_Msk {
			RCC.D2CFGR.ReplaceBits(config.APB2Div, RCC_D2CFGR_D2PPRE2_Msk, 0)
		}
	}
	if config.CLK&RCC_CLK_D3PCLK1 == RCC_CLK_D3PCLK1 {
		// See notes below regarding TinyGo source and MCU table "RCC_D3CFGR".
		if config.APB4Div < RCC.D3CFGR.Get()&RCC_D3CFGR_D3PPRE {
			RCC.D3CFGR.ReplaceBits(config.APB4Div, RCC_D3CFGR_D3PPRE, 0)
		}
	}

	return true
}

// ClockFreq returns the current frequency of all internal CPU clocks.
func (rcc *RCC_Type) ClockFreq() (clk RCC_CLOCKS) {
	clk.SYSCLK = rcc.sysclkFreq()
	clk.HCLK = clk.SYSCLK >> (rccFreqPrescalar[(((rcc.D1CFGR.Get()&
		RCC_D1CFGR_HPRE_Msk)>>RCC_D1CFGR_HPRE_Pos)&RCC_D1CFGR_HPRE_Msk)>>
		RCC_D1CFGR_HPRE_Pos] & 0x1F)
	clk.PCLK1 = clk.HCLK >> (rccFreqPrescalar[(((rcc.D2CFGR.Get()&
		RCC_D2CFGR_D2PPRE1_Msk)>>RCC_D2CFGR_D2PPRE1_Pos)&RCC_D2CFGR_D2PPRE1_Msk)>>
		RCC_D2CFGR_D2PPRE1_Pos] & 0x1F)
	clk.PCLK2 = clk.HCLK >> (rccFreqPrescalar[(((rcc.D2CFGR.Get()&
		RCC_D2CFGR_D2PPRE2_Msk)>>RCC_D2CFGR_D2PPRE2_Pos)&RCC_D2CFGR_D2PPRE2_Msk)>>
		RCC_D2CFGR_D2PPRE2_Pos] & 0x1F)
	clk.PCLK3 = clk.HCLK >> (rccFreqPrescalar[(((rcc.D1CFGR.Get()&
		RCC_D1CFGR_D1PPRE_Msk)>>RCC_D1CFGR_D1PPRE_Pos)&RCC_D1CFGR_D1PPRE_Msk)>>
		RCC_D1CFGR_D1PPRE_Pos] & 0x1F)
	clk.PCLK4 = clk.HCLK >> (rccFreqPrescalar[(((rcc.D3CFGR.Get()&
		RCC_D3CFGR_D3PPRE_Msk)>>RCC_D3CFGR_D3PPRE_Pos)&RCC_D3CFGR_D3PPRE_Msk)>>
		RCC_D3CFGR_D3PPRE_Pos] & 0x1F)
	return
}

func (rcc *RCC_Type) sysclkFreq() uint32 {
	switch (rcc.CFGR.Get() & RCC_CFGR_SWS_Msk) >> RCC_CFGR_SWS_Pos {
	case RCC_SYSCLK_SRC_HSI:
		div := (rcc.CR.Get() & RCC_CR_HSIDIV_Msk) >> RCC_CR_HSIDIV_Pos
		return RCC_HSI_FREQ_HZ >> div
	case RCC_SYSCLK_SRC_CSI:
		return RCC_CSI_FREQ_HZ
	case RCC_SYSCLK_SRC_HSE:
		return RCC_HSE_FREQ_HZ
	case RCC_SYSCLK_SRC_PLL:
		clk := rcc.pll1Freq()
		return clk.PLL_P
	}
	return 0
}

func (rcc *RCC_Type) pll1Freq() (clk RCC_PLL_CLOCKS) {

	var in uint32
	switch (rcc.PLLCKSELR.Get() & RCC_PLLCKSELR_PLLSRC_Msk) >>
		RCC_PLLCKSELR_PLLSRC_Pos {
	case RCC_PLL_SRC_HSI:
		if rcc.CR.HasBits(RCC_CR_HSIRDY) {
			div := (rcc.CR.Get() & RCC_CR_HSIDIV_Msk) >> RCC_CR_HSIDIV_Pos
			in = RCC_HSI_FREQ_HZ >> div
		}
	case RCC_PLL_SRC_CSI:
		if rcc.CR.HasBits(RCC_CR_CSIRDY) {
			in = RCC_CSI_FREQ_HZ
		}
	case RCC_PLL_SRC_HSE:
		if rcc.CR.HasBits(RCC_CR_HSERDY) {
			in = RCC_HSE_FREQ_HZ
		}
	case RCC_PLL_SRC_NONE:
		// PLL disabled
	}

	clk.PLL_P = 0
	clk.PLL_Q = 0
	clk.PLL_R = 0

	m := (rcc.PLLCKSELR.Get() & RCC_PLLCKSELR_DIVM1_Msk) >>
		RCC_PLLCKSELR_DIVM1_Pos
	n := ((rcc.PLL1DIVR.Get() & RCC_PLL1DIVR_DIVN1_Msk) >>
		RCC_PLL1DIVR_DIVN1_Pos) + 1

	var fracn uint32
	if rcc.PLLCFGR.HasBits(RCC_PLLCFGR_PLL1FRACEN) {
		fracn = (rcc.PLL1FRACR.Get() & RCC_PLL1FRACR_FRACN1_Msk) >>
			RCC_PLL1FRACR_FRACN1_Pos
	}

	if 0 != m {
		if rcc.PLLCFGR.HasBits(RCC_PLLCFGR_DIVP1EN) {
			p := ((rcc.PLL1DIVR.Get() & RCC_PLL1DIVR_DIVP1_Msk) >>
				RCC_PLL1DIVR_DIVP1_Pos) + 1
			clk.PLL_P = calcPLLFreq(in, m, n, fracn, p)
		}
		if rcc.PLLCFGR.HasBits(RCC_PLLCFGR_DIVQ1EN) {
			q := ((rcc.PLL1DIVR.Get() & RCC_PLL1DIVR_DIVQ1_Msk) >>
				RCC_PLL1DIVR_DIVQ1_Pos) + 1
			clk.PLL_Q = calcPLLFreq(in, m, n, fracn, q)
		}
		if rcc.PLLCFGR.HasBits(RCC_PLLCFGR_DIVR1EN) {
			r := ((rcc.PLL1DIVR.Get() & RCC_PLL1DIVR_DIVR1_Msk) >>
				RCC_PLL1DIVR_DIVR1_Pos) + 1
			clk.PLL_R = calcPLLFreq(in, m, n, fracn, r)
		}
	}

	return clk
}

func calcPLLFreq(in, m, n, fracn, pqr uint32) (freq uint32) {

	// We are trying to find H, where:
	//   S := HSE or HSI or CSI (based on PLL source)
	//   H := (S/M * (N + F/8192)) / P
	//
	// return uint32(((float32(in) / float32(m) *
	// 	(float32(n) + (float32(fracn) / 0x2000))) / float32(pqr)))
	//
	// BUG(?): The compiler builds and links this code just fine, with obvious
	//         FP instructions in the disassembly, but we always get a hardfault
	//         at runtime when this function is called from ANY context.
	//         I know we don't currently support hardware FPU, but shouldn't it be
	//         able to use softfp?

	return uint32(uint64(in) * uint64(8192*n+fracn) / uint64(8192*m*pqr))
}

type RCC_FLAG uint8

const (
	RCC_FLAG_MASK     RCC_FLAG = 0x1F
	RCC_FLAG_HSIRDY   RCC_FLAG = 0x22
	RCC_FLAG_HSIDIV   RCC_FLAG = 0x25
	RCC_FLAG_CSIRDY   RCC_FLAG = 0x28
	RCC_FLAG_HSI48RDY RCC_FLAG = 0x2D
	RCC_FLAG_D1CKRDY  RCC_FLAG = 0x2E
	RCC_FLAG_CPUCKRDY RCC_FLAG = 0x2E
	RCC_FLAG_D2CKRDY  RCC_FLAG = 0x2F
	RCC_FLAG_CDCKRDY  RCC_FLAG = 0x2F
	RCC_FLAG_HSERDY   RCC_FLAG = 0x31
	RCC_FLAG_PLLRDY   RCC_FLAG = 0x39
	RCC_FLAG_PLL2RDY  RCC_FLAG = 0x3B
	RCC_FLAG_PLL3RDY  RCC_FLAG = 0x3D
	RCC_FLAG_LSERDY   RCC_FLAG = 0x41
	RCC_FLAG_LSIRDY   RCC_FLAG = 0x61
	RCC_FLAG_CPURST   RCC_FLAG = 0x91
	RCC_FLAG_D1RST    RCC_FLAG = 0x93
	RCC_FLAG_CDRST    RCC_FLAG = 0x93
	RCC_FLAG_D2RST    RCC_FLAG = 0x94
	RCC_FLAG_BORRST   RCC_FLAG = 0x95
	RCC_FLAG_PINRST   RCC_FLAG = 0x96
	RCC_FLAG_PORRST   RCC_FLAG = 0x97
	RCC_FLAG_SFTRST   RCC_FLAG = 0x98
	RCC_FLAG_IWDG1RST RCC_FLAG = 0x9A
	RCC_FLAG_WWDG1RST RCC_FLAG = 0x9C
	RCC_FLAG_LPWR1RST RCC_FLAG = 0x9E
	RCC_FLAG_LPWR2RST RCC_FLAG = 0x9F
	RCC_FLAG_C1RST             = RCC_FLAG_CPURST
	RCC_FLAG_C2RST    RCC_FLAG = 0x92
	RCC_FLAG_SFTR1ST           = RCC_FLAG_SFTRST
	RCC_FLAG_SFTR2ST  RCC_FLAG = 0x99
	RCC_FLAG_WWDG2RST RCC_FLAG = 0x9D
	RCC_FLAG_IWDG2RST RCC_FLAG = 0x9B
)

func (f RCC_FLAG) Get() bool {
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
		r = RCC.CR.Get()
	case 2:
		r = RCC.BDCR.Get()
	case 3:
		r = RCC.CSR.Get()
	case 4:
		r = RCC.RSR.Get()
	default:
		r = RCC.CIFR.Get()
	}
	return 0 != r&(1<<(f&RCC_FLAG_MASK))
}

type RCC_CLOCKS struct {
	SYSCLK uint32
	CPUCLK uint32
	HCLK   uint32
	PCLK1  uint32
	PCLK2  uint32
	PCLK3  uint32
	PCLK4  uint32
}

type RCC_PLL_CLOCKS struct {
	PLL_P uint32
	PLL_Q uint32
	PLL_R uint32
}

// RCC_PLL_PARAM represents PLL frequency parameters.
type RCC_PLL_PARAM struct {
	M    uint32 // (1-63) division factor for PLL VCO input clock
	N    uint32 // (4-512) Multiplication factor for PLL VCO output clock
	P    uint32 // (2-128, must be even) division factor for system clock
	Q    uint32 // (1-128) division factor for peripheral clocks
	R    uint32 // (1-128) division factor for peripheral clocks
	Frac uint32 // (0-8191) fractional part of multiplication factor for PLL VCO
	In   uint32 // PLL clock input range
	Out  uint32 // PLL clock output range
}

// RCC_PLL_CONFIG defines an RCC PLL configuration.
type RCC_PLL_CONFIG struct {
	State uint32 // PLL state
	Src   uint32 // clock source
	Param RCC_PLL_PARAM
}

// RCC_OSC_CONFIG defines an RCC oscillator (HSE, HSI, CSI, LSE, or LSI)
// configuration.
type RCC_OSC_CONFIG struct {
	OSC     uint32         // oscillator to be configured
	HSE     uint32         // HSE state
	LSE     uint32         // LSE state
	HSI     uint32         // HSI state
	HSITrim uint32         // (RevY:0-63, RevB+:0-127) calibration trim
	LSI     uint32         // LSI state
	HSI48   uint32         // HSI48 state
	CSI     uint32         // CSI state
	CSITrim uint32         // (RevY:0-31, RevB+:0-63) calibration trim
	PLL     RCC_PLL_CONFIG // PLL structure parameters
}

// RCC_CLK_CONFIG defines an RCC SYS/AHB/APB bus clock configuration.
type RCC_CLK_CONFIG struct {
	CLK     uint32 // clock to be configured
	SYSSrc  uint32 // system clock (SYSCLK) source
	SYSDiv  uint32 // system clock (SYSCLK) divider
	AHBDiv  uint32 // AHB clock (HCLK) divider (derived from SYSCLK)
	APB3Div uint32 // APB3 clock (D1PCLK1) divider (derived from HCLK)
	APB1Div uint32 // APB1 clock (PCLK1) divider (derived from HCLK)
	APB2Div uint32 // APB2 clock (PCLK2) divider (derived from HCLK)
	APB4Div uint32 // APB4 clock (D3PCLK1) divider (derived from HCLK)
}

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

var rccC2AllocFlash volatile.Register32

func (c *RCC_CORE_Type) AllocFlash(enable bool) bool {
	switch c {
	case RCC_CORE1:
		// Flash is implicitly allocated to core 1
		return true

	case RCC_CORE2:
		if enable {
			RCC_CORE2.AHB3ENR.SetBits(RCC_AHB3ENR_FLASHEN)
		} else {
			RCC_CORE2.AHB3ENR.ClearBits(RCC_AHB3ENR_FLASHEN)
		}
		// Verify the change was applied
		if RCC_CORE2.AHB3ENR.HasBits(RCC_AHB3ENR_FLASHEN) {
			rccC2AllocFlash.Set(1)
			return enable
		}
		rccC2AllocFlash.Set(0)
		return !enable

	default:
		return false
	}
}

func (rcc *RCC_Type) registerEnable(bus unsafe.Pointer) (*volatile.Register32, uint32) {
	switch bus {
	case unsafe.Pointer(SYSCFG):
		return &rcc.APB4ENR, RCC_APB4ENR_SYSCFGEN
	case unsafe.Pointer(GPIOA):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOAEN
	case unsafe.Pointer(GPIOB):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOBEN
	case unsafe.Pointer(GPIOC):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOCEN
	case unsafe.Pointer(GPIOD):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIODEN
	case unsafe.Pointer(GPIOE):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOEEN
	case unsafe.Pointer(GPIOF):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOFEN
	case unsafe.Pointer(GPIOG):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOGEN
	case unsafe.Pointer(GPIOH):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOHEN
	case unsafe.Pointer(GPIOI):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOIEN
	case unsafe.Pointer(GPIOJ):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOJEN
	case unsafe.Pointer(GPIOK):
		return &rcc.AHB4ENR, RCC_AHB4ENR_GPIOKEN
	}
	return nil, 0
}

func (rcc *RCC_Type) Enable(bus unsafe.Pointer, enable bool) bool {
	if reg, en := rcc.registerEnable(bus); nil != reg {
		if enable {
			reg.SetBits(en)
		} else {
			reg.ClearBits(en)
		}
		return enable == reg.HasBits(en)
	}
	// failed to find an RCC "enable" register for given bus
	return false
}

func (rcc *RCC_Type) IsEnabled(bus unsafe.Pointer) bool {
	if reg, en := rcc.registerEnable(bus); nil != reg {
		return reg.HasBits(en)
	}
	// failed to find an RCC "enable" register for given bus
	return false
}

// EnableGPIO enables the bus clock necessary for peripheral read/write access
// for each GPIO port. If a clock fails to enable, returns false immediately and
// all remaining GPIO clocks are unaffected.
func (rcc *RCC_Type) EnableGPIO() bool {
	for _, g := range []*GPIO_Type{
		GPIOA, GPIOB, GPIOC, GPIOD, GPIOE, GPIOF,
		GPIOG, GPIOH, GPIOI, GPIOJ, GPIOK,
	} {
		if !rcc.Enable(unsafe.Pointer(g), true) {
			return false
		}
	}
	return true
}
