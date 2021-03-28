package usb

type (
	deviceCDCACMEventID uint8

	// deviceCDCACMConfig defines the platform-specific configuration settings for
	// a single CDC-ACM device class on a given USB port.
	//
	// To connect the USB CDC-ACM device driver to a particular platform, there
	// should be one instance for each USB CDC-ACM port, defined in global array
	// configDeviceCDCACM, indexed by USB port.
	deviceCDCACMConfig struct {
		// USB configuration
		interfaceSpeed uint8
		// Serial line configuration
		lineCodingSize       uint32
		lineCodingBaudRate   uint32
		lineCodingCharFormat uint32
		lineCodingParityType uint32
		lineCodingDataBits   uint32
		// Interface indices
		commInterfaceIndex uint8
		dataInterfaceIndex uint8
		// Endpoints
		interruptInEndpoint uint8
		bulkInEndpoint      uint8
		bulkOutEndpoint     uint8
		// Packet size
		interruptInPacketSize uint16
		interruptInInterval   uint8
		bulkInPacketSize      uint16
		bulkOutPacketSize     uint16
	}

	deviceCDCACMRequestParam struct {
		buffer         *[]uint8 // The pointer to the address of the buffer for CDC class request.
		length         *uint32  // The pointer to the length of the buffer for CDC class request.
		interfaceIndex uint16   // The interface index of the setup packet.
		setupValue     uint16   // The wValue field of the setup packet.
		isSetup        bool     // The flag indicates if it is a setup packet
	}

	deviceCDCACMPipe struct {
		pipeDataBuffer []uint8 // pipe data buffer backup when stall
		pipeDataLen    uint32  // pipe data length backup when stall
		pipeStall      uint8   // pipe is stall
		ep             uint8   // The endpoint number of the pipe.
		isBusy         bool    // The pipe is transferring packet
	}

	deviceCDCACM struct {
		deviceHandle    *device            // The handle of the USB device.
		configHandle    *deviceClassConfig // The class configure structure.
		commHandle      *deviceInterface   // The CDC communication interface handle.
		dataHandle      *deviceInterface   // The CDC data interface handle.
		bulkIn          deviceCDCACMPipe   // The bulk in pipe for sending packet to host.
		bulkOut         deviceCDCACMPipe   // The bulk out pipe for receiving packet from host.
		interruptIn     deviceCDCACMPipe   // The interrupt in pipe for notifying the device state to host.
		configuration   uint8              // The current configuration value.
		interfaceNumber uint8              // The current interface number.
		alternate       uint8              // The alternate setting value of the interface.
		hasSentState    bool               // The device has primed the state in interrupt pipe
		speed           uint8              // Speed of USB device (Full/Low/High)
		lineCodingSize  uint32             // Size of line-coding message
		baudRate        uint32             // Data terminal rate
		charFormat      uint32             // Character format
		parityType      uint32             // Parity type
		dataBits        uint32             // Data word size
	}
)

const (
	deviceCDCACMEventSendResponse            deviceCDCACMEventID = iota + 1 // This event indicates the bulk send transfer is complete or cancelled etc.
	deviceCDCACMEventRecvResponse                                           // This event indicates the bulk receive transfer is complete or cancelled etc..
	deviceCDCACMEventSerialStateNotif                                       // This event indicates the serial state has been sent to the host.
	deviceCDCACMEventSendEncapsulatedCommand                                // This event indicates the device received the SEND_ENCAPSULATED_COMMAND request.
	deviceCDCACMEventGetEncapsulatedResponse                                // This event indicates the device received the GET_ENCAPSULATED_RESPONSE request.
	deviceCDCACMEventSetCommFeature                                         // This event indicates the device received the SET_COMM_FEATURE request.
	deviceCDCACMEventGetCommFeature                                         // This event indicates the device received the GET_COMM_FEATURE request.
	deviceCDCACMEventClearCommFeature                                       // This event indicates the device received the CLEAR_COMM_FEATURE request.
	deviceCDCACMEventGetLineCoding                                          // This event indicates the device received the GET_LINE_CODING request.
	deviceCDCACMEventSetLineCoding                                          // This event indicates the device received the SET_LINE_CODING request.
	deviceCDCACMEventSetControlLineState                                    // This event indicates the device received the SET_CONTRL_LINE_STATE request.
	deviceCDCACMEventSendBreak                                              // This event indicates the device received the SEND_BREAK request.
)

var (
	deviceCDCACMConfigInstance = [configDeviceCDCACMCount][]deviceClassConfig{
		{{
			driver: &deviceCDCACM{},
			info: deviceClassInfo{
				classID: deviceClassCDC,
				interfaceList: []deviceClassInterface{
					{ // USB CDC-ACM communications/control interface
						classCode:    deviceCDCCommClass,
						subclassCode: deviceCDCAbstractControlModel,
						protocolCode: deviceCDCNoClassSpecificProtocol,
						deviceInterface: []deviceInterface{{
							alternateSetting: 0,
							endpointList: []deviceEndpoint{
								{transferType: specEndpointInterrupt},
							},
						}},
					},
					{ // USB CDC-ACM serial data interface
						classCode:    deviceCDCDataClass,
						subclassCode: 0x00,
						protocolCode: deviceCDCNoClassSpecificProtocol,
						deviceInterface: []deviceInterface{{
							alternateSetting: 0,
							endpointList: []deviceEndpoint{
								{transferType: specEndpointBulk},
								{transferType: specEndpointBulk},
							},
						}},
					},
				},
			},
		}},
	}
)

// setCDCACM assigns a CDC-ACM configuration to the receiver port p and then
// returns the configured deviceClassDriver and deviceClass that were associated
// with the receiver.
func (p *port) setCDCACM() (*deviceCDCACM, *deviceClass) {

	// verify a valid port was provided
	if nil == p || nil == p.device.controller || p.mode != modeDevice {
		return nil, nil
	}

	// get a reference to each of the class interfaces
	comm := &deviceCDCACMConfigInstance[p.device.port][0].info.interfaceList[0]
	data := &deviceCDCACMConfigInstance[p.device.port][0].info.interfaceList[1]

	// configDeviceCDCACM must be defined per package API. these settings will be
	// platform-specific, and will probably be implemented in a build tag-
	// constrained source file. the length of this array corresponds to the number
	// of USB CDC-ACM ports that are being created, and the index of each element
	// corresponds to the physical USB port (core index).

	// CDC-ACM communications/control interface
	comm.interfaceNumber =
		configDeviceCDCACM[p.device.port].commInterfaceIndex
	comm.deviceInterface[0].endpointList[0].endpointAddress =
		configDeviceCDCACM[p.device.port].interruptInEndpoint | (specIn << 7)
	comm.deviceInterface[0].endpointList[0].maxPacketSize =
		configDeviceCDCACM[p.device.port].interruptInPacketSize
	comm.deviceInterface[0].endpointList[0].interval =
		configDeviceCDCACM[p.device.port].interruptInInterval

	// CDC-ACM data interface
	data.interfaceNumber =
		configDeviceCDCACM[p.device.port].dataInterfaceIndex
	data.deviceInterface[0].endpointList[0].endpointAddress =
		configDeviceCDCACM[p.device.port].bulkInEndpoint | (specIn << 7)
	data.deviceInterface[0].endpointList[0].maxPacketSize =
		configDeviceCDCACM[p.device.port].bulkInPacketSize
	data.deviceInterface[0].endpointList[1].endpointAddress =
		configDeviceCDCACM[p.device.port].bulkOutEndpoint | (specOut << 7)
	data.deviceInterface[0].endpointList[1].maxPacketSize =
		configDeviceCDCACM[p.device.port].bulkOutPacketSize

	cls := p.device.class(deviceCDCACMConfigInstance[p.device.port])
	acm := cls.classConfig[0].driver.(*deviceCDCACM)

	return acm, cls
}

func (acm *deviceCDCACM) init(device *device, config *deviceClassConfig) status {
	// initialize our associated object references
	acm.deviceHandle = device
	acm.configHandle = config

	// initialize line-coding details from global configuration
	acm.speed = configDeviceCDCACM[device.port].interfaceSpeed
	acm.lineCodingSize = configDeviceCDCACM[device.port].lineCodingSize
	acm.baudRate = configDeviceCDCACM[device.port].lineCodingBaudRate
	acm.charFormat = configDeviceCDCACM[device.port].lineCodingCharFormat
	acm.parityType = configDeviceCDCACM[device.port].lineCodingParityType
	acm.dataBits = configDeviceCDCACM[device.port].lineCodingDataBits

	// initialize remaining state data
	acm.configuration = 0
	acm.alternate = 0xFF

	return statusSuccess
}

func (acm *deviceCDCACM) deinit() status {
	return statusSuccess
}

func (acm *deviceCDCACM) event(event deviceClassEventID, param interface{}) status {
	return statusSuccess
}

func (acm *deviceCDCACM) send(ep uint8, buffer []uint8, length uint32) status {
	return statusSuccess
}

func (acm *deviceCDCACM) receive(ep uint8, buffer []uint8, length uint32) status {
	return statusSuccess
}
