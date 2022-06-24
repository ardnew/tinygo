//go:build usb.hid && mimxrt1062
// +build usb.hid,mimxrt1062

package usb

import "unsafe"

// endpointQueueHead returns the queue head for the given endpoint address,
// encoded as direction D and endpoint number N with the 8-bit mask DxxxNNNN.
//
//go:inline
func (d *dhw) endpointQueueHead(endpoint uint8) *dhwEndpoint {
	return &descHID[d.cc.config-1].qh[endpointIndex(endpoint)]
}

// transferControl returns the data and ackowledgement transfer descriptors for
// the control endpoint (i.e., endpoint 0).
//
//go:inline
func (d *dhw) transferControl() (dat, ack *dhwTransfer) {
	return descHID[d.cc.config-1].cd, descHID[d.cc.config-1].ad
}

//go:inline
func (d *dhw) controlStatusBuffer(data []uint8) uintptr {
	// reference to class configuration data
	c := descHID[d.cc.config-1]
	for i := range c.cx {
		c.cx[i] = 0 // zero out the control reply buffer
	}
	// copy the given data into control reply buffer
	copy(c.cx[:], data)
	return uintptr(unsafe.Pointer(&c.cx[0]))
}

// =============================================================================
//  [HID] Serial
// =============================================================================

func (d *dhw) serialConfigure() {
	hid := &descHID[d.cc.config-1]

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.rxSerialSize = descHIDSerialRxHSPacketSize
		hid.txSerialSize = descHIDSerialTxHSPacketSize
	default:
		fallthrough
	case LowSpeed, FullSpeed:
		hid.rxSerialSize = descHIDSerialRxFSPacketSize
		hid.txSerialSize = descHIDSerialTxFSPacketSize
	}

	// Rx and Tx are on same endpoint
	d.endpointEnable(descHIDEndpointSerialRx,
		false, descHIDConfigAttrSerial)

	d.endpointConfigureRx(descHIDEndpointSerialRx,
		hid.rxSerialSize, false, d.serialNotify)
	d.endpointConfigureTx(descHIDEndpointSerialTx,
		hid.txSerialSize, false, nil)
	for i := range hid.rdSerial {
		d.serialReceive(uint8(i))
	}
	d.timerConfigure(0, descHIDSerialTxSyncUs, d.serialSync)
}

func (d *dhw) serialReceive(endpoint uint8) {
	hid := &descHID[d.cc.config-1]
	num := uint16(endpoint) & descEndptAddrNumberMsk
	buf := &hid.rxSerial[num*descHIDSerialRxSize]
	d.enableInterrupts(false)
	d.transferPrepare(&hid.rdSerial[num], buf, hid.rxSerialSize, uint32(endpoint))
	deleteCache(uintptr(unsafe.Pointer(buf)), uintptr(hid.rxSerialSize))
	d.endpointReceive(descHIDEndpointSerialRx, &hid.rdSerial[num])
	d.enableInterrupts(true)
}

func (d *dhw) serialTransmit() {
	hid := &descHID[d.cc.config-1]
	xfer := &hid.tdSerial[hid.txSerialHead]
	buff := &hid.txSerial[uint16(hid.txSerialHead)*descHIDSerialTxSize]
	d.transferPrepare(xfer, buff, hid.txSerialSize, 0)
	flushCache(uintptr(unsafe.Pointer(buff)), uintptr(hid.txSerialSize))
	d.endpointTransmit(descHIDEndpointSerialTx, xfer)
	hid.txSerialHead += 1
	if hid.txSerialHead >= descHIDSerialTDCount {
		hid.txSerialHead = 0
	}
}

func (d *dhw) serialNotify(transfer *dhwTransfer) {
	hid := &descHID[d.cc.config-1]
	len := hid.rxSerialSize - (uint16(transfer.token>>16) & 0x7FFF)
	p := transfer.param
	if len == hid.rxSerialSize && 0 != hid.rxSerial[p*uint32(hid.rxSerialSize)] {
		// data packet
		hid.rxSerialIndex[p] = 0
		h := hid.rxSerialHead + 1
		if h > descHIDSerialRDCount { // should be >=
			h = 0
		}
		hid.rxSerialQueue[h] = uint16(p)
		hid.rxSerialHead = h
		hid.rxSerialFree += len
	} else {
		// short packet
		d.serialReceive(uint8(p))
	}
}

// serialFlush discards all buffered input (Rx) data.
func (d *dhw) serialFlush() {
	hid := &descHID[d.cc.config-1]
	tail := hid.rxSerialTail
	for tail != hid.rxSerialHead {
		tail += 1
		if tail > descHIDSerialRDCount {
			tail = 0
		}
		i := hid.rxSerialQueue[tail]
		d.serialReceive(uint8(i))
		hid.rxSerialTail = tail
	}
}

func (d *dhw) serialSync() {
	const autoFlushTx = true
	if !autoFlushTx {
		return
	}
	hid := &descHID[d.cc.config-1]
	if 0 == hid.txSerialFree {
		return
	}
	xfer := &hid.tdSerial[hid.txSerialHead]
	buff := &hid.txSerial[uint16(hid.txSerialHead)*descHIDSerialTxSize]
	size := descHIDSerialTxSize - hid.txSerialFree
	d.transferPrepare(xfer, buff, size, 0)
	flushCache(uintptr(unsafe.Pointer(buff)), uintptr(size))
	d.endpointTransmit(descHIDEndpointSerialTx, xfer)
	hid.txSerialHead += 1
	if hid.txSerialHead >= descHIDSerialTDCount {
		hid.txSerialHead = 0
	}
	hid.txSerialFree = 0
}

// =============================================================================
//  [HID] Keyboard
// =============================================================================

func (d *dhw) keyboard() *Keyboard { return descHID[d.cc.config-1].keyboard }

func (d *dhw) keyboardConfigure() {
	hid := &descHID[d.cc.config-1]

	// Initialize keyboard
	hid.keyboard.configure(d.dcd, hid)

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.txKeyboardSize = descHIDKeyboardTxHSPacketSize
	default:
		fallthrough
	case LowSpeed, FullSpeed:
		hid.txKeyboardSize = descHIDKeyboardTxFSPacketSize
	}

	d.endpointEnable(descHIDEndpointKeyboard,
		false, descHIDConfigAttrKeyboard)
	d.endpointEnable(descHIDEndpointMediaKey,
		false, descHIDConfigAttrMediaKey)

	d.endpointConfigureTx(descHIDEndpointKeyboard,
		hid.txKeyboardSize, false, nil)
	d.endpointConfigureTx(descHIDEndpointMediaKey,
		hid.txKeyboardSize, false, nil)
}

func (d *dhw) keyboardSendKeys(consumer bool) bool {
	hid := &descHID[d.cc.config-1]

	if !consumer {

		hid.tpKeyboard[0] = hid.keyboard.mod
		hid.tpKeyboard[1] = 0
		hid.tpKeyboard[2] = hid.keyboard.key[0]
		hid.tpKeyboard[3] = hid.keyboard.key[1]
		hid.tpKeyboard[4] = hid.keyboard.key[2]
		hid.tpKeyboard[5] = hid.keyboard.key[3]
		hid.tpKeyboard[6] = hid.keyboard.key[4]
		hid.tpKeyboard[7] = hid.keyboard.key[5]

		return d.keyboardWrite(descHIDEndpointKeyboard, hid.tpKeyboard[:])

	} else {

		// 44444444 44333333 33332222 22222211 11111111  [ word ]
		// 98765432 10987654 32109876 54321098 76543210  [ index ]  (right-to-left)

		hid.tpKeyboard[1] = uint8((hid.keyboard.con[1] << 2) | ((hid.keyboard.con[0] >> 8) & 0x03))
		hid.tpKeyboard[2] = uint8((hid.keyboard.con[2] << 4) | ((hid.keyboard.con[1] >> 6) & 0x0F))
		hid.tpKeyboard[3] = uint8((hid.keyboard.con[3] << 6) | ((hid.keyboard.con[2] >> 4) & 0x3F))
		hid.tpKeyboard[4] = uint8(hid.keyboard.con[3] >> 2)
		hid.tpKeyboard[5] = hid.keyboard.sys[0]
		hid.tpKeyboard[6] = hid.keyboard.sys[1]
		hid.tpKeyboard[7] = hid.keyboard.sys[2]

		return d.keyboardWrite(descHIDEndpointMediaKey, hid.tpKeyboard[:])

	}
}

func (d *dhw) keyboardWrite(endpoint uint8, data []uint8) bool {
	hid := &descHID[d.cc.config-1]

	size := uint16(len(data))
	xfer := &hid.tdKeyboard[hid.txKeyboardHead]
	when := ticks()
	for {
		if 0 == xfer.token&0x80 {
			if 0 != xfer.token&0x68 {
				// TODO: token contains error, how to handle?
			}
			hid.txKeyboardPrev = false
			break
		}
		if hid.txKeyboardPrev {
			return false
		}
		if ticks()-when > descHIDKeyboardTxTimeoutMs {
			// Waited too long, assume host connection dropped
			hid.txKeyboardPrev = true
			return false
		}
	}
	// Without this delay, the order packets are transmitted is seriously screwy.
	udelay(60)
	buff := hid.txKeyboard[hid.txKeyboardHead*descHIDKeyboardTxSize:]
	_ = copy(buff, data)
	d.transferPrepare(xfer, &buff[0], size, 0)
	flushCache(uintptr(unsafe.Pointer(&buff[0])), descHIDKeyboardTxSize)
	d.endpointTransmit(endpoint, xfer)
	hid.txKeyboardHead += 1
	if hid.txKeyboardHead >= descHIDKeyboardTDCount {
		hid.txKeyboardHead = 0
	}
	return true
}

// =============================================================================
//  [HID] Mouse
// =============================================================================

func (d *dhw) mouseConfigure() {
	hid := &descHID[d.cc.config-1]

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.txMouseSize = descHIDMouseTxHSPacketSize
	default:
		fallthrough
	case LowSpeed, FullSpeed:
		hid.txMouseSize = descHIDMouseTxFSPacketSize
	}

	d.endpointEnable(descHIDEndpointMouse,
		false, descHIDConfigAttrMouse)

	d.endpointConfigureTx(descHIDEndpointMouse,
		hid.txMouseSize, false, nil)
}

// =============================================================================
//  [HID] Joystick
// =============================================================================

func (d *dhw) joystickConfigure() {
	hid := &descHID[d.cc.config-1]

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		hid.txJoystickSize = descHIDJoystickTxHSPacketSize
	default:
		fallthrough
	case LowSpeed, FullSpeed:
		hid.txJoystickSize = descHIDJoystickTxFSPacketSize
	}

	d.endpointEnable(descHIDEndpointJoystick,
		false, descHIDConfigAttrJoystick)

	d.endpointConfigureTx(descHIDEndpointJoystick,
		hid.txJoystickSize, false, nil)
}
