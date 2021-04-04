package usb

import "bytes"

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
		interfaceSpeed uint8 // Low/Full/High-speed
		// CDC-ACM Serial line configuration
		lineCodingSize       uint32 // Size of line-coding message
		lineCodingBaudRate   uint32 // Data terminal rate
		lineCodingCharFormat uint32 // Character format
		lineCodingParityType uint32 // Parity type
		lineCodingDataBits   uint32 // Data word size
		// CDC-ACM Communication/control interface
		commInterfaceIndex        uint8  // Communication/control interface index
		commInterruptInEndpoint   uint8  // Interrupt input endpoint index (address)
		commInterruptInPacketSize uint16 // Interrupt input packet size
		commInterruptInInterval   uint8  // Interrupt input interval
		// CDC-ACM Data interface
		dataInterfaceIndex    uint8  // Data interface index
		dataBulkInEndpoint    uint8  // Bulk input endpoint index (address)
		dataBulkInPacketSize  uint16 // Bulk input packet size
		dataBulkOutEndpoint   uint8  // Bulk output endpoint index (address)
		dataBulkOutPacketSize uint16 // Bulk output packet size
	}

	deviceCDCACM struct {
		device          *device            // The handle of the USB device.
		config          *deviceClassConfig // The class configure structure.
		comm            *deviceInterface   // The CDC communication interface handle.
		data            *deviceInterface   // The CDC data interface handle.
		bulkIn          deviceCDCACMPipe   // The bulk in pipe for sending packet to host.
		bulkOut         deviceCDCACMPipe   // The bulk out pipe for receiving packet from host.
		interruptIn     deviceCDCACMPipe   // The interrupt in pipe for notifying the device state to host.
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
		pipeStall      bool    // pipe is stall
		ep             uint8   // The endpoint number of the pipe.
		isBusy         bool    // The pipe is transferring packet
	}
)

const (
	deviceCDCACMEventSendResponse            deviceClassEventID = iota + 1 // This event indicates the bulk send transfer is complete or cancelled etc.
	deviceCDCACMEventRecvResponse                                          // This event indicates the bulk receive transfer is complete or cancelled etc..
	deviceCDCACMEventSerialStateNotify                                     // This event indicates the serial state has been sent to the host.
	deviceCDCACMEventSendEncapsulatedCommand                               // This event indicates the device received the SEND_ENCAPSULATED_COMMAND request.
	deviceCDCACMEventGetEncapsulatedResponse                               // This event indicates the device received the GET_ENCAPSULATED_RESPONSE request.
	deviceCDCACMEventSetCommFeature                                        // This event indicates the device received the SET_COMM_FEATURE request.
	deviceCDCACMEventGetCommFeature                                        // This event indicates the device received the GET_COMM_FEATURE request.
	deviceCDCACMEventClearCommFeature                                      // This event indicates the device received the CLEAR_COMM_FEATURE request.
	deviceCDCACMEventGetLineCoding                                         // This event indicates the device received the GET_LINE_CODING request.
	deviceCDCACMEventSetLineCoding                                         // This event indicates the device received the SET_LINE_CODING request.
	deviceCDCACMEventSetControlLineState                                   // This event indicates the device received the SET_CONTRL_LINE_STATE request.
	deviceCDCACMEventSendBreak                                             // This event indicates the device received the SEND_BREAK request.
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
							endpoint: []deviceEndpoint{
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
							endpoint: []deviceEndpoint{
								{transferType: specEndpointBulk},
								{transferType: specEndpointBulk},
							},
						}},
					},
				},
			},
		}},
	}

	deviceCDCACMBufferInvalid32 = u32LE(0xFFFFFFFF)
)

func (acm *deviceCDCACM) init(device *device, config *deviceClassConfig) status {

	// initialize our associated object references
	acm.device = device
	acm.config = config

	// initialize line-coding details from global configuration
	acm.speed = configDeviceCDCACM[device.port].interfaceSpeed
	acm.lineCodingSize = configDeviceCDCACM[device.port].lineCodingSize
	acm.baudRate = configDeviceCDCACM[device.port].lineCodingBaudRate
	acm.charFormat = configDeviceCDCACM[device.port].lineCodingCharFormat
	acm.parityType = configDeviceCDCACM[device.port].lineCodingParityType
	acm.dataBits = configDeviceCDCACM[device.port].lineCodingDataBits

	// initialize remaining state data
	acm.alternate = 0xFF

	return statusSuccess
}

func (acm *deviceCDCACM) deinit() status {
	return statusSuccess
}

func (acm *deviceCDCACM) initEndpoints() status {

	// find the communication/control interface in our CDC-ACM configuration list
	acm.interfaceNumber, acm.comm = acm.findInterface(deviceCDCCommClass)

	// verify we found a valid communication/control interface
	if nil == acm.comm {
		return statusInvalidHandle
	}

	// initialize all of the communication/control input endpoints
	if s := acm.initInterfaceEndpoints(specIn, specEndpointInterrupt); !s.OK() {
		return s
	}

	// find the data interface in our CDC-ACM configuration list
	_, acm.data = acm.findInterface(deviceCDCDataClass)

	// verify we found a valid data interface
	if nil == acm.data {
		return statusInvalidHandle
	}

	// initialize all of the data input endpoints
	if s := acm.initInterfaceEndpoints(specIn, specEndpointBulk); !s.OK() {
		return s
	}
	// initialize all of the data output endpoints
	if s := acm.initInterfaceEndpoints(specOut, specEndpointBulk); !s.OK() {
		return s
	}

	return statusSuccess
}

func (acm *deviceCDCACM) initInterfaceEndpoints(direction, transferType uint8) status {

	// select the relevant fields and callbacks for our given interface type
	pipe, deviceInterface, endpointCallback :=
		acm.endpointInterface(direction, transferType)
	if nil == deviceInterface {
		return statusInvalidParameter
	}

	// initialize each endpoint with given direction and transfer type
	for _, ep := range deviceInterface.endpoint {
		num, dir := unpackEndpoint(ep.address)
		if dir == direction && ep.transferType == transferType {
			config := deviceEndpointConfig{
				maxPacketSize: ep.maxPacketSize,
				address:       ep.address,
				transferType:  ep.transferType,
				zlt:           0,
				interval:      ep.interval,
			}
			callback := deviceEndpointCallback{
				param:    acm,
				callback: endpointCallback,
			}
			*pipe = deviceCDCACMPipe{
				pipeDataBuffer: deviceCDCACMBufferInvalid32,
				pipeDataLen:    0,
				pipeStall:      false,
				ep:             num,
				isBusy:         false,
			}
			if s := acm.device.initEndpoint(config, callback); !s.OK() {
				return s
			}
		}
	}

	return statusSuccess
}

func (acm *deviceCDCACM) deinitEndpoints() status {

	if nil == acm.device {
		return statusInvalidHandle
	}

	if nil != acm.comm {
		for i := range acm.comm.endpoint {
			_ = acm.device.deinitEndpoint(acm.comm.endpoint[i].address)
		}
		acm.comm = nil
	}
	if nil != acm.data {
		for i := range acm.data.endpoint {
			_ = acm.device.deinitEndpoint(acm.data.endpoint[i].address)
		}
		acm.data = nil
	}

	return statusSuccess
}

func (acm *deviceCDCACM) event(event deviceClassEventID, param interface{}) (s status) {

	// assume success unless error condition deliberately detected
	s = statusSuccess

	switch event {
	case deviceClassEventDeviceReset:
		// bus reset, clear the selected configuration
		acm.config = nil

	case deviceClassEventSetConfiguration:
		if configuration, ok := param.(uint8); ok {
			// configuration index is 1-based, meaning configuration 0 is invalid
			if 0 == configuration || int(configuration) > len(acm.device.class.config) {
				return statusInvalidParameter
			}
			if &acm.device.class.config[configuration-1] == acm.config {
				break // configuration already selected
			}
			// de-initialize endpoints of current configuration
			if s = acm.deinitEndpoints(); !s.OK() {
				break
			}
			// select new configuration, reset alternate setting
			acm.config = &acm.device.class.config[configuration-1]
			acm.alternate = 0
			// initialize endpoints of new configuration
			if s = acm.initEndpoints(); !s.OK() {
				break
			}
		} else {
			// unexpected parameter, should be uint8 (configuration index)
			s = statusInvalidParameter
		}

	case deviceClassEventSetInterface:
		if interfaceAlternate, ok := param.(uint16); ok {
			alternate := uint8(interfaceAlternate & 0xFF)
			if acm.interfaceNumber != uint8(interfaceAlternate>>8) {
				break // requested alternate from interface different than current
			}
			if alternate == acm.alternate {
				break // requested alternate same as current
			}
			// de-initialize endpoints of current configuration
			if s = acm.deinitEndpoints(); !s.OK() {
				break
			}
			// select new alternate
			acm.alternate = alternate
			// initialize endpoints of new configuration
			if s = acm.initEndpoints(); !s.OK() {
				break
			}
		} else {
			// unexpected parameter, should be uint16: interface (hi), alternate (lo)
			s = statusInvalidParameter
		}

	case deviceClassEventSetEndpointHalt:
		// verify parameter and fields
		if address, ok := param.(uint8); ok {
			if nil != acm.config && nil != acm.comm && nil != acm.data {
				// check if given endpoint is a communication/control endpoint
				for _, e := range acm.comm.endpoint {
					if address == e.address {
						// found endpoint, set stall flag
						acm.interruptIn.pipeStall = true
						// notify device controller
						c := acm.device.controller.control(deviceControlEndpointStall, address)
						if s.OK() && !c.OK() {
							s = c
						}
					}
				}
				// check if given endpoint is a data endpoint
				for _, e := range acm.data.endpoint {
					if address == e.address {
						_, direction := unpackEndpoint(address)
						// found endpoint, set stall flag
						if specIn == direction {
							acm.bulkIn.pipeStall = true
						} else {
							acm.bulkOut.pipeStall = true
						}
						// notify device controller
						c := acm.device.controller.control(deviceControlEndpointStall, address)
						if s.OK() && !c.OK() {
							s = c
						}
					}
				}
			} else {
				s = statusInvalidHandle
			}
		} else {
			// unexpected parameter, should be uint8 (endpoint address)
			s = statusInvalidParameter
		}

	case deviceClassEventClearEndpointHalt:
		// verify parameter and fields
		if address, ok := param.(uint8); ok {
			if nil != acm.config && nil != acm.comm && nil != acm.data {
				// check if given endpoint is a communication/control endpoint
				for _, e := range acm.comm.endpoint {
					if address == e.address {
						// found endpoint, notify device controller
						c := acm.device.controller.control(deviceControlEndpointUnstall, address)
						if s.OK() && !c.OK() {
							s = c
						}
						_, direction := unpackEndpoint(address)
						if specIn == direction {
							// flush any buffered data written to the stalled endpoint
							if acm.interruptIn.pipeStall {
								// clear stall flag
								acm.interruptIn.pipeStall = false
								// verify the buffer has valid data
								if !bytes.Equal(acm.interruptIn.pipeDataBuffer, deviceCDCACMBufferInvalid32) {
									// transmit
									u := acm.device.controller.send(
										acm.interruptIn.ep,
										acm.interruptIn.pipeDataBuffer,
										acm.interruptIn.pipeDataLen)
									if !u.OK() {
										// notify upper layer driver of communication/control event
										_ = acm.interruptInEvent(
											deviceEndpointCallbackMessage{
												buffer:  acm.interruptIn.pipeDataBuffer,
												length:  acm.interruptIn.pipeDataLen,
												isSetup: false,
											},
											acm,
										)
										if s.OK() {
											s = u
										}
									}
									// clear the stalled endpoint buffer
									acm.interruptIn.pipeDataBuffer = deviceCDCACMBufferInvalid32
									acm.interruptIn.pipeDataLen = 0
								}
							}
						}
					}
				}
				// check if given endpoint is a data endpoint
				for _, e := range acm.data.endpoint {
					if address == e.address {
						// found endpoint, notify device controller
						c := acm.device.controller.control(deviceControlEndpointUnstall, address)
						if s.OK() && !c.OK() {
							s = c
						}
						// check if endpoint is an input or output
						_, direction := unpackEndpoint(address)
						if specIn == direction {
							// flush any buffered data written to the stalled endpoint
							if acm.bulkIn.pipeStall {
								// clear stall flag
								acm.bulkIn.pipeStall = false
								// verify the buffer has valid data
								if !bytes.Equal(acm.bulkIn.pipeDataBuffer, deviceCDCACMBufferInvalid32) {
									// transmit
									u := acm.device.controller.send(
										acm.bulkIn.ep,
										acm.bulkIn.pipeDataBuffer,
										acm.bulkIn.pipeDataLen)
									if !u.OK() {
										// notify upper layer driver of data input event
										_ = acm.bulkInEvent(
											deviceEndpointCallbackMessage{
												buffer:  acm.bulkIn.pipeDataBuffer,
												length:  acm.bulkIn.pipeDataLen,
												isSetup: false,
											},
											acm,
										)
										if s.OK() {
											s = u
										}
									}
									// clear the stalled endpoint buffer
									acm.bulkIn.pipeDataBuffer = deviceCDCACMBufferInvalid32
									acm.bulkIn.pipeDataLen = 0
								}
							}
						} else {
							// flush any buffered data read from the stalled endpoint
							if acm.bulkOut.pipeStall {
								// clear stall flag
								acm.bulkOut.pipeStall = false
								// verify the buffer has valid data
								if !bytes.Equal(acm.bulkOut.pipeDataBuffer, deviceCDCACMBufferInvalid32) {
									// receive
									u := acm.device.controller.receive(
										acm.bulkOut.ep,
										acm.bulkOut.pipeDataBuffer,
										acm.bulkOut.pipeDataLen)
									if !u.OK() {
										// notify upper layer driver of data output event
										_ = acm.bulkOutEvent(
											deviceEndpointCallbackMessage{
												buffer:  acm.bulkOut.pipeDataBuffer,
												length:  acm.bulkOut.pipeDataLen,
												isSetup: false,
											},
											acm,
										)
										if s.OK() {
											s = u
										}
									}
									// clear the stalled endpoint buffer
									acm.bulkOut.pipeDataBuffer = deviceCDCACMBufferInvalid32
									acm.bulkOut.pipeDataLen = 0
								}
							}
						}
					}
				}
			} else {
				s = statusInvalidHandle
			}
		} else {
			// unexpected parameter, should be uint8 (endpoint address)
			s = statusInvalidParameter
		}
	case deviceClassEventClassRequest:
		// verify parameter and fields
		if request, ok := param.(deviceControlRequest); ok {
			// verify requested interface is receiver's interface and request is CDC
			if (request.setup.wIndex&0xFF) != uint16(acm.interfaceNumber) ||
				(request.setup.bmRequestType&specRequestTypeTypeMsk) != specRequestTypeTypeClass {
				// construct standard parameter to be passed on to upper layer drivevr
				param := deviceCDCACMRequestParam{
					buffer:         &request.buffer,
					length:         &request.length,
					interfaceIndex: request.setup.wIndex,
					setupValue:     request.setup.wValue,
					isSetup:        request.isSetup,
				}
				// translate request code to event code
				var event deviceClassEventID
				switch request.setup.bRequest {
				case deviceCDCRequestSendEncapsulatedCommand:
					event = deviceCDCACMEventSendEncapsulatedCommand
				case deviceCDCRequestGetEncapsulatedResponse:
					event = deviceCDCACMEventGetEncapsulatedResponse
				case deviceCDCRequestSetCommFeature:
					event = deviceCDCACMEventSetCommFeature
				case deviceCDCRequestGetCommFeature:
					event = deviceCDCACMEventGetCommFeature
				case deviceCDCRequestClearCommFeature:
					event = deviceCDCACMEventClearCommFeature
				case deviceCDCRequestGetLineCoding:
					event = deviceCDCACMEventGetLineCoding
				case deviceCDCRequestSetLineCoding:
					event = deviceCDCACMEventSetLineCoding
				case deviceCDCRequestSetControlLineState:
					event = deviceCDCACMEventSetControlLineState
				case deviceCDCRequestSendBreak:
					event = deviceCDCACMEventSendBreak
				default:
					s = statusInvalidRequest
				}
				// notify request to upper layer driver
				s = acm.config.driver.event(event, param)
			} else {
				s = statusInvalidRequest
			}
		} else {
			s = statusInvalidParameter
		}
	}

	return s
}

func (acm *deviceCDCACM) send(ep uint8, buffer []uint8, length uint32) status {
	var pipe *deviceCDCACMPipe
	// select which endpoint pipe to write into
	switch ep {
	case acm.interruptIn.ep:
		pipe = &acm.interruptIn
	case acm.bulkIn.ep:
		pipe = &acm.bulkIn
	default:
		return statusInvalidParameter
	}
	if pipe.isBusy {
		return statusBusy
	}
	// set pipe busy flag
	pipe.isBusy = true
	// check if pipe stall flag is set
	if pipe.pipeStall {
		// write data to pipe buffer
		pipe.pipeDataBuffer = buffer
		pipe.pipeDataLen = length
		return statusSuccess
	}
	// pass data to device controller for transmission
	if s := acm.device.controller.send(ep, buffer, length); !s.OK() {
		pipe.isBusy = false
		return s
	}
	return statusSuccess
}

func (acm *deviceCDCACM) receive(ep uint8, buffer []uint8, length uint32) status {
	var pipe *deviceCDCACMPipe
	// select which endpoint pipe to read from
	switch ep {
	case acm.bulkOut.ep:
		pipe = &acm.bulkOut
	default:
		return statusInvalidParameter
	}
	if pipe.isBusy {
		return statusBusy
	}
	// set pipe busy flag
	pipe.isBusy = true
	// check if pipe stall flag is set
	if pipe.pipeStall {
		// write data to pipe buffer
		pipe.pipeDataBuffer = buffer
		pipe.pipeDataLen = length
		return statusSuccess
	}
	// pass data to device controller for reception
	if s := acm.device.controller.receive(ep, buffer, length); !s.OK() {
		pipe.isBusy = false
		return s
	}
	return statusSuccess
}

// findInterface returns the interface number and deviceInterface from the
// receiver's CDC-ACM configuration list whose classCode and alternateSetting
// equals the given classCode and receiver's current alternateSetting;
// otherwise, no such interface is found, returns 0 and nil.
func (acm *deviceCDCACM) findInterface(classCode uint8) (uint8, *deviceInterface) {
	if nil == acm.device || nil == acm.config {
		return 0, nil
	}
	for c := range acm.config.info.interfaceList {
		dci := &acm.config.info.interfaceList[c]
		if classCode == dci.classCode {
			for d := range dci.deviceInterface {
				if acm.alternate == dci.deviceInterface[d].alternateSetting {
					return dci.interfaceNumber, &dci.deviceInterface[d]
				}
			}
		}
	}
	return 0, nil
}

func (acm *deviceCDCACM) endpointInterface(direction, transferType uint8) (
	*deviceCDCACMPipe, *deviceInterface, deviceEndpointCallbackFunc,
) {
	switch transferType {
	case specEndpointInterrupt:
		switch direction {
		case specIn:
			return &acm.interruptIn, acm.comm, acm.interruptInEvent
		}
	case specEndpointBulk:
		switch direction {
		case specIn:
			return &acm.bulkIn, acm.data, acm.bulkInEvent
		case specOut:
			return &acm.bulkOut, acm.data, acm.bulkOutEvent
		}
	}
	return nil, nil, nil
}

func (acm *deviceCDCACM) interruptInEvent(message deviceEndpointCallbackMessage, param interface{}) status {
	if a, ok := param.(*deviceCDCACM); ok {
		a.interruptIn.isBusy = false
		if nil != a.config && nil != a.config.driver {
			a.config.driver.event(deviceCDCACMEventSerialStateNotify, message)
			return statusSuccess
		}
	}
	return statusInvalidParameter
}

func (acm *deviceCDCACM) bulkInEvent(message deviceEndpointCallbackMessage, param interface{}) status {
	if a, ok := param.(*deviceCDCACM); ok {
		a.bulkIn.isBusy = false
		if nil != a.config && nil != a.config.driver {
			a.config.driver.event(deviceCDCACMEventSendResponse, message)
			return statusSuccess
		}
	}
	return statusInvalidParameter
}

func (acm *deviceCDCACM) bulkOutEvent(message deviceEndpointCallbackMessage, param interface{}) status {
	if a, ok := param.(*deviceCDCACM); ok {
		a.bulkOut.isBusy = false
		if nil != a.config && nil != a.config.driver {
			a.config.driver.event(deviceCDCACMEventRecvResponse, message)
			return statusSuccess
		}
	}
	return statusInvalidParameter
}
