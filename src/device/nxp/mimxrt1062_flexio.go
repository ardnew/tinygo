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

	usedTimers   uint32
	usedShifters uint32
}

const (
	NumFlexIO = 3 // number of FlexIO peripherals of iMXRT1062

	NumFlexPins = 14 // Teensy 4.1 = 22, MicroMod = 15

	NumFlexTimers   = 8
	NumFlexShifters = 8
)

var flexIO = [NumFlexIO]FlexIO{
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
	ErrFlexIOLimitCallbacks = errors.New("insufficient FLEXIO callbacks available")
	ErrFlexIOLimitTimers    = errors.New("insufficient FLEXIO timers available")
	ErrFlexIOLimitShifters  = errors.New("insufficient FLEXIO shifters available")
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

func (f *FlexIO) addHandler(h func() bool) error {
	for i, c := range f.callback {
		if c == nil {
			f.callback[i] = h
			// install and enable interrupt handler if it hasn't yet been created.
			if f.interrupt == nil {
				var irq interrupt.Interrupt
				switch f {
				case &flexIO[0]:
					irq = interrupt.New(IRQ_FLEXIO1, flexIO[0].handleInterrupt)
				case &flexIO[1]:
					irq = interrupt.New(IRQ_FLEXIO2, flexIO[1].handleInterrupt)
				case &flexIO[2]:
					irq = interrupt.New(IRQ_FLEXIO3, flexIO[2].handleInterrupt)
				}
				f.interrupt = &irq
				f.interrupt.Enable()
			}
			return nil
		}
	}
	return ErrFlexIOLimitCallbacks
}

func (f *FlexIO) requestTimers(n int) (uint32, error) {
	if n > 0 && n+bits.OnesCount32(f.usedTimers) > NumFlexTimers {
		return 0, ErrFlexIOLimitTimers
	}
	var timers uint32
	for n > 0 {
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

func (f *FlexIO) requestShifter(excludeChannel ...uint8) (uint32, error) {
	if bits.OnesCount32(f.usedShifters) >= NumFlexShifters {
		return 0, ErrFlexIOLimitShifters
	}
	var shifter uint32
	for i := uint32(0); i < NumFlexShifters; i++ {
		m := uint32(1) << i
		if (f.usedShifters & m) == 0 {
			exclude := false
			for _, x := range excludeChannel {
				if f.dmaChannel[i] == x {
					exclude = true
					break
				}
			}
			if !exclude {
				shifter |= m
				break
			}
		}
	}
	f.usedShifters |= shifter
	return shifter, nil
}
