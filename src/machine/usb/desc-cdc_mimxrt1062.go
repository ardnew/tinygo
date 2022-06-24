//go:build usb.cdc && mimxrt1062
// +build usb.cdc,mimxrt1062

package usb

import "runtime/volatile"

// descCDCCount defines the number of USB cores that may be configured as
// CDC-ACM (single) devices.
const descCDCCount = 1

// Constants for USB CDC-ACM device classes.
const (

	// USB bus configuration attributes

	descCDCMaxPowerMa = 100 // Maximum current (mA) requested from host

	// CDC-ACM Control Buffers

	descCDCQHCount = 2 * (descCDCEndpointCount + 1)
	descCDCCxCount = 8

	// CDC-ACM Data Buffers

	descCDCRDCount = 2 * descCDCEndpointCount
	descCDCRxSize  = descCDCDataRxPacketSize
	descCDCRxCount = descCDCRxSize * descCDCRDCount

	descCDCTDCount = descCDCEndpointCount
	descCDCTxSize  = 4 * descCDCDataTxPacketSize
	descCDCTxCount = descCDCTxSize * descCDCTDCount

	descCDCTxTimeoutMs = 120 // millisec
	descCDCTxSyncUs    = 75  // microsec

	// Default CDC-ACM Endpoint Configurations (High-Speed)

	descCDCStatusInterval   = descCDCStatusHSInterval   // Status
	descCDCStatusPacketSize = descCDCStatusHSPacketSize //

	descCDCDataRxPacketSize = descCDCDataRxHSPacketSize // Data Rx

	descCDCDataTxPacketSize = descCDCDataTxHSPacketSize // Data Tx

	// CDC-ACM Endpoint Configurations for Full-Speed Device

	descCDCStatusFSInterval   = 5  // Status
	descCDCStatusFSPacketSize = 16 //  (full-speed)

	descCDCDataRxFSPacketSize = 64 // Data Rx (full-speed)

	descCDCDataTxFSPacketSize = 64 // Data Tx (full-speed)

	// CDC-ACM Endpoint Configurations for High-Speed Device

	descCDCStatusHSInterval   = 5  // Status
	descCDCStatusHSPacketSize = 16 //  (high-speed)

	descCDCDataRxHSPacketSize = 512 // Data Rx (high-speed)

	descCDCDataTxHSPacketSize = 512 // Data Tx (high-speed)
)

// descCDC0QH is an array of endpoint queue heads, which is where all
// transfers for a given endpoint are managed, for the default CDC-ACM (single)
// device class configuration (index 1).
//
// From the iMXRT1062 Reference Manual:
//
//	Software must ensure that no interface data structure reachable
//	by the Device Controller spans a 4K-page boundary.
//
//	The [queue head] is a 48-byte data structure, but must be aligned on
//	64-byte boundaries.
//
//	Endpoint queue heads are arranged in an array in a continuous area of
//	memory pointed to by the USB.ENDPOINTLISTADDR pointer. The even-numbered
//	device queue heads in the list support receive endpoints (OUT/SETUP) and
//	the odd-numbered queue heads in the list are used for transmit endpoints
//	(IN/INTERRUPT). The device controller will index into this array based upon
//	the endpoint number received from the USB bus. All information necessary to
//	respond to transactions for all primed transfers is contained in this list
//	so the Device Controller can readily respond to incoming requests without
//	having to traverse a linked list.
//
//go:align 4096
var descCDC0QH [descCDCQHCount]dhwEndpoint

// descCDC0CD is the transfer descriptor for messages transmitted or received
// on the status/control endpoint 0 for the default CDC-ACM (single) device
// class configuration (index 1).
//
//go:align 32
var descCDC0CD dhwTransfer

// descCDC0Cx is the buffer for control/status data received on endpoint 0 of
// the default CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Cx [descCDCCxCount]uint8

// descCDC0AD is the transfer descriptor for ackowledgement (ACK) messages
// transmitted or received on the status/control endpoint 0 for the default
// CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0AD dhwTransfer

// descCDC0Dx is the transmit (Tx) buffer of descriptor data on endpoint 0
// for the default CDC-ACM (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Dx [descCDCConfigSize]uint8

// descCDC0RD is an array of transfer descriptors for Rx (OUT) transfers,
// which describe to the device controller the location and quantity of data
// being received for a given transfer, for the default CDC-ACM (single) device
// class configuration (index 1).
//
//go:align 32
var descCDC0RD [descCDCRDCount]dhwTransfer

// descCDC0Rx is the receive (Rx) transfer buffer for the default CDC-ACM
// (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Rx [descCDCRxCount]uint8

// descCDC0TD is an array of transfer descriptors for Tx (IN) transfers,
// which describe to the device controller the location and quantity of data
// being transmitted for a given transfer, for the default CDC-ACM (single)
// device class configuration (index 1).
//
//go:align 32
var descCDC0TD [descCDCTDCount]dhwTransfer

// descCDC0Tx is the transmit (Tx) transfer buffer for the default CDC-ACM
// (single) device class configuration (index 1).
//
//go:align 32
var descCDC0Tx [descCDCTxCount]uint8

var (
	descCDC0RDNum [descCDCRDCount]uint16
	descCDC0RDIdx [descCDCRDCount]uint16
	descCDC0RDQue [(descCDCRDCount + 1)]uint16
)

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
// iMXRT1062 targets only.
//
// Instances of this type (elements of descCDCData) are embedded in elements
// of the common/target-agnostic CDC-ACM class configurations (descCDC).
// Methods defined on this type implement target-specific functionality, and
// some of these methods are required by the common device controller driver.
// Thus, this type functions as a hardware abstraction layer (HAL).
type descCDCClassData struct {
	// CDC-ACM Control Buffers

	qh *[descCDCQHCount]dhwEndpoint // endpoint queue heads

	cd *dhwTransfer              // control endpoint 0 Rx/Tx transfer descriptor
	cx *[descCDCCxCount]uint8    // control endpoint 0 Rx/Tx transfer buffer
	ad *dhwTransfer              // control endpoint 0 Rx/Tx ACK transfer descriptor
	dx *[descCDCConfigSize]uint8 // control endpoint 0 Tx (IN) descriptor transfer buffer

	// CDC-ACM Data Buffers

	rd *[descCDCRDCount]dhwTransfer // bulk data endpoint Rx (OUT) transfer descriptors
	rx *[descCDCRxCount]uint8       // bulk data endpoint Rx (OUT) transfer buffer
	td *[descCDCTDCount]dhwTransfer // bulk data endpoint Tx (IN) transfer descriptors
	tx *[descCDCTxCount]uint8       // bulk data endpoint Tx (IN) transfer buffer

	rxCount *[descCDCRDCount]uint16
	rxIndex *[descCDCRDCount]uint16
	rxQueue *[(descCDCRDCount + 1)]uint16

	lc *descCDCLineCoding // UART line coding
	ls *descCDCLineState  // UART line state

	st volatile.Register8

	sxSize uint16
	rxSize uint16
	txSize uint16

	txHead uint8
	txFree uint16
	txPrev bool

	rxHead uint8
	rxTail uint8
	rxFree uint16
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
// specific (iMXRT1062) CDC-ACM (single) device class configurations' control
// and data structures, ordered by configuration index (offset by -1). Each
// element is embedded in a corresponding element of descCDC.
//
//go:align 64
var descCDCData = [dcdCount]descCDCClassData{
	{ // -- CDC-ACM (single) Class Configuration Index 1 --

		// CDC-ACM Control Buffers

		qh: &descCDC0QH,

		cd: &descCDC0CD,
		cx: &descCDC0Cx,
		ad: &descCDC0AD,
		dx: &descCDC0Dx,

		// CDC-ACM Data Buffers

		rd: &descCDC0RD,
		rx: &descCDC0Rx,
		td: &descCDC0TD,
		tx: &descCDC0Tx,

		rxCount: &descCDC0RDNum,
		rxIndex: &descCDC0RDIdx,
		rxQueue: &descCDC0RDQue,

		lc: &descCDC0LC,
		ls: &descCDC0LS,

		sxSize: descCDCStatusPacketSize,
		rxSize: descCDCDataRxPacketSize,
		txSize: descCDCDataTxPacketSize,
	},
}
