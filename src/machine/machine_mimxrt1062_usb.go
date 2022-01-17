//go:build mimxrt1062 && serial.usb
// +build mimxrt1062,serial.usb

package machine

import (
	"machine/usb"
)

// usbUART wraps the usb.UART so that Configure() can be overridden
// to accept machine.UARTConfig as its argument
type usbUART struct {
	*usb.UART
}

func (u *usbUART) Configure(cfg UARTConfig) {
	u.UART.Configure(usb.UARTConfig{})
}

var USB = &usbUART{&usb.UART{Port: 0}}
