package usb

import (
	"errors"
)

var (
	ErrInvalidPort = errors.New("invalid USB port")
)

type (
	UARTConfig struct {
		BaudRate uint32
	}

	// UART represents a virtual serial (UART) device emulation using the USB
	// CDC-ACM device class driver.
	UART struct {
		port        uint8                   // USB port (core index, e.g., 0-1)
		desc        *ConfigDeviceDescriptor // User-provided USB device descriptor information
		classHandle *deviceClass            // USB device class handle
		acmHandle   *deviceCDCACM           // USB CDC-ACM device class handle
	}
)

func (uart *UART) SetPort(port uint8) {
	if port >= ConfigPortCount || port >= configDeviceCount ||
		port >= configDeviceCDCACMCount {
		return // invalid port for USB CDC-ACM device class
	}
	// TODO: if changed, may need to reset port controller and tell host to
	//       re-enumerate devices
	uart.port = port
}

func (uart *UART) SetDeviceDescriptor(desc *ConfigDeviceDescriptor) {
	if nil == desc {
		return // invalid device descriptor information
	}
	// TODO: if changed, may need to reset port controller and tell host to
	//       re-enumerate devices
	uart.desc = desc
}

func (uart *UART) Configure(config UARTConfig) error {

	if uart.port >= ConfigPortCount || uart.port >= configDeviceCount ||
		uart.port >= configDeviceCDCACMCount {
		return ErrInvalidPort
	}

	// change baud rate from default, if provided
	if config.BaudRate != 0 {
		configDeviceCDCACM[uart.port].lineCodingBaudRate = config.BaudRate
	}

	// verify we have a free USB port and take ownership of it
	port, status := initPort(uart.port, modeDevice)
	if !status.OK() {
		return status
	}

	// apply the CDC-ACM configuration to our port
	uart.acmHandle, uart.classHandle = port.initCDCACM(uart)

	// enable USB device mode functionality, which allows a host to enumerate us
	status = port.device.controller.enable(true)
	if !status.OK() {
		return status
	}

	return nil
}

func (uart *UART) deviceEvent(ev deviceEventID, param interface{}) status {
	return statusSuccess
}

func (uart *UART) classEvent(ev uint32, param interface{}) status {
	return statusSuccess
}
