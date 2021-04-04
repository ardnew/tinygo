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

	deviceEventFunc            func(ev deviceEventID, param interface{}) status
	deviceEndpointCallbackFunc func(message deviceEndpointCallbackMessage, param interface{}) status

	deviceNotificationID   uint8
	deviceControlID        uint8
	deviceStatusID         uint8
	deviceStateID          uint8
	deviceEndpointStatusID uint8
	deviceEventID          uint8
	deviceClassID          uint8
	deviceClassEventID     uint8

	deviceEndpointCallbackMessage struct {
		buffer  []uint8 // Transferred buffer
		length  uint32  // Transferred data length
		isSetup bool    // Is in a setup phase
	}

	deviceEndpointCallbackList [2 * configDeviceMaxEndpoints]deviceEndpointCallback
	deviceEndpointCallback     struct {
		callback deviceEndpointCallbackFunc
		param    interface{} // Parameter for callback function
		isBusy   bool
	}

	deviceEndpointConfig struct {
		maxPacketSize uint16 // Endpoint maximum packet size
		address       uint8  // Endpoint address
		transferType  uint8  // Endpoint transfer type
		zlt           uint8  // ZLT flag
		interval      uint8  // Endpoint interval
	}

	deviceEndpointStatus struct {
		address uint8  // Endpoint address
		status  uint16 // Endpoint status (idle or stalled)
	}

	// deviceEndpoint contains the information for a USB device endpoint.
	deviceEndpoint struct {
		address       uint8  // Endpoint address
		transferType  uint8  // Endpoint transfer type
		maxPacketSize uint16 // Endpoint maximum packet size
		interval      uint8  // Endpoint interval
	}

	// deviceInterface contains the endpoints and class-specific information for
	// a USB device interface.
	deviceInterface struct {
		alternateSetting uint8            // Alternate setting number
		endpoint         []deviceEndpoint // Endpoints of the interface
		classSpecific    interface{}      // Class specific structure handle
	}

	deviceNotification struct {
		buffer  []uint8              // Transferred buffer
		length  uint32               // Transferred data length
		code    deviceNotificationID // Notification code
		isSetup bool                 // Is in a setup phase
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
		port             uint8                      // USB port (core index)
		controller       deviceController           // Controller interface
		class            *deviceClass               // USB device class
		endpointCallback deviceEndpointCallbackList // Endpoint callback function structure
		deviceAddress    uint8                      // Current device address
		state            deviceStateID              // Current device state
		isResetting      bool                       // Is doing device reset or not
		hwTick           int64                      // (volatile) Current hw tick (ms)
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
		device            *device             // USB device handle
		config            []deviceClassConfig // USB device class configuration list
		event             deviceEventFunc     // application callback
		setupBuffer       []uint8             // Setup packet data buffer
		transcationBuffer uint16              // Get status/configuration, get/set interface, get sync frame
	}

	// // 			This structure is used to pass the control request information.
	// // 			The structure is used in following two cases.
	// // 			1. Case one, the host wants to send data to the device in the control data stage: @n
	// // 			        a. If a setup packet is received, the structure is used to pass the setup packet data and wants to get the
	// // 			buffer to receive data sent from the host.
	// // 			           The field isSetup is 1.
	// // 			           The length is the requested buffer length.
	// // 			           The buffer is filled by the class or application by using the valid buffer address.
	// // 			           The setup is the setup packet address.
	// // 			        b. If the data received is sent by the host, the structure is used to pass the data buffer address and the
	// // 			data
	// // 			length sent by the host.
	// // 			           In this way, the field isSetup is 0.
	// // 			           The buffer is the address of the data sent from the host.
	// // 			           The length is the received data length.
	// // 			           The setup is the setup packet address. @n
	// // 			2. Case two, the host wants to get data from the device in control data stage: @n
	// // 			           If the setup packet is received, the structure is used to pass the setup packet data and wants to get the
	// // 			data buffer address to send data to the host.
	// // 			           The field isSetup is 1.
	// // 			           The length is the requested data length.
	// // 			           The buffer is filled by the class or application by using the valid buffer address.
	// // 			           The setup is the setup packet address.
	deviceControlRequest struct {
		setup   deviceSetup // Setup data
		buffer  []uint8     // Buffer
		length  uint32      // Buffer length or requested length
		isSetup bool        // Indicates whether a setup packet is received
	}

	// // deviceGetDescriptorCommon contains the result of a control request for:
	// // get descriptor common
	// deviceGetDescriptorCommon struct {
	// 	buffer []uint8 // Buffer
	// 	length uint32  // Buffer length
	// }

	// // deviceGetDeviceDescriptor contains the result of a control request for:
	// // get device descriptor
	// deviceGetDeviceDescriptor struct {
	// 	buffer []uint8 // Buffer
	// 	length uint32  // Buffer length
	// }

	// // deviceGetDeviceQualifierDescriptor contains the result of a control
	// // request for: get device qualifier descriptor
	// deviceGetDeviceQualifierDescriptor struct {
	// 	buffer []uint8 // Buffer
	// 	length uint32  // Buffer length
	// }

	// // deviceGetConfigurationDescriptor contains the result of a control request
	// // for: get configuration descriptor
	// deviceGetConfigurationDescriptor struct {
	// 	buffer        []uint8 // Buffer
	// 	length        uint32  // Buffer length
	// 	configuration uint8   // The configuration number
	// }

	// // deviceGetBOSDescriptor contains the result of a control request for: get
	// // bos descriptor
	// deviceGetBOSDescriptor struct {
	// 	buffer []uint8 // Buffer
	// 	length uint32  // Buffer length
	// }

	// // deviceGetStringDescriptor contains the result of a control request for:
	// // get string descriptor
	// deviceGetStringDescriptor struct {
	// 	buffer      []uint8 // Buffer
	// 	length      uint32  // Buffer length
	// 	languageID  uint16  // Language ID
	// 	stringIndex uint8   // String index
	// }

	// // deviceGetHIDDescriptor contains the result of a control request for: get
	// // HID descriptor
	// deviceGetHIDDescriptor struct {
	// 	buffer          []uint8 // Buffer
	// 	length          uint32  // Buffer length
	// 	interfaceNumber uint8   // The interface number
	// }

	// // deviceGetHIDReportDescriptor contains the result of a control request for:
	// // get HID report descriptor
	// deviceGetHIDReportDescriptor struct {
	// 	buffer          []uint8 // Buffer
	// 	length          uint32  // Buffer length
	// 	interfaceNumber uint8   // The interface number
	// }

	// // deviceGetHIDPhysicalDescriptor contains the result of a control request
	// // for: get HID physical descriptor
	// deviceGetHIDPhysicalDescriptor struct {
	// 	buffer          []uint8 // Buffer
	// 	length          uint32  // Buffer length
	// 	index           uint8   // Physical index
	// 	interfaceNumber uint8   // The interface number
	// }

)

// Unexported enumerated constant values for USB device.
const (
	deviceNotifyBusReset          deviceNotificationID = iota + 0x10 // Reset signal detected
	deviceNotifySuspend                                              // Suspend signal detected
	deviceNotifyResume                                               // Resume signal detected
	deviceNotifyLPMSleep                                             // LPM signal detected
	deviceNotifyLPMResume                                            // Resume signal detected
	deviceNotifyError                                                // Errors happened in bus
	deviceNotifyDetach                                               // Device disconnected from a host
	deviceNotifyAttach                                               // Device connected to a host
	deviceNotifyDCDDetectFinished                                    // Device charger detection finished
)

const (
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
)

const (
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
)

const (
	deviceStateConfigured deviceStateID = iota // Device state, Configured
	deviceStateAddress                         // Device state, Address
	deviceStateDefault                         // Device state, Default
	deviceStateAddressing                      // Device state, Address setting
	deviceStateTestMode                        // Device state, Test mode
	deviceStateInit                            // Device state, initializing
)

const (
	deviceEndpointStateIdle    deviceEndpointStatusID = iota // Endpoint state, idle
	deviceEndpointStateStalled                               // Endpoint state, stalled
)

const (
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
)

const (
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
)

const (
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
	d.controller = d.initController() // obtain a device controller handle
	d.deviceAddress = 0
	d.state = deviceStateDefault
	d.isResetting = false
	d.hwTick = 0
	for i := range d.endpointCallback {
		d.endpointCallback[i].callback = nil
		d.endpointCallback[i].param = nil
		d.endpointCallback[i].isBusy = false
	}

	// initialize platform via device controller interface
	return d.controller.init()
}

func (d *device) deinit() status {

	// de=initialize device
	s := d.controller.deinit()
	d.controller = nil
	return s
}

func (d *device) initClass(config []deviceClassConfig, event deviceEventFunc) *deviceClass {

	// verify a device configuration was provided
	if nil == d || nil == config || len(config) == 0 {
		return nil
	}

	c := &deviceClassInstance[d.port]

	// initialize device class
	c.device = d
	c.config = config
	c.event = event
	c.setupBuffer = nil // TODO
	c.transcationBuffer = 0

	// add a class reference to the receiver
	d.class = c

	// initialze each of the device class drivers
	for i := range c.config {
		if nil != c.config[i].driver {
			if !c.config[i].driver.init(d, &c.config[i]).OK() {
				// remove the driver from configuration if it fails initialization
				c.config[i].driver = nil
			}
		}
	}

	return c
}

func (d *device) initControlPipes() status {

	addrIn, addrOut :=
		uint8(specEndpointControl|specDescriptorEndpointAddressDirectionIn),
		uint8(specEndpointControl|specDescriptorEndpointAddressDirectionOut)

	if s := d.initEndpoint(
		deviceEndpointConfig{
			maxPacketSize: configDeviceControllerMaxPacketSize,
			address:       addrIn,
			transferType:  specEndpointControl,
			zlt:           1,
			interval:      0,
		},
		deviceEndpointCallback{
			callback: d.controlEndpoint,
			param:    d.class,
		},
	); !s.OK() {
		return s
	}

	if s := d.initEndpoint(
		deviceEndpointConfig{
			maxPacketSize: configDeviceControllerMaxPacketSize,
			address:       addrOut,
			transferType:  specEndpointControl,
			zlt:           1,
			interval:      0,
		},
		deviceEndpointCallback{
			callback: d.controlEndpoint,
			param:    d.class,
		},
	); !s.OK() {
		_ = d.deinitEndpoint(addrIn)
		return s
	}

	return statusSuccess
}

func (d *device) initEndpoint(config deviceEndpointConfig, callback deviceEndpointCallback) status {

	endpoint, direction := unpackEndpoint(config.address)

	if endpoint >= configDeviceMaxEndpoints {
		return statusInvalidParameter
	}

	d.endpointCallback[(endpoint<<1)|direction].callback = callback.callback
	d.endpointCallback[(endpoint<<1)|direction].param = callback.param
	d.endpointCallback[(endpoint<<1)|direction].isBusy = false

	return d.controller.control(deviceControlEndpointInit, config)
}

func (d *device) deinitEndpoint(address uint8) status {

	s := d.controller.control(deviceControlEndpointDeinit, address)

	endpoint, direction := unpackEndpoint(address)

	if endpoint >= configDeviceMaxEndpoints {
		return statusInvalidParameter
	}

	d.endpointCallback[(endpoint<<1)|direction].callback = nil
	d.endpointCallback[(endpoint<<1)|direction].param = nil
	d.endpointCallback[(endpoint<<1)|direction].isBusy = false

	return s
}

func (d *device) controlEndpoint(message deviceEndpointCallbackMessage, param interface{}) status {
	return statusSuccess
}

func (d *device) event(ev deviceEventID, param interface{}) {

	switch ev {
	case deviceEventBusReset:
		// initialize control pipes
		d.initControlPipes()

		// notify all classes of a bus reset signal
		for i := range d.class.config {
			_ = d.class.config[i].driver.event(deviceClassEventDeviceReset, d.class)
		}
	}
	// notify the application driver of all events
	if nil != d.class.event {
		_ = d.class.event(ev, param)
	}
}

func (d *device) notify(message deviceNotification) {

	switch message.code {
	case deviceNotifyBusReset:
		d.notifyReset(message)

	default:
		endpoint, direction := unpackEndpoint(uint8(message.code))

		if endpoint < configDeviceMaxEndpoints {
			if nil != d.endpointCallback[(endpoint<<1)|direction].callback {
				if message.isSetup {
					d.endpointCallback[0].isBusy = false
					d.endpointCallback[1].isBusy = false
				} else {
					d.endpointCallback[(endpoint<<1)|direction].isBusy = false
				}
				// call endpoint callback
				_ = d.endpointCallback[(endpoint<<1)|direction].callback(
					deviceEndpointCallbackMessage{
						buffer:  message.buffer,
						length:  message.length,
						isSetup: message.isSetup,
					},
					d.endpointCallback[(endpoint<<1)|direction].param,
				)
			}
		}
	}
}

func (d *device) status(deviceStatus deviceStatusID, param interface{}) status {

	if nil == param {
		return statusInvalidParameter
	}

	switch deviceStatus {
	case deviceStatusSpeed:
		return d.controller.control(deviceControlGetSpeed, param)

	case deviceStatusOTG:
		return d.controller.control(deviceControlGetOTGStatus, param)

	case deviceStatusDeviceState:
		if state, ok := param.(*deviceStateID); ok {
			*state = d.state
			return statusSuccess
		}
		return statusInvalidParameter

	case deviceStatusAddress:
		if address, ok := param.(*uint8); ok {
			*address = d.deviceAddress
			return statusSuccess
		}
		return statusInvalidParameter

	case deviceStatusDevice:
		return d.controller.control(deviceControlGetDeviceStatus, param)

	case deviceStatusEndpoint:
		return d.controller.control(deviceControlGetEndpointStatus, param)

	case deviceStatusSynchFrame:
		return d.controller.control(deviceControlGetSynchFrame, param)

	default:
		return statusInvalidParameter
	}
}

func (d *device) setStatus(deviceStatus deviceStatusID, param interface{}) status {

	switch deviceStatus {
	case deviceStatusOTG:
		return d.controller.control(deviceControlSetOTGStatus, param)

	case deviceStatusDeviceState:
		if state, ok := param.(deviceStateID); ok {
			d.state = state
			return statusSuccess
		}
		return statusInvalidParameter

	case deviceStatusAddress:
		if d.state != deviceStateAddressing {
			if address, ok := param.(uint8); ok {
				d.deviceAddress = address
				d.state = deviceStateAddressing
				return d.controller.control(deviceControlPreSetDeviceAddress, d.deviceAddress)
			}
			return statusInvalidParameter
		}
		return d.controller.control(deviceControlSetDeviceAddress, d.deviceAddress)

	case deviceStatusBusResume:
		return d.controller.control(deviceControlResume, param)

	case deviceStatusBusSleepResume:
		return d.controller.control(deviceControlSleepResume, param)

	case deviceStatusBusSuspend:
		return d.controller.control(deviceControlSuspend, param)

	case deviceStatusBusSleep:
		return d.controller.control(deviceControlSleep, param)

	default:
		return statusInvalidParameter
	}
}

func (d *device) notifyReset(message deviceNotification) {

	d.isResetting = true
	_ = d.controller.control(deviceControlSetDefaultStatus, nil)

	d.state = deviceStateDefault
	d.deviceAddress = 0

	for count := 0; count < 2*configDeviceMaxEndpoints; count++ {
		d.endpointCallback[count].callback = nil
		d.endpointCallback[count].param = nil
		d.endpointCallback[count].isBusy = false
	}

	d.event(deviceEventBusReset, nil)
	d.isResetting = false
}

func (d *device) busSpeed(speed *uint8) status {
	return d.status(deviceStatusSpeed, speed)
}
