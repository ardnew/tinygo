// +build stm32h7

package runtime

import "device/arm"

const asyncScheduler = false

type timeUnit int64

//go:extern _svectors
var _svectors [0]uint8

//go:extern _evectors
var _evectors [0]uint8

//go:extern _svtor
var _svtor [0]uint8 // vectors location in RAM

func postinit() {}

//export Reset_Handler
func main() {

	// Disable interrupts
	irq := arm.DisableInterrupts()

	// Reset shared bus clocks and initialize RAM, VTOR, and HSEM (for IPC)
	initSystem()

	// reenable interrupts
	arm.EnableInterrupts(irq)

	// configure core and peripheral clocks/PLLs/PFDs
	initClocks()

	run()
	abort()
}

func waitForEvents() {
	arm.Asm("wfe")
}

func putchar(c byte) {}
