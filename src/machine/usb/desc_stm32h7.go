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
)

// Constants for USB HID (keyboard, mouse, joystick) device classes.
const (

	// USB bus configuration attributes

	descHIDMaxPowerMa = 100 // Maximum current (mA) requested from host
)
