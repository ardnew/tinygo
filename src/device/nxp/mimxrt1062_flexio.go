package nxp

import (
	"device/arm"
	"errors"
	"math/bits"
	"runtime/interrupt"
)

type FlexIO struct {
	*FLEXIO_Type // structure containing all peripheral registers.

	busClock Clock

	interrupt *interrupt.Interrupt
	callback  [NumFlexTimers]func() bool

	dmaChannel [NumFlexShifters]uint8

	usedTimers uint32
}

const (
	NumFlexIO = 3 // number of FlexIO peripherals of iMXRT1062

	NumFlexPins = 14 // Teensy 4.1 = 22, MicroMod = 15

	NumFlexShifters = 8
	NumFlexTimers   = 8
)

var FlexIO = [NumFlexIO]FlexIO{
	{
		FLEXIO_Type: FLEXIO1,
		dmaChannel: [NumFlexShifters]uint8{
			dmaSourceFlexIO1Req0, dmaSourceFlexIO1Req1,
			dmaSourceFlexIO1Req2, dmaSourceFlexIO1Req3,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	},
	{
		FLEXIO_Type: FLEXIO2,
		dmaChannel: [NumFlexShifters]uint8{
			dmaSourceFlexIO2Req0, dmaSourceFlexIO2Req1,
			dmaSourceFlexIO2Req2, dmaSourceFlexIO2Req3,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	},
	{
		FLEXIO_Type: FLEXIO3,
		dmaChannel: [NumFlexShifters]uint8{
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	},
}

var (
	ErrFlexIOCallbackFull = errors.New("no more callbacks can be added to FLEXIO interrupt handler")
	ErrFlexIOTimersFull   = errors.New("insufficient FLEXIO timers available")
)

const (
	dmaSourceNONE = 0xFF

	dmaSourceFlexIO1Req0 = 0 | 0x00 // FlexIO1 Request0
	dmaSourceFlexIO1Req1 = 0 | 0x00 // FlexIO1 Request1
	dmaSourceFlexIO1Req2 = 0 | 0x40 // FlexIO1 Request2
	dmaSourceFlexIO1Req3 = 0 | 0x40 // FlexIO1 Request3

	dmaSourceFlexIO2Req0 = 1 | 0x00 // FlexIO2 Request0
	dmaSourceFlexIO2Req1 = 1 | 0x00 // FlexIO2 Request1
	dmaSourceFlexIO2Req2 = 1 | 0x40 // FlexIO2 Request2
	dmaSourceFlexIO2Req3 = 1 | 0x40 // FlexIO2 Request3
)

func (f *FlexIO) handleInterrupt(interrupt.Interrupt) {
	for _, h := range f.callback {
		if h != nil && h() {
			return
		}
	}
	// no callbacks installed
	arm.AsmFull(`
		dsb 0xF
	`, nil)
}

func (f *FlexIO) addHandler(h func()) error {
	for i, c := range f.callback {
		if c == nil {
			f.callback[i] = h
			// install and enable interrupt handler if it hasn't yet been created.
			if f.interrupt == nil {
				var irq interrupt.Interrupt
				switch f {
				case &FlexIO1:
					irq = interrupt.New(IRQ_FLEXIO1, f.handleInterrupt)
				case &FlexIO2:
					irq = interrupt.New(IRQ_FLEXIO2, f.handleInterrupt)
				case &FlexIO3:
					irq = interrupt.New(IRQ_FLEXIO3, f.handleInterrupt)
				}
				f.interrupt = &irq
				f.interrupt.Enable()
			}
			return nil
		}
	}
	return ErrFlexIOCallbackFull
}

func (f *FlexIO) requestTimers(n int) (uint32, error) {
	var timers uint32
	for n > 0 {
		if uint32(n)+bits.OnesCount32(f.usedTimers) > NumFlexTimers {
			return ErrFlexIOTimersFull
		}
		for i := uint32(0); i < NumFlexTimers; i++ {
			m := uint32(1) << i
			if (f.usedTimers & m) == 0 {
				timers |= m
				n--
				break
			}
		}
	}
	f.usedTimers |= timers
	return timers, nil
}
