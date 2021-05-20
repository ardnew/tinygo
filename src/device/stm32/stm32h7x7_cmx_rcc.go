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

	// The auto-generated TinyGo source contains bitmasks for the wrong table
	// (RCC_AHB3ENR), and doesn't have those required for RCC_D3CFGR (which has
	// been recreated from STM32 HAL SDK below). Need to determine if this is a
	// bug with the SVD itself or the parser/generator.

	RCC_D3CFGR_D3PPRE_Pos = 4                            //
	RCC_D3CFGR_D3PPRE_Msk = 0x7 << RCC_D3CFGR_D3PPRE_Pos // 0x00000070
	RCC_D3CFGR_D3PPRE     = RCC_D3CFGR_D3PPRE_Msk        // D3PPRE1[2:0] bits (APB4 prescaler)

	RCC_AHB3ENR_FLASHEN_Pos = 8
	RCC_AHB3ENR_FLASHEN_Msk = 0x1 << RCC_AHB3ENR_FLASHEN_Pos // 0x00000100
	RCC_AHB3ENR_FLASHEN     = RCC_AHB3ENR_FLASHEN_Msk

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

	RCC_CFGR_SW_HSI = 0 << RCC_CFGR_SW_Pos
	RCC_CFGR_SW_CSI = 1 << RCC_CFGR_SW_Pos
	RCC_CFGR_SW_HSE = 2 << RCC_CFGR_SW_Pos
	RCC_CFGR_SW_PLL = 3 << RCC_CFGR_SW_Pos

	RCC_CFGR_SWS_HSI = 0 << RCC_CFGR_SWS_Pos
	RCC_CFGR_SWS_CSI = 1 << RCC_CFGR_SWS_Pos
	RCC_CFGR_SWS_HSE = 2 << RCC_CFGR_SWS_Pos
	RCC_CFGR_SWS_PLL = 3 << RCC_CFGR_SWS_Pos

	RCC_PLLCKSELR_PLLSRC_HSI  = 0 << RCC_PLLCKSELR_PLLSRC_Pos
	RCC_PLLCKSELR_PLLSRC_CSI  = 1 << RCC_PLLCKSELR_PLLSRC_Pos
	RCC_PLLCKSELR_PLLSRC_HSE  = 2 << RCC_PLLCKSELR_PLLSRC_Pos
	RCC_PLLCKSELR_PLLSRC_NONE = 3 << RCC_PLLCKSELR_PLLSRC_Pos

	RCC_PLLCFGR_PLL1RGE_0 = 0 << RCC_PLLCFGR_PLL1RGE_Pos // 0x00000000: Clock range frequency between 1 and 2 MHz
	RCC_PLLCFGR_PLL1RGE_1 = 1 << RCC_PLLCFGR_PLL1RGE_Pos // 0x00000004: Clock range frequency between 2 and 4 MHz
	RCC_PLLCFGR_PLL1RGE_2 = 2 << RCC_PLLCFGR_PLL1RGE_Pos // 0x00000008: Clock range frequency between 4 and 8 MHz
	RCC_PLLCFGR_PLL1RGE_3 = 3 << RCC_PLLCFGR_PLL1RGE_Pos // 0x0000000C: Clock range frequency between 8 and 16 MHz

	RCC_PLLCFGR_PLL1VCOSEL_WIDE   = 0 << RCC_PLLCFGR_PLL1VCOSEL_Pos
	RCC_PLLCFGR_PLL1VCOSEL_MEDIUM = 1 << RCC_PLLCFGR_PLL1VCOSEL_Pos
)

// Default oscillator frequencies
const (
	RCC_HSE_FREQ_HZ uint32 = 25000000
	RCC_CSI_FREQ_HZ uint32 = 4000000
	RCC_HSI_FREQ_HZ uint32 = 64000000
)

var RCC_FREQ_PRESCALAR = [...]uint8{
	0, 0, 0, 0, 1, 2, 3, 4, 1, 2, 3, 4, 6, 7, 8, 9,
}

type RCC_CLOCKS_Type struct {
	SYSCLK uint32
	CPUCLK uint32
	HCLK   uint32
	PCLK1  uint32
	PCLK2  uint32
	PCLK3  uint32
	PCLK4  uint32
}

type RCC_PLL_CLOCKS_Type struct {
	P uint32
	Q uint32
	R uint32
}

// ClockFreq returns the current frequency of all internal CPU clocks.
func (rcc *RCC_Type) ClockFreq() (clk RCC_CLOCKS_Type) {
	clk.SYSCLK = rcc.sysclkFreq()
	clk.HCLK = clk.SYSCLK >> (RCC_FREQ_PRESCALAR[(((rcc.D1CFGR.Get()&
		RCC_D1CFGR_HPRE_Msk)>>RCC_D1CFGR_HPRE_Pos)&RCC_D1CFGR_HPRE_Msk)>>
		RCC_D1CFGR_HPRE_Pos] & 0x1F)
	clk.PCLK1 = clk.HCLK >> (RCC_FREQ_PRESCALAR[(((rcc.D2CFGR.Get()&
		RCC_D2CFGR_D2PPRE1_Msk)>>RCC_D2CFGR_D2PPRE1_Pos)&RCC_D2CFGR_D2PPRE1_Msk)>>
		RCC_D2CFGR_D2PPRE1_Pos] & 0x1F)
	clk.PCLK2 = clk.HCLK >> (RCC_FREQ_PRESCALAR[(((rcc.D2CFGR.Get()&
		RCC_D2CFGR_D2PPRE2_Msk)>>RCC_D2CFGR_D2PPRE2_Pos)&RCC_D2CFGR_D2PPRE2_Msk)>>
		RCC_D2CFGR_D2PPRE2_Pos] & 0x1F)
	clk.PCLK3 = clk.HCLK >> (RCC_FREQ_PRESCALAR[(((rcc.D1CFGR.Get()&
		RCC_D1CFGR_D1PPRE_Msk)>>RCC_D1CFGR_D1PPRE_Pos)&RCC_D1CFGR_D1PPRE_Msk)>>
		RCC_D1CFGR_D1PPRE_Pos] & 0x1F)
	clk.PCLK4 = clk.HCLK >> (RCC_FREQ_PRESCALAR[(((rcc.D3CFGR.Get()&
		RCC_D3CFGR_D3PPRE_Msk)>>RCC_D3CFGR_D3PPRE_Pos)&RCC_D3CFGR_D3PPRE_Msk)>>
		RCC_D3CFGR_D3PPRE_Pos] & 0x1F)
	return
}

func (rcc *RCC_Type) sysclkFreq() uint32 {
	switch rcc.CFGR.Get() & RCC_CFGR_SWS_Msk {
	case RCC_CFGR_SWS_HSI:
		div := (rcc.CR.Get() & RCC_CR_HSIDIV_Msk) >> RCC_CR_HSIDIV_Pos
		return RCC_HSI_FREQ_HZ >> div
	case RCC_CFGR_SWS_CSI:
		return RCC_CSI_FREQ_HZ
	case RCC_CFGR_SWS_HSE:
		return RCC_HSE_FREQ_HZ
	case RCC_CFGR_SWS_PLL:
		clk := rcc.pll1Freq()
		return clk.P
	}
	return 0
}

func (rcc *RCC_Type) pll1Freq() (clk RCC_PLL_CLOCKS_Type) {

	var in uint32
	switch rcc.PLLCKSELR.Get() & RCC_PLLCKSELR_PLLSRC_Msk {
	case RCC_PLLCKSELR_PLLSRC_HSI:
		if rcc.CR.HasBits(RCC_CR_HSIRDY) {
			div := (rcc.CR.Get() & RCC_CR_HSIDIV_Msk) >> RCC_CR_HSIDIV_Pos
			in = RCC_HSI_FREQ_HZ >> div
		}
	case RCC_PLLCKSELR_PLLSRC_CSI:
		if rcc.CR.HasBits(RCC_CR_CSIRDY) {
			in = RCC_CSI_FREQ_HZ
		}
	case RCC_PLLCKSELR_PLLSRC_HSE:
		if rcc.CR.HasBits(RCC_CR_HSERDY) {
			in = RCC_HSE_FREQ_HZ
		}
	case RCC_PLLCKSELR_PLLSRC_NONE:
		// PLL disabled
	}

	clk.P = 0
	clk.Q = 0
	clk.R = 0

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
			clk.P = calcPLLFreq(in, m, n, fracn, p)
		}
		if rcc.PLLCFGR.HasBits(RCC_PLLCFGR_DIVQ1EN) {
			q := ((rcc.PLL1DIVR.Get() & RCC_PLL1DIVR_DIVQ1_Msk) >>
				RCC_PLL1DIVR_DIVQ1_Pos) + 1
			clk.Q = calcPLLFreq(in, m, n, fracn, q)
		}
		if rcc.PLLCFGR.HasBits(RCC_PLLCFGR_DIVR1EN) {
			r := ((rcc.PLL1DIVR.Get() & RCC_PLL1DIVR_DIVR1_Msk) >>
				RCC_PLL1DIVR_DIVR1_Pos) + 1
			clk.R = calcPLLFreq(in, m, n, fracn, r)
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
