//go:build mimxrt1062
// +build mimxrt1062

package usb

// descCPUFrequencyHz defines the target CPU frequency (Hz).
const descCPUFrequencyHz = 600000000

// descCoreCount defines the number of USB PHY cores available on this platform,
// independent of the number of cores which shall be configured as TinyGo USB
// host/device controller instances.
const descCoreCount = 2

// General USB device identification constants.
const (
	descCommonVendorID  = 0x16C0
	descCommonProductID = 0x0483
	descCommonReleaseID = 0x0101 // BCD (1.1)

	descCommonLanguage     = descLanguageEnglish
	descCommonManufacturer = "TinyGo"
	descCommonProduct      = "USB"
	descCommonSerialNumber = "00000"
)
