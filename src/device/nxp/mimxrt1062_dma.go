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

	handleDone func()
	handleHalf func()
	handleSoft func()
}

type DMAChannelID uint8

func (id DMAChannelID) DMAChannel() *DMAChannel {
	return &_DMAChannel[id]
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

func (d *DMAChannel) IsComplete() bool     { return d.TCD.CSR.HasBits(DMA_TCD0_CSR_DONE) }
func (d *DMAChannel) ClearComplete()       { d.CDNE.Set(uint8(d.id)) }
func (d *DMAChannel) IsError() bool        { return d.ERR.HasBits(uint32(1) << d.id) }
func (d *DMAChannel) ClearError()          { d.CERR.Set(uint8(d.id)) }
func (d *DMAChannel) Source() uintptr      { return uintptr(d.TCD.SADDR.Get()) }
func (d *DMAChannel) Destination() uintptr { return uintptr(d.TCD.DADDR.Get()) }
func (d *DMAChannel) ClearInterrupt()      { d.CINT.Set(uint8(d.id)) }

func (d *DMAChannel) SetSource(src uintptr, len uint32) {
	attr := dmaTransferSizeAttr(len)
	d.TCD.SADDR.Set(uint32(src))
	d.TCD.SOFF.Set(0)
	d.TCD.ATTR.ReplaceBits(attr, 0xFF, 8)
	if InProgramMemory(src) || d.TCD.NBYTES.Get() == 0 {
		d.TCD.NBYTES.Set(len)
	}
	d.TCD.SLAST.Set(0)
}

func (d *DMAChannel) SetSource8(src *volatile.Register8) {
	d.SetSource(uintptr(unsafe.Pointer(src)), 1)
}
func (d *DMAChannel) SetSource16(src *volatile.Register16) {
	d.SetSource(uintptr(unsafe.Pointer(src)), 2)
}
func (d *DMAChannel) SetSource32(src *volatile.Register32) {
	d.SetSource(uintptr(unsafe.Pointer(src)), 4)
}

func (d *DMAChannel) SetDestination(dst uintptr, len uint32) {
	attr := dmaTransferSizeAttr(len)
	d.TCD.DADDR.Set(uint32(dst))
	d.TCD.DOFF.Set(0)
	d.TCD.ATTR.ReplaceBits(attr, 0xFF, 0)
	if InProgramMemory(dst) || d.TCD.NBYTES.Get() == 0 {
		d.TCD.NBYTES.Set(len)
	}
	d.TCD.DLAST.Set(0)
}

func (d *DMAChannel) SetDestination8(dst *volatile.Register8) {
	d.SetDestination(uintptr(unsafe.Pointer(dst)), 1)
}
func (d *DMAChannel) SetDestination16(dst *volatile.Register16) {
	d.SetDestination(uintptr(unsafe.Pointer(dst)), 2)
}
func (d *DMAChannel) SetDestination32(dst *volatile.Register32) {
	d.SetDestination(uintptr(unsafe.Pointer(dst)), 4)
}

// SetTransferSize sets the data size used for each triggered transfer.
func (d *DMAChannel) SetTransferSize(len uint32) {
	// will panic if len is invalid
	attr := dmaTransferSizeAttr(len)
	d.TCD.NBYTES.Set(len)
	if d.TCD.SOFF.Get() != 0 {
		d.TCD.SOFF.Set(uint16(len))
	}
	if d.TCD.DOFF.Get() != 0 {
		d.TCD.DOFF.Set(uint16(len))
	}
	d.TCD.ATTR.Set((d.TCD.ATTR.Get() & 0xF8F8) | (attr << 8) | attr)
}

// SetTransferCount sets the number of triggered transfers until complete.
func (d *DMAChannel) SetTransferCount(len uint32) {
	if !d.TCD.BITER.HasBits(DMA_TCD0_BITER_ELINKYES_ELINK) {
		if len < 0x8000 {
			d.TCD.BITER.Set(uint16(len))
			d.TCD.CITER.Set(uint16(len))
		}
	} else {
		if len < 0x200 {
			d.TCD.BITER.Set((d.TCD.BITER.Get() & 0xFE00) | uint16(len))
			d.TCD.CITER.Set((d.TCD.CITER.Get() & 0xFE00) | uint16(len))
		}
	}
}

func (d *DMAChannel) SetInterruptOnComplete(handle func()) {
	d.handleDone = handle
	if handle != nil {
		d.TCD.CSR.SetBits(DMA_TCD0_CSR_INTMAJOR)
	} else {
		d.TCD.CSR.ClearBits(DMA_TCD0_CSR_INTMAJOR)
	}
}

func (d *DMAChannel) SetInterruptOnHalfComplete(handle func()) {
	d.handleHalf = handle
	if handle != nil {
		d.TCD.CSR.SetBits(DMA_TCD0_CSR_INTHALF)
	} else {
		d.TCD.CSR.ClearBits(DMA_TCD0_CSR_INTHALF)
	}
}

func (d *DMAChannel) SetInterruptOnSoftware(handle func()) {
	d.handleSoft = handle
}

func (d *DMAChannel) SetInterruptPriority(priority uint8) {
	d.id.setInterruptPriority(priority)
}

func (d *DMAChannel) InterruptPending() bool {
	return d.id.interruptPending()
}

func (d *DMAChannel) SetInterruptPending(set bool) {
	d.id.setInterruptPending(set)
}

func (d *DMAChannel) SetDisableOnComplete(set bool) {
	if set {
		d.TCD.CSR.SetBits(DMA_TCD0_CSR_DREQ)
	} else {
		d.TCD.CSR.ClearBits(DMA_TCD0_CSR_DREQ)
	}
}

func (d *DMAChannel) SetHardwareTrigger(muxSrc DMAChannelID) {
	cfg := (uint32(muxSrc) << DMAMUX_CHCFG_SOURCE_Pos) & DMAMUX_CHCFG_SOURCE_Msk
	cfg |= DMAMUX_CHCFG_ENBL
	DMAMUX.CHCFG[d.id].Set(0)
	DMAMUX.CHCFG[d.id].Set(cfg)
}

func (d *DMAChannel) SetTriggerOnTransfer(src DMAChannelID) {
	c := src.DMAChannel()
	c.TCD.BITER.ReplaceBits(DMA_TCD0_BITER_ELINKYES_ELINK|
		(uint16(d.id)<<DMA_TCD0_BITER_ELINKYES_LINKCH_Pos),
		DMA_TCD0_BITER_ELINKYES_LINKCH_Msk, 0)
	c.TCD.CITER.Set(c.TCD.BITER.Get())
}

func (d *DMAChannel) SetTriggerOnCompletion(src DMAChannelID) {
	c := src.DMAChannel()
	c.TCD.CSR.ReplaceBits(DMA_TCD0_CSR_MAJORELINK|
		(uint16(d.id)<<DMA_TCD0_CSR_MAJORLINKCH_Pos),
		DMA_TCD0_CSR_MAJORLINKCH_Msk|DMA_TCD0_CSR_DONE, 0)
}

func (d *DMAChannel) Trigger() { d.SSRT.Set(uint8(d.id)) }

func (d *DMAChannel) handleInterrupt() {
	// Determine which interrupt handler to call — complete or half-complete
	if d.IsComplete() {
		if d.handleDone != nil {
			d.handleDone()
		}
	} else {
		if d.handleHalf != nil {
			d.handleHalf()
		}
	}
	// Always call software interrupt handler if installed
	if d.handleSoft != nil {
		d.handleSoft()
	}
}

const (
	dmaSourceNONE DMAChannelID = 0xFF

	dmaSourceFlexIO1Req0 DMAChannelID = 0 | 0x00 // (0)  FlexIO1 Request 0
	dmaSourceFlexIO1Req1 DMAChannelID = 0 | 0x00 // (0)  FlexIO1 Request 1
	dmaSourceFlexIO1Req2 DMAChannelID = 0 | 0x40 // (64) FlexIO1 Request 2
	dmaSourceFlexIO1Req3 DMAChannelID = 0 | 0x40 // (64) FlexIO1 Request 3

	dmaSourceFlexIO2Req0 DMAChannelID = 1 | 0x00 // (1)  FlexIO2 Request 0
	dmaSourceFlexIO2Req1 DMAChannelID = 1 | 0x00 // (1)  FlexIO2 Request 1
	dmaSourceFlexIO2Req2 DMAChannelID = 1 | 0x40 // (65) FlexIO2 Request 2
	dmaSourceFlexIO2Req3 DMAChannelID = 1 | 0x40 // (65) FlexIO2 Request 3

	dmaSourceFlexPWM1Read0  DMAChannelID = 0x20 | 0 | 0 // (32)  FlexPWM1[0] Read
	dmaSourceFlexPWM1Read1  DMAChannelID = 0x20 | 0 | 1 // (33)  FlexPWM1[1] Read
	dmaSourceFlexPWM1Read2  DMAChannelID = 0x20 | 0 | 2 // (34)  FlexPWM1[2] Read
	dmaSourceFlexPWM1Read3  DMAChannelID = 0x20 | 0 | 3 // (35)  FlexPWM1[3] Read
	dmaSourceFlexPWM1Write0 DMAChannelID = 0x20 | 4 | 0 // (36)  FlexPWM1[0] Write
	dmaSourceFlexPWM1Write1 DMAChannelID = 0x20 | 4 | 1 // (37)  FlexPWM1[1] Write
	dmaSourceFlexPWM1Write2 DMAChannelID = 0x20 | 4 | 2 // (38)  FlexPWM1[2] Write
	dmaSourceFlexPWM1Write3 DMAChannelID = 0x20 | 4 | 3 // (39)  FlexPWM1[3] Write

	dmaSourceFlexPWM2Read0  DMAChannelID = 0x60 | 0 | 0 //	(96)  FlexPWM2[0] Read
	dmaSourceFlexPWM2Read1  DMAChannelID = 0x60 | 0 | 1 //	(97)  FlexPWM2[1] Read
	dmaSourceFlexPWM2Read2  DMAChannelID = 0x60 | 0 | 2 //	(98)  FlexPWM2[2] Read
	dmaSourceFlexPWM2Read3  DMAChannelID = 0x60 | 0 | 3 //	(99)  FlexPWM2[3] Read
	dmaSourceFlexPWM2Write0 DMAChannelID = 0x60 | 4 | 0 //	(100) FlexPWM2[0] Write
	dmaSourceFlexPWM2Write1 DMAChannelID = 0x60 | 4 | 1 //	(101) FlexPWM2[1] Write
	dmaSourceFlexPWM2Write2 DMAChannelID = 0x60 | 4 | 2 //	(102) FlexPWM2[2] Write
	dmaSourceFlexPWM2Write3 DMAChannelID = 0x60 | 4 | 3 //	(103) FlexPWM2[3] Write

	dmaSourceFlexPWM3Read0  DMAChannelID = 0x28 | 0 | 0 //	(40)  FlexPWM3[0] Read
	dmaSourceFlexPWM3Read1  DMAChannelID = 0x28 | 0 | 1 //	(41)  FlexPWM3[1] Read
	dmaSourceFlexPWM3Read2  DMAChannelID = 0x28 | 0 | 2 //	(42)  FlexPWM3[2] Read
	dmaSourceFlexPWM3Read3  DMAChannelID = 0x28 | 0 | 3 //	(43)  FlexPWM3[3] Read
	dmaSourceFlexPWM3Write0 DMAChannelID = 0x28 | 4 | 0 //	(44)  FlexPWM3[0] Write
	dmaSourceFlexPWM3Write1 DMAChannelID = 0x28 | 4 | 1 //	(45)  FlexPWM3[1] Write
	dmaSourceFlexPWM3Write2 DMAChannelID = 0x28 | 4 | 2 //	(46)  FlexPWM3[2] Write
	dmaSourceFlexPWM3Write3 DMAChannelID = 0x28 | 4 | 3 //	(47)  FlexPWM3[3] Write

	dmaSourceFlexPWM4Read0  DMAChannelID = 0x68 | 0 | 0 // (104) FlexPWM4[0] Read
	dmaSourceFlexPWM4Read1  DMAChannelID = 0x68 | 0 | 1 // (105) FlexPWM4[1] Read
	dmaSourceFlexPWM4Read2  DMAChannelID = 0x68 | 0 | 2 // (106) FlexPWM4[2] Read
	dmaSourceFlexPWM4Read3  DMAChannelID = 0x68 | 0 | 3 // (107) FlexPWM4[3] Read
	dmaSourceFlexPWM4Write0 DMAChannelID = 0x68 | 4 | 0 // (108) FlexPWM4[0] Write
	dmaSourceFlexPWM4Write1 DMAChannelID = 0x68 | 4 | 1 // (109) FlexPWM4[1] Write
	dmaSourceFlexPWM4Write2 DMAChannelID = 0x68 | 4 | 2 // (110) FlexPWM4[2] Write
	dmaSourceFlexPWM4Write3 DMAChannelID = 0x68 | 4 | 3 // (111) FlexPWM4[3] Write
)

const (
	DMATransferSize1Bytes  = 0x0 // transfer 1 byte (8 bits) every time
	DMATransferSize2Bytes  = 0x1 // transfer 2 bytes (16 bits) every time
	DMATransferSize4Bytes  = 0x2 // transfer 4 bytes (32 bits) every time
	DMATransferSize8Bytes  = 0x3 // transfer 8 bytes (64 bits) every time
	DMATransferSize16Bytes = 0x4 // transfer 16 bytes (128 bits) every time
	DMATransferSize32Bytes = 0x5 // transfer 32 bytes (256 bits) every time
)

func dmaTransferSizeAttr(bytes uint32) uint16 {
	if bits.OnesCount32(bytes) != 1 {
		panic("invalid DMA source transfer size")
	}
	return uint16(bits.TrailingZeros32(bytes))
}

func (c DMAChannelID) allocated() bool    { return (1<<c)&chanMask != 0 }
func (c DMAChannelID) irq() uint8         { return uint8(c % NumDMAInterrupts) }
func (c DMAChannelID) irqInstalled() bool { return (1<<c.irq())&irqMask != 0 }
func (c DMAChannelID) irqConnected() bool { return (1<<c)&irqChan != 0 }

func (c DMAChannelID) initInterrupt() {

	// Flag this channel as having an interrupt handler now installed.
	irqChan |= 1 << c
	// Do not re-install ISR if one has already been created on the receiver's
	// DMA channel or on its surrogate pair's channel; e.g., don't install a new
	// interrupt ISR for DMA channel 1 if we have previously installed one for
	// channel 17.
	// This will also catch the case when we are called with a channel that has
	// already been init'd as well, since it obviously will have the same IRQ.
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

func (c DMAChannelID) setInterruptPriority(priority uint8) {
	// Nothing to do unless this IRQ has an interrupt installed.
	if !c.irqInstalled() {
		return
	}
	_DMAInterrupt[c.irq()].SetPriority(priority)
}

const (
	nvicSetPending = 0xE000E200
	nvicClrPending = 0xE000E280
)

func (c DMAChannelID) interruptPending() bool {
	// Not pending unless this IRQ has an interrupt installed.
	if !c.irqInstalled() {
		return false
	}
	irq := c.irq()
	return ((*volatile.Register32)(unsafe.Pointer(
		nvicSetPending | (uintptr(irq) >> 5),
	))).HasBits(
		uint32(1) << (irq & 0x1F),
	)
}

func (c DMAChannelID) setInterruptPending(set bool) {
	// Nothing to do unless this IRQ has an interrupt installed.
	if !c.irqInstalled() {
		return
	}
	irq := c.irq()
	address := uintptr(irq) >> 5
	if set {
		address |= nvicSetPending
	} else {
		address |= nvicClrPending
	}
	((*volatile.Register32)(unsafe.Pointer(address))).Set(
		uint32(1) << (irq & 0x1F),
	)
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
	// Dispatch ISR if group 1 interrupt initialized and request bit set.
	irq := c.irq()
	if _DMAChannel[irq].DMA_Type != nil &&
		_DMAChannel[irq].INT.HasBits(1<<irq) {
		_DMAChannel[irq].handleInterrupt()
		_DMAChannel[irq].CINT.Set(irq) // Clear interrupt
	}
	// Dispatch ISR if group 2 interrupt initialized and request bit set.
	irq += NumDMAInterrupts
	if _DMAChannel[irq].DMA_Type != nil &&
		_DMAChannel[irq].INT.HasBits(1<<irq) {
		_DMAChannel[irq].handleInterrupt()
		_DMAChannel[irq].CINT.Set(irq) // Clear interrupt
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
	// Nothing to do unless this IRQ has an interrupt installed.
	if !c.irqInstalled() {
		return
	}
	irq := c.irq()
	if enable {
		// Enable interrupt via NVIC controller
		_DMAInterrupt[irq].Enable()
	} else {
		// Disable interrupt via NVIC controller
		_DMAInterrupt[irq].Disable()
		// Clear interrupt on group 1 channel
		if _DMAChannel[irq].DMA_Type != nil {
			_DMAChannel[irq].CINT.Set(irq)
		}
		// Clear interrupt on group 2 channel
		irq += NumDMAInterrupts
		if _DMAChannel[irq].DMA_Type != nil {
			_DMAChannel[irq].CINT.Set(irq)
		}
	}
}

func EnableDMAInterrupts(enable bool) {
	mask := irqMask
	for mask != 0 {
		ch := DMAChannelID(bits.TrailingZeros16(mask))
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
