//go:build usb.cdc && (atsamd51 || atsame5x)
// +build usb.cdc
// +build atsamd51 atsame5x

package usb

import "runtime/volatile"

// descCDCCount defines the number of USB cores that may be configured as
// CDC-ACM (single) devices.
const descCDCCount = 1

// Constants for USB CDC-ACM device classes.
const (

	// USB Bus Configuration Attributes

	descCDCMaxPowerMa = 100 // Maximum current (mA) requested from host

	// CDC-ACM Endpoint Descriptor Buffers

	descCDCEDCount = descMaxEndpoints

	// Setup packet is only 8 bytes in length. However, under certain scenarios,
	// USB DMA controller may decide to overwrite/overflow the buffer with 2 extra
	// bytes of CRC. From datasheet's "Management of SETUP Transactions" section:
	//   | If the number of received data bytes is the maximum data payload
	//   | specified by PCKSIZE.SIZE minus one, only the first CRC data is written
	//   | to the data buffer. If the number of received data is equal or less
	//   | than the data payload specified by PCKSIZE.SIZE minus two, both CRC
	//   | data bytes are written to the data buffer.
	// Thus, we need to allocate 2 extra bytes for control endpoint 0 Rx (OUT).
	descCDCSxSize = 8 + 2
	descCDCCxSize = descControlPacketSize

	// CDC-ACM Data Buffers

	descCDCRxSize = descCDCDataRxPacketSize
	descCDCTxSize = descCDCDataTxPacketSize

	descCDCTxTimeoutMs = 120 // millisec
	descCDCTxSyncUs    = 75  // microsec

	// Default CDC-ACM Endpoint Configurations (Full-Speed)

	descCDCStatusInterval   = descCDCStatusFSInterval   // Status
	descCDCStatusPacketSize = descCDCStatusFSPacketSize //

	descCDCDataRxPacketSize = descCDCDataRxFSPacketSize // Data Rx
	descCDCDataTxPacketSize = descCDCDataTxFSPacketSize // Data Tx

	// CDC-ACM Endpoint Configurations for Full-Speed Device

	descCDCStatusFSInterval   = 5  // Status
	descCDCStatusFSPacketSize = 64 //  (full-speed)

	descCDCDataRxFSPacketSize = 64 // Data Rx (full-speed)
	descCDCDataTxFSPacketSize = 64 // Data Tx (full-speed)

	// CDC-ACM Endpoint Configurations for High-Speed Device

	//  - N/A, SAMx51 only has a full-speed PHY
)

// descCDC0ED is an array of endpoint descriptors, which describes to the USB
// DMA controller the buffer and transfer properties for each endpoint, for the
// default CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0ED [descCDCEDCount]dhwEPAddrDesc

// descCDC0Sx is the receive (Rx) buffer for setup packets on control endpoint
// 0 of the default CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Sx [descCDCSxSize]uint8

// descCDC0Cx is the transmit (Tx) buffer for control/status packets on control
// endpoint 0 of the default CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Cx [descCDCCxSize]uint8

// descCDC0Dx is the transmit (Tx) buffer of descriptor data on endpoint 0
// for the default CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Dx [descCDCConfigSize]uint8

// descCDC0Rx is the receive (Rx) transfer buffer for the default CDC-ACM
// (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Rx [descCDCRxSize]uint8

// descCDC0Tx is the transmit (Tx) transfer buffer for the default CDC-ACM
// (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Tx [descCDCTxSize]uint8

// descCDC0Rq is the receive (Rx) transfer buffer for the default CDC-ACM
// (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Rq [descCDCRxSize]uint8

// descCDC0Tq is the transmit (Tx) transfer buffer for the default CDC-ACM
// (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Tq [descCDCTxSize]uint8

// descCDC0LC is the emulated UART's line coding configuration for the
// default CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0LC descCDCLineCoding

// descCDC0LS is the emulated UART's line state for the default CDC-ACM
// (single) device class configuration (index 1).
//
//go:align 32
var descCDC0LS descCDCLineState

// descCDCClassData holds the buffers and control states for all CDC-ACM
// (single) device class configurations, ordered by index (offset by -1), for
// SAMx51 targets only.
//
// Instances of this type (elements of descCDCData) are embedded in elements
// of the common/target-agnostic CDC-ACM class configurations (descCDC).
// Methods defined on this type implement target-specific functionality, and
// some of these methods are required by the common device controller driver.
// Thus, this type functions as a hardware abstraction layer (HAL).
type descCDCClassData struct {
	// CDC-ACM Control Buffers

	ed *[descCDCEDCount]dhwEPAddrDesc // endpoint descriptors

	sx *[descCDCSxSize]uint8     // control endpoint 0 Rx (OUT) setup packets
	cx *[descCDCCxSize]uint8     // control endpoint 0 Tx (IN) control/status packets
	dx *[descCDCConfigSize]uint8 // control endpoint 0 Tx (IN) descriptor transfer buffer

	// CDC-ACM Data Buffers

	rx *[descCDCRxSize]uint8 // bulk data endpoint Rx (OUT) transfer buffer
	tx *[descCDCTxSize]uint8 // bulk data endpoint Tx (IN) transfer buffer

	rxq *[descCDCRxSize]uint8 // CDC-ACM UART Rx FIFO
	txq *[descCDCTxSize]uint8 // CDC-ACM UART Tx FIFO

	rq *Queue // CDC-ACM UART Rx Queue (backed by FIFO rxq)
	tq *Queue // CDC-ACM UART Tx Queue (backed by FIFO txq)

	lc *descCDCLineCoding // UART line coding
	ls *descCDCLineState  // UART line state

	st volatile.Register8

	sxSize uint32
	rxSize uint32
	txSize uint32
}

// setState is a wrapper for converting and storing the given descCDCState
// value as a uint8 in the receiver's volatile.Register8 field st.
//
//go:inline
func (c *descCDCClassData) setState(state descCDCState) {
	s := descCDCState(c.st.Get())
	c.st.Set(uint8(s.set(state)))
}

// state is a wrapper for retrieving and converting the receiver's
// volatile.Register8 field st from uint8 to descCDCState.
//
//go:inline
func (c *descCDCClassData) state() descCDCState {
	return descCDCState(c.st.Get())
}

// descCDCData holds statically-allocated instances for each of the target-
// specific (SAMx51) CDC-ACM (single) device class configurations' control and
// data structures, ordered by configuration index (offset by -1). Each element
// is embedded in a corresponding element of descCDC.
var descCDCData = [dcdCount]descCDCClassData{
	{ // -- CDC-ACM (single) Class Configuration Index 1 --

		// CDC-ACM Control Buffers

		ed: &descCDC0ED,

		sx: &descCDC0Sx,
		cx: &descCDC0Cx,
		dx: &descCDC0Dx,

		// CDC-ACM Data Buffers

		rx: &descCDC0Rx,
		tx: &descCDC0Tx,

		rxq: &descCDC0Rq,
		txq: &descCDC0Tq,

		rq: &Queue{},
		tq: &Queue{},

		lc: &descCDC0LC,
		ls: &descCDC0LS,

		sxSize: descCDCStatusPacketSize,
		rxSize: descCDCDataRxPacketSize,
		txSize: descCDCDataTxPacketSize,
	},
}
