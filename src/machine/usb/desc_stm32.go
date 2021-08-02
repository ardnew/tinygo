// +build stm32

package usb

import "runtime/volatile"

// descCDCACMCount defines the number of USB cores that may be configured as
// CDC-ACM (single) devices.
const descCDCACMCount = 1

// descHIDCount defines the number of USB cores that may be configured as a
// composite (keyboard+mouse+joystick+serial) human interface device (HID).
const descHIDCount = 0

// General USB device identification constants.
const (
	descCommonVendorID  = 0x0483
	descCommonProductID = 0x5740
	descCommonReleaseID = 0x0101 // BCD (1.1)

	descCommonLanguage     = descLanguageEnglish
	descCommonManufacturer = "TinyGo"
	descCommonProduct      = "USB"
	descCommonSerialNumber = "000018"
)

// Constants for USB CDC-ACM device classes.
const (

	// CDC-ACM Control Endpoint 0 Configuration

	descCDCACMCxCount = 64 // Control buffer is always 64 bytes

	// CDC-ACM UART Buffer & Transfer Configuration

	descCDCACMTxTimeoutMs = 120 // millisec
	descCDCACMTxSyncUs    = 75  // microsec

	// CDC-ACM Endpoint Configurations for Full-Speed Device

	descCDCACMRxFIFOFSSize = 64 << 3
	descCDCACMTxFIFOFSSize = 64 << 3

	descCDCACMStatusFSInterval   = 5  // Status
	descCDCACMStatusFSPacketSize = 16 //  (full-speed)
	descCDCACMDataRxFSPacketSize = 64 // Rx (full-speed)
	descCDCACMDataTxFSPacketSize = 64 // Tx (full-speed)

	// CDC-ACM Endpoint Configurations for High-Speed Device

	descCDCACMRxFIFOHSSize = 512 << 2
	descCDCACMTxFIFOHSSize = 512 << 2

	descCDCACMStatusHSInterval   = 5   // Status
	descCDCACMStatusHSPacketSize = 64  //  (high-speed)
	descCDCACMDataRxHSPacketSize = 512 // Rx (high-speed)
	descCDCACMDataTxHSPacketSize = 512 // Tx (high-speed)
)

// Constants for USB HID (keyboard, mouse, joystick) device classes.
const (

	// HID Control Endpoint 0 Configuration and Buffer

	descHIDCxCount = 64 // Control buffer is always 64 bytes

	// HID Serial Configuration and Buffers

	descHIDSerialTxTimeoutMs = 50 // millisec
	descHIDSerialTxSyncUs    = 75 // microsec

	// HID Keyboard Configuration and Buffers

	descHIDKeyboardTxTimeoutMs = 50 // millisec

	// HID Mouse Configuration and Buffers

	descHIDMouseTxTimeoutMs = 30 // millisec

	// HID Joystick Configuration and Buffers

	descHIDJoystickTxTimeoutMs = 30 // millisec

	// HID Endpoint Configurations for Full-Speed Device

	descHIDSerialRxFSInterval   = 2 // Serial Rx
	descHIDSerialRxFSPacketSize = 8 //  (full-speed)

	descHIDSerialTxFSInterval   = 1  // Serial Tx
	descHIDSerialTxFSPacketSize = 16 //  (full-speed)

	descHIDKeyboardTxFSInterval   = 4 // Keyboard
	descHIDKeyboardTxFSPacketSize = 8 //  (full-speed)

	descHIDMediaKeyTxFSInterval   = 4 // Keyboard Media Keys
	descHIDMediaKeyTxFSPacketSize = 8 //  (full-speed)

	descHIDMouseTxFSInterval   = 4 // Mouse
	descHIDMouseTxFSPacketSize = 8 //  (full-speed)

	descHIDJoystickTxFSInterval   = 4  // Joystick
	descHIDJoystickTxFSPacketSize = 12 //  (full-speed)

	// HID Endpoint Configurations for High-Speed Device

	descHIDSerialRxHSInterval   = 2  // Serial
	descHIDSerialRxHSPacketSize = 32 //  (high-speed)

	descHIDSerialTxHSInterval   = 1  // Serial Tx
	descHIDSerialTxHSPacketSize = 64 //  (high-speed)

	descHIDKeyboardTxHSInterval   = 1 // Keyboard
	descHIDKeyboardTxHSPacketSize = 8 //  (high-speed)

	descHIDMediaKeyTxHSInterval   = 4 // Keyboard Media Keys
	descHIDMediaKeyTxHSPacketSize = 8 //  (high-speed)

	descHIDMouseTxHSInterval   = 1 // Mouse
	descHIDMouseTxHSPacketSize = 8 //  (high-speed)

	descHIDJoystickTxHSInterval   = 2  // Joystick
	descHIDJoystickTxHSPacketSize = 12 //  (high-speed)
)

type descCDCACMTerminal int

const (
	descCDCACMTerminalDisconnected descCDCACMTerminal = iota
	descCDCACMTerminalConnected
	descCDCACMTerminalReady
)

// descCDCACM0EP is the buffer of all endpoint state/information objects for the
// default CDC-ACM (single) device class configuration (index 1).
//
// Endpoints are mapped to elements of the buffer using the "logical index" of
// each endpoint address. This uses the first bit as the endpoint direction --
// where Rx=0, Tx=1 -- followed by all bits in the endpoint number.
// In other words, Rx endpoints are the even-numbered indices, Tx endpoints are
// the odd-numbered indices.
// Or, expressed arithmetically, Rx endpoint N = 2N, Tx endpoint N = 2N + 1.
//go:align 32
var descCDCACM0EP [descCDCACMEndpointCount << 1]dhwEndpoint

// descCDCACM0Cx is the control transfer buffer for data transmitted or received
// on control endpoint 0 of the default CDC-ACM (single) device class
// configuration (index 1).
//go:align 32
var descCDCACM0Cx [descCDCACMCxCount]uint8

// descCDCACM0Dx is the transmit (Tx) buffer of descriptor data on endpoint 0
// for the default CDC-ACM (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Dx [descCDCACMConfigSize]uint8

// descCDCACM0Rx is the receive (Rx) buffer of bulk OUT endpoint data for the
// default CDC-ACM (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Rx [descCDCACMRxFIFOSize]uint8

// descCDCACM0Tx is the transmit (Tx) buffer of bulk IN endpoint data for the
// default CDC-ACM (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Tx [descCDCACMTxFIFOSize]uint8

// descCDCACM0Rq is the receive (Rx) ring buffer used to serialize data received
// from the bulk data OUT endpoint to the UART interface for the default CDC-ACM
// (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Rq [dhwDataQueueSize]uint8

// descCDCACM0Tq is the transmit (Tx) ring buffer used to serialize data written
// from the UART interface to the bulk data IN endpoint for the default CDC-ACM
// (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Tq [dhwDataQueueSize]uint8

// descCDCACM0Ts is the transfer ring buffer used to schedule transmit (Tx) data
// transfers on the bulk IN endpoint for the default CDC-ACM (single) device
// class configuration (index 1).
//go:align 32
//var descCDCACM0Xq [dhwTransferQueueSize]*dhwTransfer

// descCDCACMClassData holds the buffers and control states for all CDC-ACM
// (single) device class configurations, ordered by index (offset by -1), for
// STM32 targets only.
//
// Instances of this type (elements of descCDCACMData) are embedded in elements
// of the common/target-agnostic CDC-ACM class configurations (descCDCACM).
// Methods defined on this type implement target-specific functionality, and
// some of these methods are required by the common device controller driver.
// Thus, this type functions as a hardware abstraction layer (HAL).
type descCDCACMClassData struct {

	// CDC-ACM Status Buffers

	ep *[descCDCACMEndpointCount << 1]dhwEndpoint // endpoint buffer

	// CDC-ACM Control Buffers

	cx *[descCDCACMCxCount]uint8    // control endpoint 0 Rx/Tx transfer buffer
	dx *[descCDCACMConfigSize]uint8 // control endpoint 0 Tx (IN) descriptor transfer buffer

	// CDC-ACM Data Buffers

	rx *[descCDCACMRxFIFOSize]uint8 // bulk data endpoint Rx (OUT) transfer buffer
	tx *[descCDCACMTxFIFOSize]uint8 // bulk data endpoint Tx (IN) transfer buffer

	rq dhwDataQueue // bulk data endpoint Rx (OUT) to UART serialization queue
	tq dhwDataQueue // bulk data endpoint Tx (IN) from UART serialization queue

	//xq dhwTransferQueue // bulk data endpoint Tx (IN) transfer queue

	rxSize uint16
	txSize uint16

	state descCDCACMTerminal
}

// descCDCACMData holds statically-allocated instances for each of the target-
// specific (STM32) CDC-ACM (single) device class configurations' control & data
// structures, ordered by configuration index (offset by -1). Each element is
// embedded in a corresponding element of descCDCACM.
//go:align 32
var descCDCACMData = [dcdCount]descCDCACMClassData{

	{ // -- CDC-ACM (single) Class Configuration Index 1 --

		// CDC-ACM Status Buffers

		ep: &descCDCACM0EP, // endpoint buffer

		// CDC-ACM Control Buffers

		cx: &descCDCACM0Cx, // control endpoint 0 Rx/Tx transfer buffer
		dx: &descCDCACM0Dx, // control endpoint 0 Tx (IN) descriptor transfer buffer

		// CDC-ACM Data Buffers

		rx: &descCDCACM0Rx, // bulk data endpoint Rx (OUT) transfer buffer
		tx: &descCDCACM0Tx, // bulk data endpoint Tx (IN) transfer buffer

		rq: dhwDataQueue{ // bulk data endpoint Rx (OUT) to UART serialization queue
			fifo: &descCDCACM0Rq,
			head: &volatile.Register32{},
			tail: &volatile.Register32{},
		},
		tq: dhwDataQueue{ // bulk data endpoint Tx (IN) from UART serialization queue
			fifo: &descCDCACM0Tq,
			head: &volatile.Register32{},
			tail: &volatile.Register32{},
		},

		//xq: dhwTransferQueue{ // bulk data endpoint Tx (IN) transfer queue
		//	fifo: &descCDCACM0Ts,
		//	head: &volatile.Register32{},
		//	tail: &volatile.Register32{},
		//},

		rxSize: descCDCACMDataRxPacketSize,
		txSize: descCDCACMDataTxPacketSize,
	},
}

// descHID0EP is the buffer of all endpoint state/information objects for the
// default HID device class configuration (index 1).
//
// Endpoints are mapped to elements of the buffer using the "logical index" of
// each endpoint address. This uses the first bit as the endpoint direction --
// where Rx=0, Tx=1 -- followed by all bits in the endpoint number.
// In other  words, Rx endpoints are the even-numbered indices, Tx endpoints are
// the odd-numbered indices.
// Or, expressed arithmetically, Rx endpoint N = 2N, Tx endpoint N = 2N + 1.
//go:align 32
var descHID0EP [descHIDEndpointCount << 1]dhwEndpoint

// descHID0Cx is the control transfer buffer for data transmitted or received on
// control endpoint 0 of the default HID device class configuration (index 1).
//go:align 32
var descHID0Cx [descHIDCxCount]uint8

// descHID0Dx is the transmit (Tx) buffer of descriptor data on endpoint 0 for
// the default HID device class configuration (index 1).
//go:align 32
var descHID0Dx [descHIDConfigSize]uint8

// descHID0KeyboardTp is the keyboard HID report transmit (Tx) transfer buffer
// for the default HID device class configuration (index 1).
//go:align 32
var descHID0KeyboardTp [descHIDKeyboardTxPacketSize]uint8

var descHID0KeyboardTxKey [hidKeyboardKeyCount]uint8
var descHID0KeyboardTxCon [hidKeyboardConCount]uint16
var descHID0KeyboardTxSys [hidKeyboardSysCount]uint8

// descHID0Keyboard is the Keyboard instance with which the user may interact
// when using the default HID device class configuration (index 1).
//go:align 32
var descHID0Keyboard = Keyboard{
	key: &descHID0KeyboardTxKey,
	con: &descHID0KeyboardTxCon,
	sys: &descHID0KeyboardTxSys,
}

// descHIDClassData holds the buffers and control states for all of the HID
// device class configurations, ordered by index (offset by -1), for STM32
// targets only.
//
// Instances of this type (elements of descHIDData) are embedded in elements
// of the common/target-agnostic HID class configurations (descHID).
// Methods defined on this type implement target-specific functionality, and
// some of these methods are required by the common device controller driver.
// Thus, this type functions as a hardware abstraction layer (HAL).
type descHIDClassData struct {

	// HID Status Buffers

	ep *[descHIDEndpointCount << 1]dhwEndpoint // endpoint buffer

	// HID Control Buffers

	cx *[descHIDCxCount]uint8    // control endpoint 0 Rx/Tx transfer buffer
	dx *[descHIDConfigSize]uint8 // control endpoint 0 Tx (IN) descriptor transfer buffer

	// HID Serial Buffers

	rxSerialSize uint16
	txSerialSize uint16

	// HID Keyboard Buffers

	tpKeyboard *[descHIDKeyboardTxPacketSize]uint8 // interrupt endpoint keyboard Tx (IN) HID report bbuffer

	txKeyboardSize uint16

	// HID Mouse Buffers

	txMouseSize uint16

	// HID Joystick Buffers

	txJoystickSize uint16

	// HID Device Instances

	//serial *Serial
	keyboard *Keyboard
	//mouse *Mouse
	//joystick *Joystick
}

// descHIDData holds statically-allocated instances for each of the target-
// specific (STM32) HID device class configurations' control & data structures,
// ordered by configuration index (offset by -1). Each element is embedded in a
// corresponding element of descHID.
//go:align 32
var descHIDData = [dcdCount]descHIDClassData{

	{ // -- HID Class Configuration Index 1 --

		// HID Status Buffers

		ep: &descHID0EP, // endpoint buffer

		// HID Control Buffers

		cx: &descHID0Cx, // control endpoint 0 Rx/Tx transfer buffer
		dx: &descHID0Dx, // control endpoint 0 Tx (IN) descriptor transfer buffer

		// HID Serial Buffers

		rxSerialSize: descHIDSerialRxPacketSize,
		txSerialSize: descHIDSerialTxPacketSize,

		// HID Keyboard Buffers

		tpKeyboard: &descHID0KeyboardTp, // interrupt endpoint keyboard Tx (IN) HID report bbuffer

		txKeyboardSize: descHIDKeyboardTxPacketSize,

		// HID Mouse Buffers

		txMouseSize: descHIDMouseTxPacketSize,

		// HID Joystick Buffers

		txJoystickSize: descHIDJoystickTxPacketSize,

		// HID Device Instances

		//serial: &descHID0Serial,
		keyboard: &descHID0Keyboard,
		//mouse: &descHID0Mouse,
		//joystick: &descHID0Joystick,
	},
}
