//go:build usb.hid && mimxrt1062
// +build usb.hid,mimxrt1062

package usb

// descHIDCount defines the number of USB cores that may be configured as a
// composite (keyboard + mouse + joystick) human interface device (HID).
const descHIDCount = 1

// Constants for USB HID (keyboard, mouse, joystick) device classes.
const (

	// USB bus configuration attributes

	descHIDMaxPowerMa = 100 // Maximum current (mA) requested from host

	// HID Control Buffers

	descHIDQHCount = 2 * (descHIDEndpointCount + 1)
	descHIDCxCount = 8
	descHIDSxSize  = 8

	// HID Serial Buffers

	descHIDSerialRDCount = 8
	descHIDSerialRxSize  = descHIDSerialRxPacketSize
	descHIDSerialRxCount = descHIDSerialRxSize * descHIDSerialRDCount

	descHIDSerialTDCount = 12
	descHIDSerialTxSize  = descHIDSerialTxPacketSize
	descHIDSerialTxCount = descHIDSerialTxSize * descHIDSerialTDCount

	descHIDSerialTxTimeoutMs = 50 // millisec
	descHIDSerialTxSyncUs    = 75 // microsec

	// HID Keyboard Buffers

	descHIDKeyboardTDCount = 12
	descHIDKeyboardTxSize  = 4 * descHIDKeyboardTxPacketSize
	descHIDKeyboardTxCount = descHIDKeyboardTxSize * descHIDKeyboardTDCount

	descHIDKeyboardTxTimeoutMs = 50 // millisec

	// HID Mouse Buffers

	descHIDMouseTDCount = 4
	descHIDMouseTxSize  = 4 * descHIDMouseTxPacketSize
	descHIDMouseTxCount = descHIDMouseTxSize * descHIDMouseTDCount

	descHIDMouseTxTimeoutMs = 30 // millisec

	// HID Joystick Buffers

	descHIDJoystickTDCount = 4
	descHIDJoystickTxSize  = 4 * descHIDJoystickTxPacketSize
	descHIDJoystickTxCount = descHIDJoystickTxSize * descHIDJoystickTDCount

	descHIDJoystickTxTimeoutMs = 30 // millisec

	// Default HID Endpoint Configurations (High-Speed)

	descHIDSerialRxInterval   = descHIDSerialRxHSInterval   // Serial Rx
	descHIDSerialRxPacketSize = descHIDSerialRxHSPacketSize //

	descHIDSerialTxInterval   = descHIDSerialTxHSInterval   // Serial Tx
	descHIDSerialTxPacketSize = descHIDSerialTxHSPacketSize //

	descHIDKeyboardTxInterval   = descHIDKeyboardTxHSInterval   // Keyboard
	descHIDKeyboardTxPacketSize = descHIDKeyboardTxHSPacketSize //

	descHIDMediaKeyTxInterval   = descHIDMediaKeyTxHSInterval   // Keyboard Media Keys
	descHIDMediaKeyTxPacketSize = descHIDMediaKeyTxHSPacketSize //

	descHIDMouseTxInterval   = descHIDMouseTxHSInterval   // Mouse
	descHIDMouseTxPacketSize = descHIDMouseTxHSPacketSize //

	descHIDJoystickTxInterval   = descHIDJoystickTxHSInterval   // Joystick
	descHIDJoystickTxPacketSize = descHIDJoystickTxHSPacketSize //

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

// descHID0QH is an array of endpoint queue heads, which is where all transfers
// for a given endpoint are managed, for the default HID device class
// configuration (index 1).
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
var descHID0QH [descHIDQHCount]dhwEndpoint

// descHID0CD is the transfer descriptor for messages transmitted or received on
// the status/control endpoint 0 for the default HID device class configuration
// (index 1).
//
//go:align 32
var descHID0CD dhwTransfer

// descHID0Cx is the buffer for control/status data received on endpoint 0 of
// the default HID device class configuration (index 1).
//
//go:align 32
var descHID0Cx [descHIDCxCount]uint8

// descHID0AD is the transfer descriptor for ackowledgement (ACK) messages
// transmitted or received on the status/control endpoint 0 for the default HID
// device class configuration (index 1).
//
//go:align 32
var descHID0AD dhwTransfer

// descHID0Dx is the transmit (Tx) buffer of descriptor data on endpoint 0 for
// the default HID device class configuration (index 1).
//
//go:align 32
var descHID0Dx [descHIDConfigSize]uint8

// descHID0SerialRD is an array of transfer descriptors for serial Rx (OUT)
// transfers, which describe to the device controller the location and quantity
// of data being received for a given transfer, for the default HID device class
// configuration (index 1).
//
//go:align 32
var descHID0SerialRD [descHIDSerialRDCount]dhwTransfer

// descHID0SerialRx is the serial receive (Rx) transfer buffer for the default
// HID device class configuration (index 1).
//
//go:align 32
var descHID0SerialRx [descHIDSerialRxCount]uint8

// descHID0SerialTD is an array of transfer descriptors for serial Tx (IN)
// transfers, which describe to the device controller the location and quantity
// of data being transmitted for a given transfer, for the default HID device
// class configuration (index 1).
//
//go:align 32
var descHID0SerialTD [descHIDSerialTDCount]dhwTransfer

// descHID0SerialTx is the serial transmit (Tx) transfer buffer for the default
// HID device class configuration (index 1).
//
//go:align 32
var descHID0SerialTx [descHIDSerialTxCount]uint8

var (
	descHID0SerialRDIdx [descHIDSerialRDCount]uint16
	descHID0SerialRDQue [(descHIDSerialRDCount + 1)]uint16
)

// descHID0KeyboardTD is an array of transfer descriptors for keyboard Tx (IN)
// transfers, which describe to the device controller the location and quantity
// of data being transmitted for a given transfer, for the default HID device
// class configuration (index 1).
//
//go:align 32
var descHID0KeyboardTD [descHIDKeyboardTDCount]dhwTransfer

// descHID0KeyboardTx is the keyboard transmit (Tx) transfer buffer for the
// default HID device class configuration (index 1).
//
//go:align 32
var descHID0KeyboardTx [descHIDKeyboardTxCount]uint8

// descHID0KeyboardTp is the keyboard HID report transmit (Tx) transfer buffer
// for the default HID device class configuration (index 1).
//
//go:align 32
var descHID0KeyboardTp [descHIDKeyboardTxPacketSize]uint8

//go:align 32
var descHID0KeyboardTxKey [hidKeyboardKeyCount]uint8

//go:align 32
var descHID0KeyboardTxCon [hidKeyboardConCount]uint16

//go:align 32
var descHID0KeyboardTxSys [hidKeyboardSysCount]uint8

// descHID0MouseTD is an array of transfer descriptors for mouse Tx (IN)
// transfers, which describe to the device controller the location and quantity
// of data being transmitted for a given transfer, for the default HID device
// class configuration (index 1).
//
//go:align 32
var descHID0MouseTD [descHIDMouseTDCount]dhwTransfer

// descHID0MouseTx is the mouse transmit (Tx) transfer buffer for the default
// HID device class configuration (index 1).
//
//go:align 32
var descHID0MouseTx [descHIDMouseTxCount]uint8

// descHID0JoystickTD is an array of transfer descriptors for joystick Tx (IN)
// transfers, which describe to the device controller the location and quantity
// of data being transmitted for a given transfer, for the default HID device
// class configuration (index 1).
//
//go:align 32
var descHID0JoystickTD [descHIDJoystickTDCount]dhwTransfer

// descHID0JoystickTx is the joystick transmit (Tx) transfer buffer for the
// default HID device class configuration (index 1).
//
//go:align 32
var descHID0JoystickTx [descHIDJoystickTxCount]uint8

// descHID0Keyboard is the Keyboard instance with which the user may interact
// when using the default HID device class configuration (index 1).
//
//go:align 64
var descHID0Keyboard = Keyboard{
	key: &descHID0KeyboardTxKey,
	con: &descHID0KeyboardTxCon,
	sys: &descHID0KeyboardTxSys,
}

// descHIDClassData holds the buffers and control states for all of the HID
// device class configurations, ordered by index (offset by -1), for iMXRT1062
// targets only.
//
// Instances of this type (elements of descHIDData) are embedded in elements
// of the common/target-agnostic HID class configurations (descHID).
// Methods defined on this type implement target-specific functionality, and
// some of these methods are required by the common device controller driver.
// Thus, this type functions as a hardware abstraction layer (HAL).
type descHIDClassData struct {
	// HID Control Buffers

	qh *[descHIDQHCount]dhwEndpoint // endpoint queue heads

	cd *dhwTransfer              // control endpoint 0 Rx/Tx transfer descriptor
	cx *[descHIDCxCount]uint8    // control endpoint 0 Rx/Tx transfer buffer
	ad *dhwTransfer              // control endpoint 0 Rx/Tx ACK transfer descriptor
	dx *[descHIDConfigSize]uint8 // control endpoint 0 Tx (IN) descriptor transfer buffer

	// HID Serial Buffers

	rdSerial *[descHIDSerialRDCount]dhwTransfer // interrupt endpoint serial Rx (OUT) transfer descriptors
	rxSerial *[descHIDSerialRxCount]uint8       // interrupt endpoint serial Rx (OUT) transfer buffer
	tdSerial *[descHIDSerialTDCount]dhwTransfer // interrupt endpoint serial Tx (IN) transfer descriptors
	txSerial *[descHIDSerialTxCount]uint8       // interrupt endpoint serial Tx (IN) transfer buffer

	rxSerialIndex *[descHIDSerialRDCount]uint16
	rxSerialQueue *[(descHIDSerialRDCount + 1)]uint16

	rxSerialSize uint16
	txSerialSize uint16

	txSerialHead uint8
	txSerialFree uint16
	txSerialPrev bool

	rxSerialHead uint8
	rxSerialTail uint8
	rxSerialFree uint16

	// HID Keyboard Buffers

	tdKeyboard *[descHIDKeyboardTDCount]dhwTransfer // interrupt endpoint keyboard Tx (IN) transfer descriptors
	txKeyboard *[descHIDKeyboardTxCount]uint8       // interrupt endpoint keyboard Tx (IN) transfer buffer
	tpKeyboard *[descHIDKeyboardTxPacketSize]uint8  // interrupt endpoint keyboard Tx (IN) HID report bbuffer

	txKeyboardSize uint16

	txKeyboardHead uint8
	txKeyboardPrev bool

	// HID Mouse Buffers

	tdMouse *[descHIDMouseTDCount]dhwTransfer // interrupt endpoint mouse Tx (IN) transfer descriptors
	txMouse *[descHIDMouseTxCount]uint8       // interrupt endpoint mouse Tx (IN) transfer buffer

	txMouseSize uint16

	txMouseHead uint8
	txMousePrev bool

	// HID Joystick Buffers

	tdJoystick *[descHIDJoystickTDCount]dhwTransfer // interrupt endpoint joystick Tx (IN) transfer descriptors
	txJoystick *[descHIDJoystickTxCount]uint8       // interrupt endpoint joystick Tx (IN) transfer buffer

	txJoystickSize uint16

	txJoystickHead uint8
	txJoystickPrev bool

	// HID Device Instances

	// serial *Serial
	keyboard *Keyboard
	// mouse *Mouse
	// joystick *Joystick
}

// descHIDData holds statically-allocated instances for each of the target-
// specific (iMXRT1062) HID device class configurations' control and data
// structures, ordered by configuration index (offset by -1). Each element is
// embedded in a corresponding element of descHID.
//
//go:align 64
var descHIDData = [dcdCount]descHIDClassData{
	{ // -- HID Class Configuration Index 1 --

		// HID Control Buffers

		qh: &descHID0QH,

		cd: &descHID0CD,
		cx: &descHID0Cx,
		ad: &descHID0AD,
		dx: &descHID0Dx,

		// HID Serial Buffers

		rdSerial: &descHID0SerialRD,
		rxSerial: &descHID0SerialRx,
		tdSerial: &descHID0SerialTD,
		txSerial: &descHID0SerialTx,

		rxSerialIndex: &descHID0SerialRDIdx,
		rxSerialQueue: &descHID0SerialRDQue,

		rxSerialSize: descHIDSerialRxPacketSize,
		txSerialSize: descHIDSerialTxPacketSize,

		// HID Keyboard Buffers

		tdKeyboard: &descHID0KeyboardTD,
		txKeyboard: &descHID0KeyboardTx,
		tpKeyboard: &descHID0KeyboardTp,

		txKeyboardSize: descHIDKeyboardTxPacketSize,

		// HID Mouse Buffers

		tdMouse: &descHID0MouseTD,
		txMouse: &descHID0MouseTx,

		txMouseSize: descHIDMouseTxPacketSize,

		// HID Joystick Buffers

		tdJoystick: &descHID0JoystickTD,
		txJoystick: &descHID0JoystickTx,

		txJoystickSize: descHIDJoystickTxPacketSize,

		// HID Device Instances

		// serial: &descHID0Serial,
		keyboard: &descHID0Keyboard,
		// mouse: &descHID0Mouse,
		// joystick: &descHID0Joystick,
	},
}
