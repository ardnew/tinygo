// +build portenta_h7

package machine

// Peripheral and pin mappings for Arduino Portenta H7:
//   https://store.arduino.cc/usa/portenta-h7
//
// This only defines peripherals/pins available on the MKR header. For access
// to pins on the 80-pin high-density connectors, you must use the CPU port pin
// identifiers (e.g., PH15) defined in machine_stm32.go.

import (
	"machine/usb"
)

// GPIO Pins
const (
	D0  = PH15
	D1  = PK01
	D2  = PJ11
	D3  = PG07
	D4  = PC07
	D5  = PC06
	D6  = PA08
	D7  = PI00
	D8  = PC03
	D9  = PI01
	D10 = PC02
	D11 = PH08
	D12 = PH07
	D13 = PA10
	D14 = PA09
)

// Analog pins
const (
	A0 = PA00 // C
	A1 = PA01 // C
	A2 = PC02 // C
	A3 = PC03 // C
	A4 = PC02 // ALT0
	A5 = PC03 // ALT0
	A6 = PA04
	A7 = PA06
)

// On-board RGB LED pins
const (
	LED  = LEDR
	LEDR = PK05
	LEDG = PK06
	LEDB = PK07
)

// USB full-speed pins (embedded PHY)
const (
	USB_FS_DP_PIN = PA12
	USB_FS_DM_PIN = PA11
)

// USB full-speed pins (external PHY)
const (
	USB_HS_DP_PIN   = PB15
	USB_HS_DM_PIN   = PB14
	USB_HS_ID_PIN   = PB12
	USB_HS_VBUS_PIN = PB13
)

// USB high-speed pins (external PHY, ULPI)
const (
	USB_HS_D0_PIN  = PA03
	USB_HS_D1_PIN  = PB00
	USB_HS_D2_PIN  = PB01
	USB_HS_D3_PIN  = PB10
	USB_HS_D4_PIN  = PB11
	USB_HS_D5_PIN  = PB12
	USB_HS_D6_PIN  = PB13
	USB_HS_D7_PIN  = PB05
	USB_HS_CLK_PIN = PA05
	USB_HS_STP_PIN = PC00
	USB_HS_NXT_PIN = PH04
	USB_HS_DIR_PIN = PI11
)

var UART0 = usb.UART{Port: 0} // Embedded USB HS PHY

// InitBoard enables and configures any on-board peripherals that must always
// be available. Should only be called after runtime initialization is complete.
func InitBoard() {

	switch CoreID {

	// Cortex-M7
	case Core0:
		// Turn off LEDG used by bootloader which may remain active after USB DFU
		// "download" completes. Also configure the other 2 LEDs for convenience.
		for _, p := range []Pin{LEDR, LEDG, LEDB} {
			p.Configure(PinConfig{Mode: PinOutput})
			p.High() // HIGH=OFF, LOW=ON
		}

		// Initialize USB device class driver
		initUSB(usb.HighSpeed)

	// Cortex-M4
	case Core1:
	}
}

func initUSB(speed usb.Speed) {

	port := UART0.Port

	switch port {
	// USB OTG port 1 supports both high- and full-speed modes
	case 0:
		switch speed {
		// Configured as either full-speed (FS) or low-speed (LS)
		case usb.LowSpeed, usb.FullSpeed:
			USB_HS_DP_PIN.Configure(PinConfig{Mode: PinUSB1FSDP})
			USB_HS_DM_PIN.Configure(PinConfig{Mode: PinUSB1FSDM})
			USB_HS_ID_PIN.Configure(PinConfig{Mode: PinUSB1FSID})
			USB_HS_VBUS_PIN.Configure(PinConfig{Mode: PinUSB1FSVBUS})
		// Configured as either high-speed (HS) or super-speed (SS [unsupported])
		case usb.HighSpeed, usb.SuperSpeed, usb.DualSuperSpeed:
			for _, p := range []Pin{
				USB_HS_D0_PIN, USB_HS_D1_PIN, USB_HS_D2_PIN,
				USB_HS_D3_PIN, USB_HS_D4_PIN, USB_HS_D5_PIN,
				USB_HS_D6_PIN, USB_HS_D7_PIN, USB_HS_CLK_PIN,
				USB_HS_STP_PIN, USB_HS_NXT_PIN, USB_HS_DIR_PIN,
			} {
				p.Configure(PinConfig{Mode: PinUSB1HSULPI})
			}
		}
	// USB OTG port 2 supports full-speed (FS) mode ONLY
	case 1:
		USB_FS_DP_PIN.Configure(PinConfig{Mode: PinUSB2FSDP})
		USB_FS_DM_PIN.Configure(PinConfig{Mode: PinUSB2FSDM})
	}

	// Configure USB as a virtual UART interface using CDC-ACM device class.
	UART0.Configure(usb.UARTConfig{BusSpeed: speed})

	// Configure USB as a keyboard/joystick/mouse/serial using HID device class.
	// HID0.Configure(usb.HIDConfig{})
}
