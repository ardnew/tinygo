package usb

const (
	// Communication Class
	deviceCDCCommClass = 0x02
	// Data Class
	deviceCDCDataClass = 0x0A

	// Communication Class SubClass Codes
	deviceCDCDirectLineControlModel         = 0x01
	deviceCDCAbstractControlModel           = 0x02
	deviceCDCTelephoneControlModel          = 0x03
	deviceCDCMultiChannelControlModel       = 0x04
	deviceCDCCAPIControlMopdel              = 0x05
	deviceCDCEthernetNetworkingControlModel = 0x06
	deviceCDCATMNetworkingControlModel      = 0x07
	deviceCDCWirelessHandsetControlModel    = 0x08
	deviceCDCDeviceManagement               = 0x09
	deviceCDCMobileDirectLineModel          = 0x0A
	deviceCDCOBEX                           = 0x0B
	deviceCDCEthernetEmulationModel         = 0x0C

	// Communication Class Protocol Codes
	deviceCDCNoClassSpecificProtocol   = 0x00 // also for Data Class Protocol Code
	deviceCDCAT250Protocol             = 0x01
	deviceCDCATPCCA101Protocol         = 0x02
	deviceCDCATPCCA101AnnexO           = 0x03
	deviceCDCATGSM707                  = 0x04
	deviceCDCAT3GPP27007               = 0x05
	deviceCDCATTIACDMA                 = 0x06
	deviceCDCEthernetEmulationProtocol = 0x07
	deviceCDCExternalProtocol          = 0xFE
	deviceCDCVendorSpecific            = 0xFF // also for Data Class Protocol Code

	// Data Class Protocol Codes
	deviceCDCPyhsicalInterfaceProtocol = 0x30
	deviceCDCHDLCProtocol              = 0x31
	deviceCDCTransparentProtocol       = 0x32
	deviceCDCManagementProtocol        = 0x50
	deviceCDCDataLinkQ931Protocol      = 0x51
	deviceCDCDataLinkQ921Protocol      = 0x52
	deviceCDCDataCompressionV42BIS     = 0x90
	deviceCDCEuroISDNProtocol          = 0x91
	deviceCDCRateAdaptionISDNV24       = 0x92
	deviceCDCCAPICommands              = 0x93
	deviceCDCHostBasedDriver           = 0xFD
	deviceCDCUnitFunctional            = 0xFE

	// Descriptor SubType in Communications Class Functional Descriptors
	deviceCDCHeaderFuncDesc             = 0x00
	deviceCDCCallManagementFuncDesc     = 0x01
	deviceCDCAbstractControlFuncDesc    = 0x02
	deviceCDCDirectLineFuncDesc         = 0x03
	deviceCDCTelephoneRingerFuncDesc    = 0x04
	deviceCDCTelephoneReportFuncDesc    = 0x05
	deviceCDCUnionFuncDesc              = 0x06
	deviceCDCCountrySelectFuncDesc      = 0x07
	deviceCDCTelephoneModesFuncDesc     = 0x08
	deviceCDCTerminalFuncDesc           = 0x09
	deviceCDCNetworkChannelFuncDesc     = 0x0A
	deviceCDCProtocolUnitFuncDesc       = 0x0B
	deviceCDCExtensionUnitFuncDesc      = 0x0C
	deviceCDCMultiChannelFuncDesc       = 0x0D
	deviceCDCCAPIControlFuncDesc        = 0x0E
	deviceCDCEthernetNetworkingFuncDesc = 0x0F
	deviceCDCATMNetworkingFuncDesc      = 0x10
	deviceCDCWirelessControlFuncDesc    = 0x11
	deviceCDCMobileDirectLineFuncDesc   = 0x12
	deviceCDCMDLMDetailFuncDesc         = 0x13
	deviceCDCDeviceManagementFuncDesc   = 0x14
	deviceCDCOBEXFuncDesc               = 0x15
	deviceCDCCommandSetFuncDesc         = 0x16
	deviceCDCCommandSetDetailFuncDesc   = 0x17
	deviceCDCTelephoneControlFuncDesc   = 0x18
	deviceCDCOBEXServiceIDFuncDesc      = 0x19
)
