//go:build ra4m1

package machine

import (
	"device/renesas"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

// The SCI modules are clocked by PCLKA, which runtime_ra4m1.go sets to 48 MHz.
// See RA4M1 User's Manual (R01UH0887) section 8.2.1 and table 28.1.
const sciClockFrequency = 48_000_000

// PSEL values that select the SCI peripheral function on a pin.
// See RA4M1 User's Manual (R01UH0887) table 19.7.
const (
	pselSCIEven uint32 = 0x04 // SCI0, SCI2, SCI4, SCI6, SCI8
	pselSCIOdd  uint32 = 0x05 // SCI1, SCI3, SCI5, SCI7, SCI9
)

// UART on the RA4M1 is a SCI module in asynchronous mode.
type UART struct {
	Buffer    *RingBuffer
	Bus       *renesas.SCI2_Type
	rxIRQ     interrupt.Interrupt
	errIRQ    interrupt.Interrupt
	rxIELSR   *volatile.Register32
	errIELSR  *volatile.Register32
	event     uint32 // ELC event number of RXI. TXI, TEI and ERI follow in order.
	mstpMask  uint32
	psel      uint32
	defaultTX Pin
	defaultRX Pin
}

var (
	// sci1UART uses SCI1 on IELSR4 (RXI) and IELSR5 (ERI).
	// SCI1 has FIFO registers after SEMR, but the registers used by this
	// driver share the SCI2 layout. The FIFO stays disabled (FCR.FM=0).
	sci1UART  = &_sci1UART
	_sci1UART = UART{
		Buffer:   NewRingBuffer(),
		Bus:      (*renesas.SCI2_Type)(unsafe.Pointer(renesas.SCI1)),
		rxIELSR:  &renesas.ICU.IELSR4,
		errIELSR: &renesas.ICU.IELSR5,
		event:    0x9E, // ELC_EVENT_SCI1_RXI
		mstpMask: renesas.MSTP_MSTPCRB_MSTPB30_Msk,
		psel:     pselSCIOdd,
	}

	// sci2UART uses SCI2 on IELSR2 (RXI) and IELSR3 (ERI).
	sci2UART  = &_sci2UART
	_sci2UART = UART{
		Buffer:   NewRingBuffer(),
		Bus:      renesas.SCI2,
		rxIELSR:  &renesas.ICU.IELSR2,
		errIELSR: &renesas.ICU.IELSR3,
		event:    0xA3, // ELC_EVENT_SCI2_RXI
		mstpMask: renesas.MSTP_MSTPCRB_MSTPB29_Msk,
		psel:     pselSCIEven,
	}

	// sci9UART uses SCI9 on IELSR0 (RXI) and IELSR1 (ERI).
	sci9UART  = &_sci9UART
	_sci9UART = UART{
		Buffer:   NewRingBuffer(),
		Bus:      renesas.SCI9,
		rxIELSR:  &renesas.ICU.IELSR0,
		errIELSR: &renesas.ICU.IELSR1,
		event:    0xA8, // ELC_EVENT_SCI9_RXI
		mstpMask: renesas.MSTP_MSTPCRB_MSTPB22_Msk,
		psel:     pselSCIOdd,
	}
)

func init() {
	sci1UART.rxIRQ = interrupt.New(renesas.IRQ_IEL4, _sci1UART.handleRXI)
	sci1UART.errIRQ = interrupt.New(renesas.IRQ_IEL5, _sci1UART.handleERI)
	sci2UART.rxIRQ = interrupt.New(renesas.IRQ_IEL2, _sci2UART.handleRXI)
	sci2UART.errIRQ = interrupt.New(renesas.IRQ_IEL3, _sci2UART.handleERI)
	sci9UART.rxIRQ = interrupt.New(renesas.IRQ_IEL0, _sci9UART.handleRXI)
	sci9UART.errIRQ = interrupt.New(renesas.IRQ_IEL1, _sci9UART.handleERI)
}

// Configure the UART for asynchronous 8N1 operation.
func (uart *UART) Configure(config UARTConfig) {
	if config.BaudRate == 0 {
		config.BaudRate = 115200
	}
	if config.TX == 0 && config.RX == 0 {
		config.TX = uart.defaultTX
		config.RX = uart.defaultRX
	}

	// Release the module stop state. MSTPCRB is not write protected by PRCR.
	renesas.MSTP.MSTPCRB.ClearBits(uart.mstpMask)

	// Initialization sequence from R01UH0887 section 28.3.7, figure 28.9.
	uart.Bus.SCR.Set(0)
	uart.Bus.SCMR.Set(0xF2) // reset value, CHR1=1 selects 8 bit data
	uart.Bus.SMR.Set(0)
	uart.Bus.SEMR.Set(renesas.SCI2_SEMR_ABCS_Msk | renesas.SCI2_SEMR_BGDM_Msk)
	uart.SetBaudRate(config.BaudRate)

	uart.configurePin(config.TX)
	uart.configurePin(config.RX)

	uart.unlinkStaleEvents()
	uart.rxIELSR.Set(uart.event + eventRXI)
	uart.errIELSR.Set(uart.event + eventERI)
	uart.rxIRQ.Enable()
	uart.errIRQ.Enable()

	uart.Bus.SCR.Set(renesas.SCI2_SCR_TE_Msk | renesas.SCI2_SCR_RE_Msk | renesas.SCI2_SCR_RIE_Msk)
}

// SCI event offsets relative to the RXI event. See FSP ra4m1/bsp_elc.h.
const (
	eventRXI = 0
	eventTXI = 1
	eventTEI = 2
	eventERI = 3
)

// unlinkStaleEvents clears every ICU slot that still links an event of this
// SCI channel. The Arduino bootloader leaves SCI9 linked on IELSR4 to IELSR7.
// An event linked in two slots is delivered to only one of them.
func (uart *UART) unlinkStaleEvents() {
	slots := (*[32]volatile.Register32)(unsafe.Pointer(&renesas.ICU.IELSR0))
	for i := range slots {
		ev := slots[i].Get() & renesas.ICU_IELSR_IELS_Msk
		if ev >= uart.event+eventRXI && ev <= uart.event+eventERI {
			slots[i].Set(0)
		}
	}
}

// SetBaudRate programs CKS and BRR for the requested baud rate.
// With ABCS=1 and BGDM=1 the divisor is 8*4^CKS. See R01UH0887 table 28.10.
// SMR and BRR must be written with TE and RE cleared. See R01UH0887 28.3.7.
func (uart *UART) SetBaudRate(br uint32) {
	scr := uart.Bus.SCR.Get()
	uart.Bus.SCR.Set(0)
	defer uart.Bus.SCR.Set(scr)

	divisor := uint32(8)
	for cks := uint32(0); cks < 4; cks++ {
		n := (sciClockFrequency + divisor*br/2) / (divisor * br)
		if n >= 1 && n <= 256 {
			uart.Bus.SMR.ReplaceBits(uint8(cks), renesas.SCI2_SMR_CKS_Msk, 0)
			uart.Bus.BRR.Set(uint8(n - 1))
			return
		}
		divisor *= 4
	}
	uart.Bus.SMR.ReplaceBits(3, renesas.SCI2_SMR_CKS_Msk, 0)
	uart.Bus.BRR.Set(0xFF)
}

func (uart *UART) configurePin(p Pin) {
	if p == NoPin {
		return
	}
	enableWritingPmnPFS()
	port := getPortNumber(p)
	bit := getPinNumber(p)
	reg := (*volatile.Register32)(
		unsafe.Add(unsafe.Pointer(renesas.PFS), 0x40*port+4*int(bit)),
	)
	mode := uart.psel<<renesas.PFS_P000PFS_PSEL_Pos | renesas.PFS_P000PFS_PMR_Msk
	modeMask := uint32(renesas.PFS_P000PFS_PSEL_Msk |
		renesas.PFS_P000PFS_PMR_Msk |
		renesas.PFS_P000PFS_ASEL_Msk |
		renesas.PFS_P000PFS_ISEL_Msk |
		renesas.PFS_P000PFS_PCR_Msk |
		renesas.PFS_P000PFS_PDR_Msk)
	reg.ReplaceBits(mode, modeMask, 0)
	disableWritingPmnPFS()
}

// handleRXI services the receive data full event.
func (uart *UART) handleRXI(interrupt.Interrupt) {
	uart.Receive(uart.Bus.RDR.Get())
	uart.rxIELSR.ClearBits(renesas.ICU_IELSR_IR_Msk)
	// Read back so the IR clear completes before exception return.
	// See FSP bsp_irq.h R_BSP_IrqStatusClear.
	uart.rxIELSR.Get()
}

// handleERI services overrun, framing and parity errors.
func (uart *UART) handleERI(interrupt.Interrupt) {
	ssr := uart.Bus.SSR.Get()
	if ssr&renesas.SCI2_SSR_RDRF_Msk != 0 {
		uart.Receive(uart.Bus.RDR.Get())
	}
	const errMask = renesas.SCI2_SSR_ORER_Msk | renesas.SCI2_SSR_FER_Msk | renesas.SCI2_SSR_PER_Msk
	// Error flags clear on a write of 0. Reception resumes afterwards.
	uart.Bus.SSR.Set(ssr &^ errMask)
	uart.errIELSR.ClearBits(renesas.ICU_IELSR_IR_Msk)
	uart.errIELSR.Get()
}

func (uart *UART) writeByte(c byte) error {
	for !uart.Bus.SSR.HasBits(renesas.SCI2_SSR_TDRE_Msk) {
	}
	uart.Bus.TDR.Set(c)
	return nil
}

func (uart *UART) flush() {
	for !uart.Bus.SSR.HasBits(renesas.SCI2_SSR_TEND_Msk) {
	}
}

// DTR is not available on the RA4M1 SCI. Returns true so programs that wait
// for a serial console do not block.
func (uart *UART) DTR() bool {
	return true
}

// RTS is not available on the RA4M1 SCI.
func (uart *UART) RTS() bool {
	return false
}
