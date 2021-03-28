package usb

// Unexported USB device type definitions.
type (
	// deviceClassDriver defines the class driver interface.
	deviceClassDriver interface {
		init(device *device, config *deviceClassConfig) status    // Class driver initialization- entry  of the class driver
		deinit() status                                           // Class driver de-initialization
		event(event deviceClassEventID, param interface{}) status // Class driver event callback
		send(ep uint8, buffer []uint8, length uint32) status      // Class driver send to endpoint
		receive(ep uint8, buffer []uint8, length uint32) status   // Class driver receive from endpoint
	}

	deviceNotificationID   uint8
	deviceControlID        uint8
	deviceStatusID         uint8
	deviceStateID          uint8
	deviceEndpointStatusID uint8
	deviceEventID          uint8
	deviceClassID          uint8
	deviceClassEventID     uint8

	// deviceEndpoint contains the information for a USB device endpoint.
	deviceEndpoint struct {
		endpointAddress uint8  // Endpoint address
		transferType    uint8  // Endpoint transfer type
		maxPacketSize   uint16 // Endpoint maximum packet size
		interval        uint8  // Endpoint interval
	}

	deviceEndpointCallbackMessage struct {
		buffer  []uint8 // Transferred buffer
		length  uint32  // Transferred data length
		isSetup bool    // Is in a setup phase
	}

	deviceEndpointCallback struct {
		param  interface{} // Parameter for callback function
		isBusy bool
	}

	deviceEndpointConfig struct {
		maxPacketSize   uint16 // Endpoint maximum packet size
		endpointAddress uint8  // Endpoint address
		transferType    uint8  // Endpoint transfer type
		zlt             uint8  // ZLT flag
		interval        uint8  // Endpoint interval
	}

	deviceEndpointStatus struct {
		address uint8  // Endpoint address
		status  uint16 // Endpoint status (idle or stalled)
	}

	// deviceInterface contains the endpoints and class-specific information for
	// a USB device interface.
	deviceInterface struct {
		alternateSetting uint8            // Alternate setting number
		endpointList     []deviceEndpoint // Endpoints of the interface
		classSpecific    interface{}      // Class specific structure handle
	}

	// deviceSetup contains the setup information for a USB device.
	deviceSetup struct {
		bmRequestType uint8
		bRequest      uint8
		wValue        uint16
		wIndex        uint16
		wLength       uint16
	}

	device struct {
		port          uint8                                                // USB port (core index)
		controller    deviceController                                     // Controller interface
		epCallback    [2 * configDeviceMaxEndpoints]deviceEndpointCallback // Endpoint callback function structure
		deviceAddress uint8                                                // Current device address
		state         deviceStateID                                        // Current device state
		isResetting   bool                                                 // Is doing device reset or not
		hwTick        int64                                                // (volatile) Current hw tick (ms)
	}

	// deviceClassInterface contains the USB device class details, including
	// all of its device interfaces.
	deviceClassInterface struct {
		classCode       uint8             // Class code of the interface
		subclassCode    uint8             // Subclass code of the interface
		protocolCode    uint8             // Protocol code of the interface
		interfaceNumber uint8             // Interface number
		deviceInterface []deviceInterface // Interface structure list
	}

	// deviceClassInfo contains the USB device class ID and all of its class
	// interfaces.
	deviceClassInfo struct {
		classID       deviceClassID          // Class type
		interfaceList []deviceClassInterface // Interfaces of the class
	}

	// deviceClassConfig contains the configuration for a USB device class.
	deviceClassConfig struct {
		driver deviceClassDriver // USB device class driver interface
		info   deviceClassInfo   // Detailed information of the class
	}

	// deviceClass contains common device class state information.
	deviceClass struct {
		deviceHandle      *device             // USB device handle
		classConfig       []deviceClassConfig // USB device class configuration list
		setupBuffer       []uint8             // Setup packet data buffer
		transcationBuffer uint16              // Get status/configuration, get/set interface, get sync frame
	}
	// 			This structure is used to pass the control request information.
	// 			The structure is used in following two cases.
	// 			1. Case one, the host wants to send data to the device in the control data stage: @n
	// 			        a. If a setup packet is received, the structure is used to pass the setup packet data and wants to get the
	// 			buffer to receive data sent from the host.
	// 			           The field isSetup is 1.
	// 			           The length is the requested buffer length.
	// 			           The buffer is filled by the class or application by using the valid buffer address.
	// 			           The setup is the setup packet address.
	// 			        b. If the data received is sent by the host, the structure is used to pass the data buffer address and the
	// 			data
	// 			length sent by the host.
	// 			           In this way, the field isSetup is 0.
	// 			           The buffer is the address of the data sent from the host.
	// 			           The length is the received data length.
	// 			           The setup is the setup packet address. @n
	// 			2. Case two, the host wants to get data from the device in control data stage: @n
	// 			           If the setup packet is received, the structure is used to pass the setup packet data and wants to get the
	// 			data buffer address to send data to the host.
	// 			           The field isSetup is 1.
	// 			           The length is the requested data length.
	// 			           The buffer is filled by the class or application by using the valid buffer address.
	// 			           The setup is the setup packet address.
	deviceControlRequest struct {
		setup   deviceSetup // Setup data
		buffer  []uint8     // Buffer
		length  uint32      // Buffer length or requested length
		isSetup bool        // Indicates whether a setup packet is received
	}

	// deviceGetDescriptorCommon contains the result of a control request for:
	// get descriptor common
	deviceGetDescriptorCommon struct {
		buffer []uint8 // Buffer
		length uint32  // Buffer length
	}

	// deviceGetDeviceDescriptor contains the result of a control request for:
	// get device descriptor
	deviceGetDeviceDescriptor struct {
		buffer []uint8 // Buffer
		length uint32  // Buffer length
	}

	// deviceGetDeviceQualifierDescriptor contains the result of a control
	// request for: get device qualifier descriptor
	deviceGetDeviceQualifierDescriptor struct {
		buffer []uint8 // Buffer
		length uint32  // Buffer length
	}

	// deviceGetConfigurationDescriptor contains the result of a control request
	// for: get configuration descriptor
	deviceGetConfigurationDescriptor struct {
		buffer        []uint8 // Buffer
		length        uint32  // Buffer length
		configuration uint8   // The configuration number
	}

	// deviceGetBOSDescriptor contains the result of a control request for: get
	// bos descriptor
	deviceGetBOSDescriptor struct {
		buffer []uint8 // Buffer
		length uint32  // Buffer length
	}

	// deviceGetStringDescriptor contains the result of a control request for:
	// get string descriptor
	deviceGetStringDescriptor struct {
		buffer      []uint8 // Buffer
		length      uint32  // Buffer length
		languageID  uint16  // Language ID
		stringIndex uint8   // String index
	}

	// deviceGetHIDDescriptor contains the result of a control request for: get
	// HID descriptor
	deviceGetHIDDescriptor struct {
		buffer          []uint8 // Buffer
		length          uint32  // Buffer length
		interfaceNumber uint8   // The interface number
	}

	// deviceGetHIDReportDescriptor contains the result of a control request for:
	// get HID report descriptor
	deviceGetHIDReportDescriptor struct {
		buffer          []uint8 // Buffer
		length          uint32  // Buffer length
		interfaceNumber uint8   // The interface number
	}

	// deviceGetHIDPhysicalDescriptor contains the result of a control request
	// for: get HID physical descriptor
	deviceGetHIDPhysicalDescriptor struct {
		buffer          []uint8 // Buffer
		length          uint32  // Buffer length
		index           uint8   // Physical index
		interfaceNumber uint8   // The interface number
	}

	deviceCallbackMessage struct {
		buffer  []uint8              // Transferred buffer
		length  uint32               // Transferred data length
		code    deviceNotificationID // Notification code
		isSetup bool                 // Is in a setup phase
	}
)

// Unexported enumerated constant values for USB device.
const (
	deviceNotifyBusReset          deviceNotificationID = iota + 1 // Reset signal detected
	deviceNotifySuspend                                           // Suspend signal detected
	deviceNotifyResume                                            // Resume signal detected
	deviceNotifyLPMSleep                                          // LPM signal detected
	deviceNotifyLPMResume                                         // Resume signal detected
	deviceNotifyError                                             // Errors happened in bus
	deviceNotifyDetach                                            // Device disconnected from a host
	deviceNotifyAttach                                            // Device connected to a host
	deviceNotifyDCDDetectFinished                                 // Device charger detection finished

	deviceControlRun                 deviceControlID = iota // Enable the device functionality
	deviceControlStop                                       // Disable the device functionality
	deviceControlEndpointInit                               // Initialize a specified endpoint
	deviceControlEndpointDeinit                             // De-initialize a specified endpoint
	deviceControlEndpointStall                              // Stall a specified endpoint
	deviceControlEndpointUnstall                            // Un-stall a specified endpoint
	deviceControlGetDeviceStatus                            // Get device status
	deviceControlGetEndpointStatus                          // Get endpoint status
	deviceControlSetDeviceAddress                           // Set device address
	deviceControlGetSynchFrame                              // Get current frame
	deviceControlResume                                     // Drive controller to generate a resume signal in USB bus
	deviceControlSleepResume                                // Drive controller to generate a LPM resume signal in USB bus
	deviceControlSuspend                                    // Drive controller to enter into suspend mode
	deviceControlSleep                                      // Drive controller to enter into sleep mode
	deviceControlSetDefaultStatus                           // Set controller to default status
	deviceControlGetSpeed                                   // Get current speed
	deviceControlGetOTGStatus                               // Get OTG status
	deviceControlSetOTGStatus                               // Set OTG status
	deviceControlSetTestMode                                // Drive xCHI into test mode
	deviceControlGetRemoteWakeUp                            // Get flag of LPM Remote Wake-up Enabled by USB host.
	deviceControlDCDDisable                                 // disable dcd module function.
	deviceControlDCDEnable                                  // enable dcd module function.
	deviceControlPreSetDeviceAddress                        // Pre set device address
	deviceControlUpdateHwTick                               // update hardware tick

	deviceStatusTestMode       deviceStatusID = iota + 1 // Test mode
	deviceStatusSpeed                                    // Current speed
	deviceStatusOTG                                      // OTG status
	deviceStatusDevice                                   // Device status
	deviceStatusEndpoint                                 // Endpoint state usb_device_endpoint_status_t
	deviceStatusDeviceState                              // Device state
	deviceStatusAddress                                  // Device address
	deviceStatusSynchFrame                               // Current frame
	deviceStatusBus                                      // Bus status
	deviceStatusBusSuspend                               // Bus suspend
	deviceStatusBusSleep                                 // Bus suspend
	deviceStatusBusResume                                // Bus resume
	deviceStatusRemoteWakeup                             // Remote wakeup state
	deviceStatusBusSleepResume                           // Bus resume

	deviceStateConfigured deviceStateID = iota // Device state, Configured
	deviceStateAddress                         // Device state, Address
	deviceStateDefault                         // Device state, Default
	deviceStateAddressing                      // Device state, Address setting
	deviceStateTestMode                        // Device state, Test mode
	deviceStateInit                            // Device state, initializing

	deviceEndpointStateIdle    deviceEndpointStatusID = iota // Endpoint state, idle
	deviceEndpointStateStalled                               // Endpoint state, stalled

	deviceEventBusReset                     deviceEventID = iota + 1 // USB bus reset signal detected
	deviceEventSuspend                                               // USB bus suspend signal detected
	deviceEventResume                                                // USB bus resume signal detected. The resume signal is driven by itself or a host
	deviceEventSleeped                                               // USB bus LPM suspend signal detected
	deviceEventLPMResume                                             // USB bus LPM resume signal detected. The resume signal is driven by itself or a host
	deviceEventError                                                 // An error is happened in the bus.
	deviceEventDetach                                                // USB device is disconnected from a host.
	deviceEventAttach                                                // USB device is connected to a host.
	deviceEventSetConfiguration                                      // Set configuration.
	deviceEventSetInterface                                          // Set interface.
	deviceEventGetDeviceDescriptor                                   // Get device descriptor.
	deviceEventGetConfigurationDescriptor                            // Get configuration descriptor.
	deviceEventGetStringDescriptor                                   // Get string descriptor.
	deviceEventGetHIDDescriptor                                      // Get HID descriptor.
	deviceEventGetHIDReportDescriptor                                // Get HID report descriptor.
	deviceEventGetHIDPhysicalDescriptor                              // Get HID physical descriptor.
	deviceEventGetBOSDescriptor                                      // Get configuration descriptor.
	deviceEventGetDeviceQualifierDescriptor                          // Get device qualifier descriptor.
	deviceEventVendorRequest                                         // Vendor request.
	deviceEventSetRemoteWakeup                                       // Enable or disable remote wakeup function.
	deviceEventGetConfiguration                                      // Get current configuration index
	deviceEventGetInterface                                          // Get current interface alternate setting value
	deviceEventSetBHNPEnable                                         // Enable or disable BHNP.
	deviceEventDCDDetectionfinished                                  // The DCD detection finished

	deviceClassInvalid deviceClassID = iota
	deviceClassHID
	deviceClassCDC
	deviceClassMSC
	deviceClassAudio
	deviceClassPHDC
	deviceClassVideo
	deviceClassPrinter
	deviceClassDFU
	deviceClassCCID

	deviceClassEventInvalid deviceClassEventID = iota
	deviceClassEventClassRequest
	deviceClassEventDeviceReset
	deviceClassEventSetConfiguration
	deviceClassEventSetInterface
	deviceClassEventSetEndpointHalt
	deviceClassEventClearEndpointHalt
)

var (
	deviceClassInstance [configDeviceCount]deviceClass
)

func (d *device) init(port uint8) (s status) {

	// initialize device
	d.port = port
	d.controller = d.initController()
	d.deviceAddress = 0
	d.state = deviceStateDefault
	d.isResetting = false
	d.hwTick = 0
	for i := range d.epCallback {
		d.epCallback[i].param = nil
		d.epCallback[i].isBusy = false
	}

	// initialize platform via device controller interface
	return d.controller.init()
}

func (d *device) notification(message *deviceCallbackMessage) {

}

func (d *device) class(config []deviceClassConfig) *deviceClass {

	// verify a device configuration was provided
	if nil == d || nil == config || len(config) == 0 {
		return nil
	}

	c := &deviceClassInstance[d.port]

	// initialize device class
	c.deviceHandle = d
	c.classConfig = config
	c.setupBuffer = nil // TODO
	c.transcationBuffer = 0

	// initialze each of the device class drivers
	for i := range c.classConfig {
		if nil != c.classConfig[i].driver {
			if !c.classConfig[i].driver.init(d, &c.classConfig[i]).OK() {
				// remove the driver from configuration if it fails initialization
				c.classConfig[i].driver = nil
			}
		}
	}

	return c
}
