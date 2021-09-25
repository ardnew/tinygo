// Hand created file. DO NOT DELETE.
// Type definitions, fields, and constants associated with eDMA peripherals of
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
	ErrDMALimitChannels = errors.New("insufficient DMA channels available")
)

// DMAChannel represents an individual DMA channel.
type DMAChannel struct {
	*DMA_Type

	TCD *TCD_Type

	id DMAChannelID
}

type DMAChannelID uint8

func (id DMAChannelID) DMAChannel() *DMAChannel {
	return nil
}

func (d *DMAChannel) ID() DMAChannelID { return d.id }

const (
	NumDMAChannels   = 32
	NumDMAInterrupts = NumDMAChannels / 2
)

var (
	_DMAInstance  = DMA0
	_DMAChannel   [NumDMAChannels]DMAChannel
	_DMAInterrupt [NumDMAInterrupts]interrupt.Interrupt

	chanMask uint32 // DMA channels allocated (channel ID bitmask)

	irqMask uint16 // DMA interrupts installed (IRQ bitmask)
	irqChan uint32 // DMA channels connected to an interrupt (channel ID bitmask)
)

func AllocDMA() (*DMAChannel, error) {

	// Find a free DMA channel
	EnableDMAInterrupts(false)
	c := DMAChannelID(0)
	for {
		if !c.allocated() {
			chanMask |= 1 << c
			break
		}
		c++
		if c >= NumDMAChannels {
			EnableDMAInterrupts(true)
			return nil, ErrDMALimitChannels
		}
	}
	EnableDMAInterrupts(true)

	d := &_DMAChannel[c]

	d.DMA_Type = _DMAInstance

	d.id = c
	d.id.initInterrupt()

	d.CR.Set(DMA_CR_GRP1PRI | DMA_CR_EMLM | DMA_CR_EDBG)
	d.CERQ.Set(uint8(d.id))
	d.CERR.Set(uint8(d.id))
	d.CEEI.Set(uint8(d.id))
	d.CINT.Set(uint8(d.id))

	d.TCD = d.id.tcd()
	d.TCD.reset()

	return d, nil
}

func (d *DMAChannel) Enable(enable bool) {
	if enable {
		d.SERQ.Set(uint8(d.id))
	} else {
		d.CERQ.Set(uint8(d.id))
	}
}

func (d *DMAChannel) SetHardwareTrigger(source uint8) {
	chcfg := (uint32(source) << DMAMUX_CHCFG_SOURCE_Pos) & DMAMUX_CHCFG_SOURCE_Msk
	chcfg |= DMAMUX_CHCFG_ENBL
	DMAMUX.CHCFG[d.id].Set(0)
	DMAMUX.CHCFG[d.id].Set(chcfg)
}

func (d *DMAChannel) handleInterrupt() {
}

const (
	dmaSourceNONE = 0xFF

	dmaSourceFlexIO1Req0 = 0 | 0x00 // (0)  FlexIO1 Request 0
	dmaSourceFlexIO1Req1 = 0 | 0x00 // (0)  FlexIO1 Request 1
	dmaSourceFlexIO1Req2 = 0 | 0x40 // (64) FlexIO1 Request 2
	dmaSourceFlexIO1Req3 = 0 | 0x40 // (64) FlexIO1 Request 3

	dmaSourceFlexIO2Req0 = 1 | 0x00 // (1)  FlexIO2 Request 0
	dmaSourceFlexIO2Req1 = 1 | 0x00 // (1)  FlexIO2 Request 1
	dmaSourceFlexIO2Req2 = 1 | 0x40 // (65) FlexIO2 Request 2
	dmaSourceFlexIO2Req3 = 1 | 0x40 // (65) FlexIO2 Request 3

	dmaSourceFlexPWM1Read0  = 0x20 | 0 | 0 // (32)  FlexPWM1[0] Read
	dmaSourceFlexPWM1Read1  = 0x20 | 0 | 1 // (33)  FlexPWM1[1] Read
	dmaSourceFlexPWM1Read2  = 0x20 | 0 | 2 // (34)  FlexPWM1[2] Read
	dmaSourceFlexPWM1Read3  = 0x20 | 0 | 3 // (35)  FlexPWM1[3] Read
	dmaSourceFlexPWM1Write0 = 0x20 | 4 | 0 // (36)  FlexPWM1[0] Write
	dmaSourceFlexPWM1Write1 = 0x20 | 4 | 1 // (37)  FlexPWM1[1] Write
	dmaSourceFlexPWM1Write2 = 0x20 | 4 | 2 // (38)  FlexPWM1[2] Write
	dmaSourceFlexPWM1Write3 = 0x20 | 4 | 3 // (39)  FlexPWM1[3] Write

	dmaSourceFlexPWM2Read0  = 0x60 | 0 | 0 //	(96)  FlexPWM2[0] Read
	dmaSourceFlexPWM2Read1  = 0x60 | 0 | 1 //	(97)  FlexPWM2[1] Read
	dmaSourceFlexPWM2Read2  = 0x60 | 0 | 2 //	(98)  FlexPWM2[2] Read
	dmaSourceFlexPWM2Read3  = 0x60 | 0 | 3 //	(99)  FlexPWM2[3] Read
	dmaSourceFlexPWM2Write0 = 0x60 | 4 | 0 //	(100) FlexPWM2[0] Write
	dmaSourceFlexPWM2Write1 = 0x60 | 4 | 1 //	(101) FlexPWM2[1] Write
	dmaSourceFlexPWM2Write2 = 0x60 | 4 | 2 //	(102) FlexPWM2[2] Write
	dmaSourceFlexPWM2Write3 = 0x60 | 4 | 3 //	(103) FlexPWM2[3] Write

	dmaSourceFlexPWM3Read0  = 0x28 | 0 | 0 //	(40)  FlexPWM3[0] Read
	dmaSourceFlexPWM3Read1  = 0x28 | 0 | 1 //	(41)  FlexPWM3[1] Read
	dmaSourceFlexPWM3Read2  = 0x28 | 0 | 2 //	(42)  FlexPWM3[2] Read
	dmaSourceFlexPWM3Read3  = 0x28 | 0 | 3 //	(43)  FlexPWM3[3] Read
	dmaSourceFlexPWM3Write0 = 0x28 | 4 | 0 //	(44)  FlexPWM3[0] Write
	dmaSourceFlexPWM3Write1 = 0x28 | 4 | 1 //	(45)  FlexPWM3[1] Write
	dmaSourceFlexPWM3Write2 = 0x28 | 4 | 2 //	(46)  FlexPWM3[2] Write
	dmaSourceFlexPWM3Write3 = 0x28 | 4 | 3 //	(47)  FlexPWM3[3] Write

	dmaSourceFlexPWM4Read0  = 0x68 | 0 | 0 // (104) FlexPWM4[0] Read
	dmaSourceFlexPWM4Read1  = 0x68 | 0 | 1 // (105) FlexPWM4[1] Read
	dmaSourceFlexPWM4Read2  = 0x68 | 0 | 2 // (106) FlexPWM4[2] Read
	dmaSourceFlexPWM4Read3  = 0x68 | 0 | 3 // (107) FlexPWM4[3] Read
	dmaSourceFlexPWM4Write0 = 0x68 | 4 | 0 // (108) FlexPWM4[0] Write
	dmaSourceFlexPWM4Write1 = 0x68 | 4 | 1 // (109) FlexPWM4[1] Write
	dmaSourceFlexPWM4Write2 = 0x68 | 4 | 2 // (110) FlexPWM4[2] Write
	dmaSourceFlexPWM4Write3 = 0x68 | 4 | 3 // (111) FlexPWM4[3] Write
)

func (c DMAChannelID) allocated() bool    { return (1<<c)&chanMask != 0 }
func (c DMAChannelID) irq() uint8         { return uint8(c % NumDMAInterrupts) }
func (c DMAChannelID) irqInstalled() bool { return (1<<c.irq())&irqMask != 0 }
func (c DMAChannelID) irqConnected() bool { return (1<<c)&irqChan != 0 }

func (c DMAChannelID) initInterrupt() {

	// Flag this channel as having an interrupt handler now installed.
	irqChan |= 1 << c
	// Do not re-install ISR if one has already been created on the receiver's
	// DMA channel or on its surrogate pair channel; e.g., don't install a new
	// interrupt ISR for DMA channel 1 if we have previously installed one for
	// channel 17.
	if c.irqInstalled() {
		return
	}
	// Flag this interrupt as having an interrupt handler now installed, which
	// will handle interrupts for BOTH channels tied to the IRQ (if both channels
	// have been created/init'd).
	irqMask |= 1 << c.irq()

	// Interrupt has not yet been initialized. Install new interrupt handler.
	switch c {
	case 0, 16:
		_DMAInterrupt[0] = interrupt.New(IRQ_DMA0_DMA16,
			func(interrupt.Interrupt) { DMAChannelID(0).dispatchInterrupt() })
	case 1, 17:
		_DMAInterrupt[1] = interrupt.New(IRQ_DMA1_DMA17,
			func(interrupt.Interrupt) { DMAChannelID(1).dispatchInterrupt() })
	case 2, 18:
		_DMAInterrupt[2] = interrupt.New(IRQ_DMA2_DMA18,
			func(interrupt.Interrupt) { DMAChannelID(2).dispatchInterrupt() })
	case 3, 19:
		_DMAInterrupt[3] = interrupt.New(IRQ_DMA3_DMA19,
			func(interrupt.Interrupt) { DMAChannelID(3).dispatchInterrupt() })
	case 4, 20:
		_DMAInterrupt[4] = interrupt.New(IRQ_DMA4_DMA20,
			func(interrupt.Interrupt) { DMAChannelID(4).dispatchInterrupt() })
	case 5, 21:
		_DMAInterrupt[5] = interrupt.New(IRQ_DMA5_DMA21,
			func(interrupt.Interrupt) { DMAChannelID(5).dispatchInterrupt() })
	case 6, 22:
		_DMAInterrupt[6] = interrupt.New(IRQ_DMA6_DMA22,
			func(interrupt.Interrupt) { DMAChannelID(6).dispatchInterrupt() })
	case 7, 23:
		_DMAInterrupt[7] = interrupt.New(IRQ_DMA7_DMA23,
			func(interrupt.Interrupt) { DMAChannelID(7).dispatchInterrupt() })
	case 8, 24:
		_DMAInterrupt[8] = interrupt.New(IRQ_DMA8_DMA24,
			func(interrupt.Interrupt) { DMAChannelID(8).dispatchInterrupt() })
	case 9, 25:
		_DMAInterrupt[9] = interrupt.New(IRQ_DMA9_DMA25,
			func(interrupt.Interrupt) { DMAChannelID(9).dispatchInterrupt() })
	case 10, 26:
		_DMAInterrupt[10] = interrupt.New(IRQ_DMA10_DMA26,
			func(interrupt.Interrupt) { DMAChannelID(10).dispatchInterrupt() })
	case 11, 27:
		_DMAInterrupt[11] = interrupt.New(IRQ_DMA11_DMA27,
			func(interrupt.Interrupt) { DMAChannelID(11).dispatchInterrupt() })
	case 12, 28:
		_DMAInterrupt[12] = interrupt.New(IRQ_DMA12_DMA28,
			func(interrupt.Interrupt) { DMAChannelID(12).dispatchInterrupt() })
	case 13, 29:
		_DMAInterrupt[13] = interrupt.New(IRQ_DMA13_DMA29,
			func(interrupt.Interrupt) { DMAChannelID(13).dispatchInterrupt() })
	case 14, 30:
		_DMAInterrupt[14] = interrupt.New(IRQ_DMA14_DMA30,
			func(interrupt.Interrupt) { DMAChannelID(14).dispatchInterrupt() })
	case 15, 31:
		_DMAInterrupt[15] = interrupt.New(IRQ_DMA15_DMA31,
			func(interrupt.Interrupt) { DMAChannelID(15).dispatchInterrupt() })
	}
}

// dispatchInterrupt dispatches DMA channel ISRs. There is exactly one IRQ for
// each pair of 32 DMA channels (modulo 16); i.e., DMA channel pairs {0,16},
// {1,17}, ..., {15,31} each share a different IRQ — a total of 16 interrupts.
//
// Thus, when an interrupt is requested, we have to check both channels of the
// pair to determine which channel was the source of the interrupt.
//
// If an interrupt was requested on DMA channel N, we first check that element
// _DMAChannel[N] has been initialized (with method Configure) by testing if
// its embedded struct *DMA_Type (containing peripheral registers) is not nil.
// We can then call DMAChannel method handleInterrupt to actually process the
// interrupt request for that individual channel.
//
// After handleInterrupt has been called for a given channel, the interrupt
// request bit for that channel (in register INT) is cleared using the clear
// interrupt request register CINT.
func (c DMAChannelID) dispatchInterrupt() {
	if ch1 := c.irq(); _DMAChannel[ch1].DMA_Type != nil &&
		_DMAChannel[ch1].INT.HasBits(1<<ch1) {
		_DMAChannel[ch1].handleInterrupt()
		_DMAChannel[ch1].CINT.Set(uint8(ch1)) // Clear interrupt
	}
	if ch2 := c.irq() + NumDMAInterrupts; _DMAChannel[ch2].DMA_Type != nil &&
		_DMAChannel[ch2].INT.HasBits(1<<ch2) {
		_DMAChannel[ch2].handleInterrupt()
		_DMAChannel[ch2].CINT.Set(uint8(ch2)) // Clear interrupt
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

func (c DMAChannelID) EnableInterrupt(enable bool) {
	// Nothing to do unless this channel is connected to an interrupt.
	if !c.irqConnected() {
		return
	}
	if enable {
		_DMAInterrupt[c.irq()].Enable()
	} else {
		_DMAInterrupt[c.irq()].Disable()
		// // Clear interrupt on each channel
		if ch1 := c.irq(); _DMAChannel[ch1].DMA_Type != nil {
			_DMAChannel[ch1].CINT.Set(uint8(ch1))
		}
		if ch2 := c.irq() + NumDMAInterrupts; _DMAChannel[ch2].DMA_Type != nil {
			_DMAChannel[ch2].CINT.Set(uint8(ch2))
		}
	}
}

func EnableDMAInterrupts(enable bool) {
	mask := irqChan
	for mask != 0 {
		ch := DMAChannelID(bits.TrailingZeros32(mask))
		ch.EnableInterrupt(enable)
		mask &^= 1 << ch
	}
}

type TCD_Type struct {
	SADDR  volatile.Register32 // SADDR register, used to save source address
	SOFF   volatile.Register16 // SOFF register, save offset bytes every transfer
	ATTR   volatile.Register16 // ATTR register, source/destination transfer size and modulo
	NBYTES volatile.Register32 // Nbytes register, minor loop length in bytes
	SLAST  volatile.Register32 // SLAST register
	DADDR  volatile.Register32 // DADDR register, used for destination address
	DOFF   volatile.Register16 // DOFF register, used for destination offset
	CITER  volatile.Register16 // CITER register, current minor loop numbers, for unfinished minor loop.
	DLAST  volatile.Register32 // DLASTSGA register, next tcd address used in scatter-gather mode
	CSR    volatile.Register16 // CSR register, for TCD control status
	BITER  volatile.Register16 // BITER register, begin minor loop count.
}

func (t *TCD_Type) reset() { *t = TCD_Type{} }

func (c DMAChannelID) tcd() *TCD_Type {
	return (*TCD_Type)(unsafe.Pointer(
		uintptr(unsafe.Pointer(_DMAChannel[c].DMA_Type)) +
			unsafe.Offsetof(_DMAChannel[c].DMA_Type.TCD0_SADDR) +
			uintptr(c)*unsafe.Sizeof(TCD_Type{})),
	)
}
