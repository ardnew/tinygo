package usb

const (
	// USB speed (the value cannot be changed because EHCI QH use the value directly)
	specSpeedFull  = 0x00
	specSpeedLow   = 0x01
	specSpeedHigh  = 0x02
	specSpeedSuper = 0x04

	// USB standard descriptor endpoint type
	specEndpointControl     = 0x00
	specEndpointIsochronous = 0x01
	specEndpointBulk        = 0x02
	specEndpointInterrupt   = 0x03

	// USB standard descriptor transfer direction (cannot change the value because iTD use the value directly)
	specOut = 0
	specIn  = 1

	// USB standard descriptor length
	specDescriptorLengthDevice                         = 0x12
	specDescriptorLengthConfigure                      = 0x09
	specDescriptorLengthInterface                      = 0x09
	specDescriptorLengthEndpoint                       = 0x07
	specDescriptorLengthEndpointCompanion              = 0x06
	specDescriptorLengthDeviceQualitier                = 0x0A
	specDescriptorLengthOTGDescriptor                  = 5
	specDescriptorLengthBOSDescriptor                  = 5
	specDescriptorLengthDeviceCapabilityUSB20Extension = 0x07
	specDescriptorLengthDeviceCapabilitySuperspeed     = 0x0A

	// USB Device Capability Type Codes
	specDescriptorTypeDeviceCapabilityWireless       = 0x01
	specDescriptorTypeDeviceCapabilityUSB20Extension = 0x02
	specDescriptorTypeDeviceCapabilitySuperspeed     = 0x03

	// USB standard descriptor type
	specDescriptorTypeDevice                  = 0x01
	specDescriptorTypeConfigure               = 0x02
	specDescriptorTypeString                  = 0x03
	specDescriptorTypeInterface               = 0x04
	specDescriptorTypeEndpoint                = 0x05
	specDescriptorTypeDeviceQualitier         = 0x06
	specDescriptorTypeOtherSpeedConfiguration = 0x07
	specDescriptorTypeInterfaacePower         = 0x08
	specDescriptorTypeOTG                     = 0x09
	specDescriptorTypeInterfaceAssociation    = 0x0B
	specDescriptorTypeBOS                     = 0x0F
	specDescriptorTypeDeviceCapability        = 0x10

	specDescriptorTypeHID         = 0x21
	specDescriptorTypeHIDReport   = 0x22
	specDescriptorTypeHIDPhysical = 0x23

	specDescriptorTypeEndpointCompanion = 0x30

	// USB standard request type
	specRequestTypeDirMsk = 0x80
	specRequestTypeDirPos = 7
	specRequestTypeDirOut = 0x00
	specRequestTypeDirIn  = 0x80

	specRequestTypeTypeMsk      = 0x60
	specRequestTypeTypePos      = 5
	specRequestTypeTypeStandard = 0
	specRequestTypeTypeClass    = 0x20
	specRequestTypeTypeVendor   = 0x40

	specRequestTypeRecipientMsk       = 0x1F
	specRequestTypeRecipientPos       = 0
	specRequestTypeRecipientDevice    = 0x00
	specRequestTypeRecipientInterface = 0x01
	specRequestTypeRecipientEndpoint  = 0x02
	specRequestTypeRecipientOther     = 0x03

	// USB standard request
	specRequestStandardGetStatus        = 0x00
	specRequestStandardClearFeature     = 0x01
	specRequestStandardSetFeature       = 0x03
	specRequestStandardSetAddress       = 0x05
	specRequestStandardGetDescriptor    = 0x06
	specRequestStandardSetDescriptor    = 0x07
	specRequestStandardGetConfiguration = 0x08
	specRequestStandardSetConfiguration = 0x09
	specRequestStandardGetInterface     = 0x0A
	specRequestStandardSetInterface     = 0x0B
	specRequestStandardSynchFrame       = 0x0C

	// USB standard request GET Status
	specRequestStandardGetStatusDeviceSelfPoweredPos  = 0
	specRequestStandardGetStatusDeviceRemoteWakeupPos = 1

	specRequestStandardGetStatusEndpointHaltMsk = 0x01
	specRequestStandardGetStatusEndpointHaltPos = 0

	specRequestStandardGetStatusOTGStatusSelector = 0xF000

	// USB standard request CLEAR/SET feature
	specRequestStandardFeatureSelectorEndpointHalt       = 0
	specRequestStandardFeatureSelectorDeviceRemoteWakeup = 1
	specRequestStandardFeatureSelectorDeviceTestMode     = 2
	specRequestStandardFeatureSelectorBHNPEnable         = 3
	specRequestStandardFeatureSelectorAHNPSupport        = 4
	specRequestStandardFeatureSelectorAAltHNPSupport     = 5

	// USB standard descriptor configure bmAttributes
	specDescriptorConfigureAttributeD7Msk = 0x80
	specDescriptorConfigureAttributeD7Pos = 7

	specDescriptorConfigureAttributeSelfPoweredMsk = 0x40
	specDescriptorConfigureAttributeSelfPoweredPos = 6

	specDescriptorConfigureAttributeRemoteWakeupMsk = 0x20
	specDescriptorConfigureAttributeRemoteWakeupPos = 5

	// USB standard descriptor endpoint bmAttributes
	specDescriptorEndpointAddressDirectionMsk = 0x80
	specDescriptorEndpointAddressDirectionPos = 7
	specDescriptorEndpointAddressDirectionOut = 0
	specDescriptorEndpointAddressDirectionIn  = 0x80

	specDescriptorEndpointAddressNumberMsk   = 0x0F
	specDescriptorEndpointAddressNumberShfit = 0

	specDescriptorEndpointAttributeTypeMsk     = 0x03
	specDescriptorEndpointAttributeNumberShfit = 0

	specDescriptorEndpointAttributeSyncTypeMsk      = 0x0C
	specDescriptorEndpointAttributeSyncTypeShfit    = 2
	specDescriptorEndpointAttributeSyncTypeNoSync   = 0x00
	specDescriptorEndpointAttributeSyncTypeAsync    = 0x04
	specDescriptorEndpointAttributeSyncTypeAdaptive = 0x08
	specDescriptorEndpointAttributeSyncTypeSync     = 0x0C

	specDescriptorEndpointAttributeUsageTypeMsk                          = 0x30
	specDescriptorEndpointAttributeUsageTypeShfit                        = 4
	specDescriptorEndpointAttributeUsageTypeDataEndpoint                 = 0x00
	specDescriptorEndpointAttributeUsageTypeFeedbackEndpoint             = 0x10
	specDescriptorEndpointAttributeUsageTypeImplicitFeedbackDataEndpoint = 0x20

	specDescriptorEndpointMaxpacketsizeSizeMsk               = 0x07FF
	specDescriptorEndpointMaxpacketsizeMultTransactionsMsk   = 0x1800
	specDescriptorEndpointMaxpacketsizeMultTransactionsShfit = 11

	// USB standard descriptor otg bmAttributes
	specDescriptorOTGAttributesSRPMsk = 0x01
	specDescriptorOTGAttributesHNPMsk = 0x02
	specDescriptorOTGAttributesADPMsk = 0x04

	// USB standard descriptor device capability usb20 extension bmAttributes
	specDescriptorDeviceCapabilityUSB20ExtensionLPMMsk  = 0x02
	specDescriptorDeviceCapabilityUSB20ExtensionLPMPos  = 1
	specDescriptorDeviceCapabilityUSB20ExtensionBESLMsk = 0x04
	specDescriptorDeviceCapabilityUSB20ExtensionBESLPos = 2
)
