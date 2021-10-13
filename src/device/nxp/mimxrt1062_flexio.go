// Hand created file. DO NOT DELETE.
// Type definitions, fields, and constants associated with FlexIO peripherals of
// the NXP MIMXRT1062.

//go:build nxp && mimxrt1062

package nxp

import (
	"device/arm"
	"errors"
	"math/bits"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

var (
	ErrFlexIOLimitCallbacks  = errors.New("insufficient FlexIO callbacks available")
	ErrFlexIOLimitTimers     = errors.New("insufficient FlexIO timers available")
	ErrFlexIOLimitShifters   = errors.New("insufficient FlexIO shifters available")
	ErrFlexIOTimerIndex      = errors.New("invalid FlexIO timer index")
	ErrFlexIOShifterIndex    = errors.New("invalid FlexIO shifter index")
	ErrFlexIOInvalidRegister = errors.New("invalid FlexIO register")
	ErrFlexIOInvalidConfig   = errors.New("invalid FlexIO configuration register")
	ErrFlexIOInvalidControl  = errors.New("invalid FlexIO control register")
)

func init() {
	FlexIO1.interrupt = interrupt.New(IRQ_FLEXIO1, FlexIO1.handleInterrupt)
	FlexIO2.interrupt = interrupt.New(IRQ_FLEXIO2, FlexIO2.handleInterrupt)
	FlexIO3.interrupt = interrupt.New(IRQ_FLEXIO3, FlexIO3.handleInterrupt)
}

type FlexIO struct {
	*FLEXIO_Type // structure containing all peripheral registers.

	id FlexIOID

	interruptEnabled bool
	interrupt        interrupt.Interrupt
	callback         [NumFlexIOTimers]func() bool

	dmaChannel [NumFlexIOShifters]DMAChannelID

	usedTimers   uint32
	usedShifters uint32
}

const (
	NumFlexIO = 3 // number of FlexIO peripherals of iMXRT1062

	NumFlexIOTimers   = 8
	NumFlexIOShifters = 8

	numFlexIOAPITimers   = 4
	numFlexIOAPIShifters = 4
)

type FlexIOID uint32

// Enumerated constant values for each FlexIO hardware peripheral. Each FlexIO
// instance can be referred to by its corresponding FlexIOID number.
const (
	FIO1 FlexIOID = 1 << iota
	FIO2
	FIO3
)

func (id FlexIOID) FlexIO() *FlexIO {
	switch id {
	case FIO1:
		return &FlexIO1
	case FIO2:
		return &FlexIO2
	case FIO3:
		return &FlexIO3
	}
	return nil
}

var (
	FlexIO1 = FlexIO{
		FLEXIO_Type: FLEXIO1,
		id:          FIO1,
		dmaChannel: [NumFlexIOShifters]DMAChannelID{
			dmaSourceFlexIO1Req0, dmaSourceFlexIO1Req1,
			dmaSourceFlexIO1Req2, dmaSourceFlexIO1Req3,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	}
	FlexIO2 = FlexIO{
		FLEXIO_Type: FLEXIO2,
		id:          FIO2,
		dmaChannel: [NumFlexIOShifters]DMAChannelID{
			dmaSourceFlexIO2Req0, dmaSourceFlexIO2Req1,
			dmaSourceFlexIO2Req2, dmaSourceFlexIO2Req3,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	}
	FlexIO3 = FlexIO{
		FLEXIO_Type: FLEXIO3,
		id:          FIO3,
		dmaChannel: [NumFlexIOShifters]DMAChannelID{
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
			dmaSourceNONE, dmaSourceNONE, dmaSourceNONE, dmaSourceNONE,
		},
	}
)

func (f *FlexIO) ID() FlexIOID { return f.id }

func (f *FlexIO) DMAChannel(shifter uint8) DMAChannelID {
	if shifter < NumFlexIOShifters {
		return f.dmaChannel[shifter]
	}
	return dmaSourceNONE
}

type FlexIOPin struct {
	Bus *FlexIO
	Pin uint8
	Mux uint8
}

func (f *FlexIO) handleInterrupt(interrupt.Interrupt) {
	for _, h := range f.callback {
		if h != nil && h() {
			break
		}
	}
	// The dsb instruction is copied from NXP MCUXpresso SDK (iMXRT1062), which
	// cites the following ARM errata as motivation:
	//
	//   | ARM errata 838869, affects Cortex-M4(F) Store immediate overlapping
	//   | exception return operation might vector to incorrect interrupt.
	//   | For Cortex-M7, if core speed much faster than peripheral register write
	//   | speed, the peripheral interrupt flags may be still set after exiting
	//   | ISR, this results to the same error similar with errata 83869.
	//
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
	if n > 0 && n+bits.OnesCount32(f.usedTimers) > NumFlexIOTimers {
		return 0, ErrFlexIOLimitTimers
	}
	var timers uint32
	for n > 0 {
		for i := uint32(0); i < NumFlexIOTimers; i++ {
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

func (f *FlexIO) requestShifter(excludeChannel ...DMAChannelID) (uint32, error) {
	if bits.OnesCount32(f.usedShifters) >= NumFlexIOShifters {
		return 0, ErrFlexIOLimitShifters
	}
	var shifter uint32
	for i := uint32(0); i < NumFlexIOShifters; i++ {
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

func (f *FlexIO) SetConfig(reg *volatile.Register32, config FlexIOConfig) error {
	if reg == nil {
		return ErrFlexIOInvalidRegister
	}
	reg.Set(config.Uint32())
	return nil
}

func (f *FlexIO) SetTimerConfig(timer int, config FlexIOConfig) error {
	if timer < 0 || timer >= NumFlexIOTimers {
		return ErrFlexIOTimerIndex
	}
	if timer < numFlexIOAPITimers {
		return f.SetConfig(&f.TIMCFG[timer], config)
	}
	// There are more timers (8) than defined in both the reference manual (4)
	// and the SVD API (4). So the "devie/nxp" package TIMCFG arrays have only 4
	// registers. We have to manually get a pointer to any memory-mapped registers
	// at indices 4 - 7 using the "unsafe" package.
	off := 4 * uintptr(timer)
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&f.TIMCFG[0])) + off)
	return f.SetConfig((*volatile.Register32)(ptr), config)
}

func (f *FlexIO) SetShifterConfig(shifter int, config FlexIOConfig) error {
	if shifter < 0 || shifter >= NumFlexIOShifters {
		return ErrFlexIOShifterIndex
	}
	if shifter < numFlexIOAPIShifters {
		return f.SetConfig(&f.SHIFTCFG[shifter], config)
	}
	// There are more shifters (8) than defined in both the reference manual (4)
	// and the SVD API (4). So the "devie/nxp" package SHIFTCFG arrays have only 4
	// registers. We have to manually get a pointer to any memory-mapped registers
	// at indices 4 - 7 using the "unsafe" package.
	off := 4 * uintptr(shifter)
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&f.SHIFTCFG[0])) + off)
	return f.SetConfig((*volatile.Register32)(ptr), config)
}

type FlexIOShifterControl struct {
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

func (f *FlexIO) SetControl(reg *volatile.Register32, control FlexIOControl) error {
	if reg == nil {
		return ErrFlexIOInvalidRegister
	}
	reg.Set(control.Uint32())
	return nil
}

func (f *FlexIO) SetTimerControl(timer int, control FlexIOControl) error {
	if timer < 0 || timer >= NumFlexIOTimers {
		return ErrFlexIOTimerIndex
	}
	if timer < numFlexIOAPITimers {
		return f.SetControl(&f.TIMCTL[timer], control)
	}
	// There are more timers (8) than defined in both the reference manual (4)
	// and the SVD API (4). So the "devie/nxp" package TIMCTL arrays have only 4
	// registers. We have to manually get a pointer to any memory-mapped registers
	// at indices 4 - 7 using the "unsafe" package.
	off := 4 * uintptr(timer)
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&f.TIMCTL[0])) + off)
	return f.SetControl((*volatile.Register32)(ptr), control)
}

func (f *FlexIO) SetShifterControl(shifter int, control FlexIOControl) error {
	if shifter < 0 || shifter >= NumFlexIOShifters {
		return ErrFlexIOShifterIndex
	}
	if shifter < numFlexIOAPIShifters {
		return f.SetControl(&f.SHIFTCTL[shifter], control)
	}
	// There are more shifters (8) than defined in both the reference manual (4)
	// and the SVD API (4). So the "devie/nxp" package SHIFTCFG arrays have only 4
	// registers. We have to manually get a pointer to any memory-mapped registers
	// at indices 4 - 7 using the "unsafe" package.
	off := 4 * uintptr(shifter)
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&f.SHIFTCTL[0])) + off)
	return f.SetControl((*volatile.Register32)(ptr), control)
}
