// +build mimxrt1062

package usb

const configCPUFrequencyHz = 600000000

// The following constants must be defined for USB 2.0 host/device support.
//
// However, some or all of these constants may be unused in the core driver
// code, depending on which components are allocated by the USB driver.
//
// These values are __PLATFORM-SPECIFIC__, and they serve two primary roles:
//   1. Determine which components to allocate from the USB driver.
//   2. Configure driver attributes NOT defined by the USB 2.0 specification.
//
const (
	// configDeviceCount defines the number of USB device-mode ports available.
	configDeviceCount = configDeviceCDCACMCount

	// configHostCount defines the number of USB host-mode ports available.
	configHostCount = 0

	// configDeviceSelfPowered defines whether the device is self-powered (1) or
	// not (0).
	configDeviceSelfPowered = 1

	// configDeviceCDCACMCount defines the number of USB CDC-ACM interfaces
	// to initialize; must be less-than or equal to configDeviceCount.
	configDeviceCDCACMCount = 1

	// configInterruptPriority defines the priority number for USB interrupts.
	configInterruptPriority = 3

	// configInterruptQueueSize defines the number of interrupts to retain in the
	// queue of runtime processes.
	configInterruptQueueSize = 8

	// configDeviceMaxEndpoints defines the maximum number of endpoints supported.
	configDeviceMaxEndpoints = 4

	// configDeviceControllerMaxDTD defines the maximum number of DTD supported.
	configDeviceControllerMaxDTD = 16

	// configDeviceControllerQHAlign defines the memory alignment of QH buffer.
	// Ensure this agrees with the go:align pragma on deviceControllerQHBuffer.
	configDeviceControllerQHAlign = 2048

	// configDeviceControllerDTDAlign defines the memory alignment of DTD buffer.
	// Ensure this agrees with the go:align pragma on deviceControllerDTDBuffer.
	configDeviceControllerDTDAlign = 32

	// configDeviceControllerMaxPacketSize defines the maximum packet size for
	// communication with an endpoint. The maximum per USB 2.0 spec is 64 bytes,
	// although a platform may restrict this to something lower if needed.
	configDeviceControllerMaxPacketSize = 64

	// configDeviceControllerMaxPrimeAttempts defines the maximum number of
	// attempts to prime and endpoint for transfer. If attempts exceeds this
	// value, then the endpoint status has been reset.
	configDeviceControllerMaxPrimeAttempts = 10000000
)

// If USB CDC-ACM device support is required, the following array must be
// initialized with each device's CDC-ACM configuration.
var (
	configDeviceCDCACM = [configDeviceCDCACMCount]deviceCDCACMConfig{
		{ // USB CDC-ACM [0]
			interfaceSpeed: specSpeedFull, // USB full-speed (12 Mbit/s)
			// Serial line configuration
			lineCodingSize:       7,      // Size of line-coding message
			lineCodingBaudRate:   115200, // Data terminal rate
			lineCodingCharFormat: 0,      // Character format
			lineCodingParityType: 0,      // Parity type
			lineCodingDataBits:   8,      // Data word size
			// Communication/control interface
			commInterfaceIndex:        0, // communication/control interface index
			commInterruptInEndpoint:   1, // interrupt input endpoint index (address)
			commInterruptInPacketSize: configDeviceCDCACMFSInterruptInPacketSize,
			commInterruptInInterval:   configDeviceCDCACMFSInterruptInInterval,
			// Data interface
			dataInterfaceIndex:    1, // data interface index
			dataBulkInEndpoint:    2, // bulk input endpoint index (address)
			dataBulkInPacketSize:  configDeviceCDCACMFSBulkInPacketSize,
			dataBulkOutEndpoint:   3, // bulk output endpoint index (address)
			dataBulkOutPacketSize: configDeviceCDCACMFSBulkOutPacketSize,
		},
	}
)

// The following additional constants are not required by the USB driver but are
// used by the usb package on this platform.
const (
	// USB CDC-ACM high-speed (480 Mbit/s) packet size
	configDeviceCDCACMHSInterruptInPacketSize = 16
	configDeviceCDCACMHSInterruptInInterval   = 7 // 2^(7-1)/8 = 8ms
	configDeviceCDCACMHSBulkInPacketSize      = 512
	configDeviceCDCACMHSBulkOutPacketSize     = 512

	// USB CDC-ACM full-speed (12 Mbit/s) packet size
	configDeviceCDCACMFSInterruptInPacketSize = 16
	configDeviceCDCACMFSInterruptInInterval   = 8 // 2^(8-1)/8 = 16ms
	configDeviceCDCACMFSBulkInPacketSize      = 64
	configDeviceCDCACMFSBulkOutPacketSize     = 64
)
