// +build stm32h7

package usb

// descCoreCount defines the number of USB PHY cores available on this platform,
// independent of the number of cores which shall be configured as TinyGo USB
// host/device controller instances.
const descCoreCount = 2

// Constants for USB CDC-ACM device classes.
const (

	// USB bus configuration attributes

	descCDCACMMaxPowerMa = 100 // Maximum current (mA) requested from host

	// Default (High-Speed) CDC-ACM Endpoint Configurations

	descCDCACMRxFIFOSize = descCDCACMRxFIFOHSSize
	descCDCACMTxFIFOSize = descCDCACMTxFIFOHSSize

	descCDCACMStatusInterval   = descCDCACMStatusHSInterval   // Status
	descCDCACMStatusPacketSize = descCDCACMStatusHSPacketSize //  (Interupt IN)
	descCDCACMDataRxPacketSize = descCDCACMDataRxHSPacketSize // Rx (Bulk OUT)
	descCDCACMDataTxPacketSize = descCDCACMDataTxHSPacketSize // Tx (Bulk IN)
)

// Constants for USB HID (keyboard, mouse, joystick) device classes.
const (

	// USB bus configuration attributes

	descHIDMaxPowerMa = 100 // Maximum current (mA) requested from host

	// Default (High-Speed) HID Endpoint Configurations

	descHIDSerialRxInterval   = descHIDSerialRxHSInterval   // Serial Rx
	descHIDSerialRxPacketSize = descHIDSerialRxHSPacketSize //  (Interrupt OUT)

	descHIDSerialTxInterval   = descHIDSerialTxHSInterval   // Serial Tx
	descHIDSerialTxPacketSize = descHIDSerialTxHSPacketSize //  (Interrupt IN)

	descHIDKeyboardTxInterval   = descHIDKeyboardTxHSInterval   // Keyboard
	descHIDKeyboardTxPacketSize = descHIDKeyboardTxHSPacketSize //  (Interrupt IN)

	descHIDMediaKeyTxInterval   = descHIDMediaKeyTxHSInterval   // Keyboard Media Keys
	descHIDMediaKeyTxPacketSize = descHIDMediaKeyTxHSPacketSize //  (Interrupt IN)

	descHIDMouseTxInterval   = descHIDMouseTxHSInterval   // Mouse
	descHIDMouseTxPacketSize = descHIDMouseTxHSPacketSize //  (Interrupt IN)

	descHIDJoystickTxInterval   = descHIDJoystickTxHSInterval   // Joystick
	descHIDJoystickTxPacketSize = descHIDJoystickTxHSPacketSize //  (Interrupt IN)
)
