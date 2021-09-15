package nxp

import (
	"device/arm"
	"errors"
	"math/bits"
	"runtime/interrupt"
	"runtime/volatile"
)

type FlexIO struct {
	*FLEXIO_Type // structure containing all peripheral registers.

	flexIOID FlexIOID

	busClock Clock

	interruptEnabled bool
	interrupt        interrupt.Interrupt
	callback         [NumFlexTimers]func() bool

	dmaChannel [NumFlexShifters]uint8

	usedTimers   uint32
	usedShifters uint32
}

type FlexIOID uint32

// Enumerated constant values for each FlexIO hardware peripheral. Each FlexIO
// instance can be referred to by its corresponding FlexIOID number.
const (
	FIO1 FlexIOID = 1 << iota
	FIO2
	FIO3
)

const (
	NumFlexIO = 3 // number of FlexIO peripherals of iMXRT1062

	NumFlexPins = 14 // Teensy 4.1 = 22, MicroMod = 15

	NumFlexTimers   = 8
	NumFlexShifters = 8
)

var (
	FlexIO1 = FlexIO{
		FLEXIO_Type: FLEXIO1,
		flexIOID:    FIO1,
		dmaChannel: [NumFlexShifters]uint8{
			dmaSourceFlexIO1Req0, dmaSourceFlexIO1Req1,
			dmaSourceFlexIO1Req2, dmaSourceFlexIO1Req3,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	}
	FlexIO2 = FlexIO{
		FLEXIO_Type: FLEXIO2,
		flexIOID:    FIO2,
		dmaChannel: [NumFlexShifters]uint8{
			dmaSourceFlexIO2Req0, dmaSourceFlexIO2Req1,
			dmaSourceFlexIO2Req2, dmaSourceFlexIO2Req3,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	}
	FlexIO3 = FlexIO{
		FLEXIO_Type: FLEXIO3,
		flexIOID:    FIO3,
		dmaChannel: [NumFlexShifters]uint8{
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	}
)

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

func init() {
	FlexIO1.interrupt = interrupt.New(IRQ_FLEXIO1, FlexIO1.handleInterrupt)
	FlexIO2.interrupt = interrupt.New(IRQ_FLEXIO2, FlexIO2.handleInterrupt)
	FlexIO3.interrupt = interrupt.New(IRQ_FLEXIO3, FlexIO3.handleInterrupt)
}

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
			if !f.interruptEnabled {
				f.interrupt.Enable()
				f.interruptEnabled = true
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

type FlexIOShifterConfig struct {
	Width    uint32
	Input    uint32
	StopBit  uint32
	StartBit uint32
}

func (c FlexIOShifterConfig) Uint32() uint32 {
	return ((c.Width << FLEXIO_SHIFTCFG_PWIDTH_Pos) & FLEXIO_SHIFTCFG_PWIDTH_Msk) |
		((c.Input << FLEXIO_SHIFTCFG_INSRC_Pos) & FLEXIO_SHIFTCFG_INSRC_Msk) |
		((c.StopBit << FLEXIO_SHIFTCFG_SSTOP_Pos) & FLEXIO_SHIFTCFG_SSTOP_Msk) |
		((c.StartBit << FLEXIO_SHIFTCFG_SSTART_Pos) & FLEXIO_SHIFTCFG_SSTART_Msk)
}

type FlexIOTimerConfig struct {
	Output    uint32
	Decrement uint32
	Reset     uint32
	Disable   uint32
	Enable    uint32
	Stop      uint32
	Start     uint32
}

func (c FlexIOTimerConfig) Uint32() uint32 {
	return ((c.Output << FLEXIO_TIMCFG_TIMOUT_Pos) & FLEXIO_TIMCFG_TIMOUT_Msk) |
		((c.Decrement << FLEXIO_TIMCFG_TIMDEC_Pos) & FLEXIO_TIMCFG_TIMDEC_Msk) |
		((c.Reset << FLEXIO_TIMCFG_TIMRST_Pos) & FLEXIO_TIMCFG_TIMRST_Msk) |
		((c.Disable << FLEXIO_TIMCFG_TIMDIS_Pos) & FLEXIO_TIMCFG_TIMDIS_Msk) |
		((c.Enable << FLEXIO_TIMCFG_TIMENA_Pos) & FLEXIO_TIMCFG_TIMENA_Msk) |
		((c.Stop << FLEXIO_TIMCFG_TSTOP_Pos) & FLEXIO_TIMCFG_TSTOP_Msk) |
		((c.Start << FLEXIO_TIMCFG_TSTART_Pos) & FLEXIO_TIMCFG_TSTART_Msk)
}

type FlexIOConfig interface {
	Uint32() uint32
}

func (f *FlexIO) Configure(reg *volatile.Register32, config FlexIOConfig) error {
	reg.Set(config.Uint32())
	return nil
}

type ShifterControl struct {
	TimSel uint32
	TimPol uint32
	PinCfg uint32
	PinSel uint32
	PinPol uint32
	Mode   uint32
}

func (c FlexIOShifterControl) Uint32() uint32 {
	return ((c.TimSel << FLEXIO_SHIFTCTL_TIMSEL_Pos) & FLEXIO_SHIFTCTL_TIMSEL_Msk) |
		((c.TimPol << FLEXIO_SHIFTCTL_TIMPOL_Pos) & FLEXIO_SHIFTCTL_TIMPOL_Msk) |
		((c.PinCfg << FLEXIO_SHIFTCTL_PINCFG_Pos) & FLEXIO_SHIFTCTL_PINCFG_Msk) |
		((c.PinSel << FLEXIO_SHIFTCTL_PINSEL_Pos) & FLEXIO_SHIFTCTL_PINSEL_Msk) |
		((c.PinPol << FLEXIO_SHIFTCTL_PINPOL_Pos) & FLEXIO_SHIFTCTL_PINPOL_Msk) |
		((c.Mode << FLEXIO_SHIFTCTL_SMOD_Pos) & FLEXIO_SHIFTCTL_SMOD_Msk)
}

type FlexIOTimerControl struct {
	TrgSel uint32
	TrgPol uint32
	TrgSrc uint32
	PinCfg uint32
	PinSel uint32
	PinPol uint32
	Mode   uint32
}

func (c FlexIOTimerControl) Uint32() uint32 {
	return ((c.TrgSel << FLEXIO_TIMCTL_TRGSEL_Pos) & FLEXIO_TIMCTL_TRGSEL_Msk) |
		((c.TrgPol << FLEXIO_TIMCTL_TRGPOL_Pos) & FLEXIO_TIMCTL_TRGPOL_Msk) |
		((c.TrgSrc << FLEXIO_TIMCTL_TRGSRC_Pos) & FLEXIO_TIMCTL_TRGSRC_Msk) |
		((c.PinCfg << FLEXIO_TIMCTL_PINCFG_Pos) & FLEXIO_TIMCTL_PINCFG_Msk) |
		((c.PinSel << FLEXIO_TIMCTL_PINSEL_Pos) & FLEXIO_TIMCTL_PINSEL_Msk) |
		((c.PinPol << FLEXIO_TIMCTL_PINPOL_Pos) & FLEXIO_TIMCTL_PINPOL_Msk) |
		((c.Mode << FLEXIO_TIMCTL_TIMOD_Pos) & FLEXIO_TIMCTL_TIMOD_Msk)
}

type FlexIOControl interface {
	Uint32() uint32
}

func (f *FlexIO) Control(reg *volatile.Register32, control FlexIOControl) error {
	reg.Set(control.Uint32())
	return nil
}
