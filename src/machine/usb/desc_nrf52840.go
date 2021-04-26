// +build nrf52840

package usb

// descCPUFrequencyHz defines the target CPU frequency (Hz).
const descCPUFrequencyHz = 64000000 // 64 MHz

// General USB device identification constants.
const (
	descCommonVendorID  = 0x1915
	descCommonProductID = 0x719E
	descCommonReleaseID = 0x0101 // BCD (1.1)

	descCommonLanguage     = descLanguageEnglish
	descCommonManufacturer = "Nordic Semiconductor"
	descCommonProduct      = "TinyGo USB"
	descCommonSerialNumber = "1"
)

// Constants for USB CDC-ACM device classes.
const (
	// descCDCACMCount defines the number of USB cores that will be configured as
	// CDC-ACM (single) devices.
	descCDCACMCount = 1

	descCDCACMQHCount = 2 * (descCDCACMEndpointCount + 1)
	descCDCACMRDCount = 2 * descCDCACMEndpointCount
	descCDCACMTDCount = descCDCACMEndpointCount
	descCDCACMRxCount = descCDCACMRxSize * descCDCACMRDCount
	descCDCACMTxCount = descCDCACMTxSize * descCDCACMTDCount
	descCDCACMCxCount = 8

	descCDCACMMaxPower = 50 // 100 mA

	descCDCACMTxTimeoutMs = 120 // millisec
	descCDCACMTxSyncUs    = 75  // microsec

	descCDCACMStatusPacketSize = 16
	descCDCACMDataRxPacketSize = descCDCACMDataRxFSPacketSize // full-speed
	descCDCACMDataTxPacketSize = descCDCACMDataTxFSPacketSize // full-speed
	descCDCACMRxSize           = descCDCACMDataRxPacketSize
	descCDCACMTxSize           = 4 * descCDCACMDataTxPacketSize

	// nRF52840 is full-speed (12 Mbit/sec) only
	descCDCACMDataRxFSPacketSize = 64 // full-speed
	descCDCACMDataTxFSPacketSize = 64 // full-speed
	// High-speed (480 Mbit/sec) not supported
	descCDCACMDataRxHSPacketSize = descCDCACMDataRxFSPacketSize // high-speed
	descCDCACMDataTxHSPacketSize = descCDCACMDataTxFSPacketSize // high-speed
)

// descCDCACM0QH is an array of endpoint queue heads, which is where all
// transfers for a given endpoint are managed, for the default CDC-ACM (single)
// device class configuration (index 1).
//go:align 32
var descCDCACM0QH [descCDCACMQHCount]dcdEndpoint

// descCDCACM0CD is the transfer descriptor for data messages transmitted or
// received on the status/control endpoint 0 for the default CDC-ACM (single)
// device class configuration (index 1).
//go:align 32
var descCDCACM0CD dcdTransfer

// descCDCACM0AD is the transfer descriptor for ackowledgement (ACK) messages
// transmitted or received on the status/control endpoint 0 for the default
// CDC-ACM (single) device class configuration (index 1).
//go:align 32
var descCDCACM0AD dcdTransfer

// descCDCACM0RD is an array of transfer descriptors for Rx (OUT) transfers,
// which describe to the device controller the location and quantity of data
// being received for a given transfer, for the default CDC-ACM (single) device
// class configuration (index 1).
//go:align 32
var descCDCACM0RD [descCDCACMRDCount]dcdTransfer

// descCDCACM0TD is an array of transfer descriptors for Tx (IN) transfers,
// which describe to the device controller the location and quantity of data
// being transmitted for a given transfer, for the default CDC-ACM (single)
// device class configuration (index 1).
//go:align 32
var descCDCACM0TD [descCDCACMTDCount]dcdTransfer

// descCDCACM0Cx is the buffer for control/status data received on endpoint 0 of
// the default CDC-ACM (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Cx [descCDCACMCxCount]uint8

// descCDCACM0Rx is the receive (Rx) buffer of data endpoints for the default
// CDC-ACM (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Rx [descCDCACMRxCount]uint8

// descCDCACM0Tx is the transmit (Tx) buffer of data endpoints for the default
// CDC-ACM (single) device class configuration (index 1).
//go:align 32
var descCDCACM0Tx [descCDCACMTxCount]uint8

// descCDCACM0Dx is the transmit (Tx) buffer of descriptor data for the default
// CDC-ACM (single) device class configuration (index 1).
var descCDCACM0Dx [descCDCACMConfigSize]uint8

var descCDCACM0RDNum [descCDCACMRDCount]uint16
var descCDCACM0RDIdx [descCDCACMRDCount]uint16
var descCDCACM0RDQue [descCDCACMRDCount + 1]uint16
