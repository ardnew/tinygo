// +build stm32h7x7_cm4

package runtime

import (
	"device/stm32"
	"runtime/volatile"
)

func resetCore() {}

func initCore() {}

func initCoreFreq() uint32 {

	// M4 core runs at the slower HCLK frequency (240 MHz)
	return stm32.RCC.ClockFreq().HCLK
}
