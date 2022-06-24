//go:build usb.cdc && mimxrt1062
// +build usb.cdc,mimxrt1062

package usb

import "unsafe"

// endpointQueueHead returns the queue head for the given endpoint address,
// encoded as direction D and endpoint number N with the 8-bit mask DxxxNNNN.
//
//go:inline
func (d *dhw) endpointQueueHead(endpoint uint8) *dhwEndpoint {
	return &descCDC[d.cc.config-1].qh[endpointIndex(endpoint)]
}

// transferControl returns the data and ackowledgement transfer descriptors for
// the control endpoint (i.e., endpoint 0).
//
//go:inline
func (d *dhw) transferControl() (dat, ack *dhwTransfer) {
	return descCDC[d.cc.config-1].cd, descCDC[d.cc.config-1].ad
}

//go:inline
func (d *dhw) controlStatusBuffer(data []uint8) uintptr {
	// reference to class configuration data
	c := descCDC[d.cc.config-1]
	for i := range c.cx {
		c.cx[i] = 0 // zero out the control reply buffer
	}
	// copy the given data into control reply buffer
	copy(c.cx[:], data)
	return uintptr(unsafe.Pointer(&c.cx[0]))
}

// =============================================================================
//  [CDC-ACM] Serial UART (Virtual COM Port)
// =============================================================================

func (d *dhw) cdcConfigure() {
	acm := &descCDC[d.cc.config-1]

	acm.setState(descCDCStateConfigured)

	switch d.speed {
	case HighSpeed, SuperSpeed, DualSuperSpeed:
		acm.rxSize = descCDCDataRxHSPacketSize
		acm.txSize = descCDCDataTxHSPacketSize
	default:
		fallthrough
	case LowSpeed, FullSpeed:
		acm.rxSize = descCDCDataRxFSPacketSize
		acm.txSize = descCDCDataTxFSPacketSize
	}
	acm.txHead = 0
	acm.txFree = 0
	acm.txPrev = false
	acm.rxHead = 0
	acm.rxTail = 0
	acm.rxFree = 0

	d.endpointEnable(descCDCEndpointStatus,
		false, descCDCConfigAttrStatus)
	d.endpointEnable(descCDCEndpointDataRx,
		false, descCDCConfigAttrDataRx)
	d.endpointEnable(descCDCEndpointDataTx,
		false, descCDCConfigAttrDataTx)

	d.endpointConfigureTx(descCDCEndpointStatus,
		acm.sxSize, false, nil)
	d.endpointConfigureRx(descCDCEndpointDataRx,
		acm.rxSize, false, d.cdcNotify)
	d.endpointConfigureTx(descCDCEndpointDataTx,
		acm.txSize, true, nil)
	for i := range acm.rd {
		d.cdcReceive(uint8(i))
	}
	d.timerConfigure(0, descCDCTxSyncUs, d.cdcSync)
}

func (d *dhw) cdcSetLineState(state uint16) {
	acm := &descCDC[d.cc.config-1]
	acm.setState(descCDCStateLineState)
	if acm.ls.parse(state) {
		// TBD: respond to changes in line state?
	}
}

func (d *dhw) cdcSetLineCoding(coding []uint8) {
	acm := &descCDC[d.cc.config-1]
	acm.setState(descCDCStateLineCoding)
	if acm.lc.parse(coding) {
		switch acm.lc.baud {
		case 134:
			// if acm.ls.dataTerminalReady {
			// reboot CPU
			// }
			d.enableSOF(true, descCDCInterfaceCount)
		}
	}
}

func (d *dhw) cdcReady() bool {
	acm := &descCDC[d.cc.config-1]
	// Ensure we have received SET_CONFIGURATION class request, and then both
	// SET_LINE_STATE and SET_LINE_CODING CDC requests (in that order).
	return d.state() == dcdStateConfigured &&
		acm.st.Get() == uint8(descCDCStateLineCoding)
}

func (d *dhw) cdcReceive(endpoint uint8) {
	acm := &descCDC[d.cc.config-1]
	num := uint16(endpoint) & descEndptAddrNumberMsk
	buf := &acm.rx[num*descCDCRxSize]
	d.enableInterrupts(false)
	d.transferPrepare(&acm.rd[num], buf, acm.rxSize, uint32(endpoint))
	deleteCache(uintptr(unsafe.Pointer(buf)), uintptr(acm.rxSize))
	d.endpointReceive(descCDCEndpointDataRx, &acm.rd[num])
	d.enableInterrupts(true)
}

func (d *dhw) cdcNotify(transfer *dhwTransfer) {
	acm := &descCDC[d.cc.config-1]
	len := acm.rxSize - (uint16(transfer.token>>16) & 0x7FFF)
	p := transfer.param
	if 0 == len {
		// zero-length packet (ZLP)
		d.cdcReceive(uint8(p))
	} else {
		// data packet
		h := acm.rxHead
		if h != acm.rxTail {
			// previous packet is still buffered
			q := acm.rxQueue[h]
			n := acm.rxCount[q]
			if len <= descCDCRxSize-n {
				// previous buffer has enough free space for this packet's data
				_ = copy(acm.rx[q*descCDCRxSize+n:],
					acm.rx[p*descCDCRxSize:uint16(p)*descCDCRxSize+len])
				acm.rxCount[q] = n + len
				acm.rxFree += len
				d.cdcReceive(uint8(p))
				return
			}
		}
		// add this packet to Rx buffer
		acm.rxCount[p] = len
		acm.rxIndex[p] = 0
		h += 1
		if h > descCDCRDCount { // should be >=
			h = 0
		}
		acm.rxQueue[h] = uint16(p)
		acm.rxHead = h
		acm.rxFree += len
	}
}

// uartFlush discards all buffered input (Rx) data.
func (d *dhw) cdcFlush() {
	acm := &descCDC[d.cc.config-1]
	tail := acm.rxTail
	for tail != acm.rxHead {
		tail += 1
		if tail > descCDCRDCount {
			tail = 0
		}
		i := acm.rxQueue[tail]
		acm.rxFree -= acm.rxCount[i] - acm.rxIndex[i]
		d.cdcReceive(uint8(i))
		acm.rxTail = tail
	}
}

func (d *dhw) cdcAvailable() int {
	return int(descCDC[d.cc.config-1].rxFree)
}

func (d *dhw) cdcPeek() (uint8, bool) {
	acm := &descCDC[d.cc.config-1]
	tail := acm.rxTail
	if tail == acm.rxHead {
		return 0, false
	}
	tail += 1
	if tail > descCDCRDCount {
		tail = 0
	}
	i := acm.rxQueue[tail]
	return acm.rx[i*descCDCRxSize+acm.rxIndex[i]], true
}

func (d *dhw) cdcReadByte() (uint8, bool) {
	b := []uint8{0}
	n, err := d.cdcRead(b)
	return b[0], n > 0 && err == nil
}

func (d *dhw) cdcRead(data []uint8) (int, error) {
	acm := &descCDC[d.cc.config-1]
	read := uint16(0)
	size := uint16(len(data))
	tail := acm.rxTail
	dest := uint16(0)
	d.enableInterrupts(false)
	for read < size && tail != acm.rxHead {
		tail += 1
		if tail > descCDCRDCount {
			tail = 0
		}
		i := acm.rxQueue[tail]
		count := uint16(size - read)
		avail := acm.rxCount[i] - acm.rxIndex[i]
		start := i*descCDCRxSize + acm.rxIndex[i]
		if avail > count {
			// partially consume packet
			_ = copy(data[dest:], acm.rx[start:start+count])
			acm.rxFree -= count
			acm.rxIndex[i] += count
			read += count
		} else {
			// fully consume packet
			_ = copy(data[dest:], acm.rx[start:start+avail])
			dest += avail //* uint16(unsafe.Sizeof(&data[0]))
			read += avail
			acm.rxFree -= avail
			acm.rxTail = tail
			d.cdcReceive(uint8(i))
		}
	}
	d.enableInterrupts(true)
	return int(read), nil
}

func (d *dhw) cdcWriteByte(c uint8) error {
	_, err := d.cdcWrite([]uint8{c})
	return err
}

func (d *dhw) cdcWrite(data []uint8) (int, error) {
	acm := &descCDC[d.cc.config-1]
	sent := 0
	size := len(data)
	for size > 0 {
		xfer := &acm.td[acm.txHead]
		wait := false
		when := int64(0)
		for 0 == acm.txFree {
			if 0 == xfer.token&0x80 {
				if 0 != xfer.token&0x68 {
					// TODO: token contains error, how to handle?
				}
				acm.txFree = descCDCTxSize
				acm.txPrev = false
				break
			}
			if !wait {
				wait = true
				when = ticks()
			}
			if acm.txPrev {
				return sent, nil
			}
			if ticks()-when > descCDCTxTimeoutMs {
				acm.txPrev = true
				return sent, nil
			}
		}
		buff := acm.tx[(int(acm.txHead)*descCDCTxSize)+
			(descCDCTxSize-int(acm.txFree)):]
		if size > int(acm.txFree) {
			_ = copy(buff, data[sent:sent+int(acm.txFree)])
			tx := &acm.tx[int(acm.txHead)*descCDCTxSize]
			d.transferPrepare(xfer, tx, descCDCTxSize, 0)
			flushCache(uintptr(unsafe.Pointer(tx)), descCDCTxSize)
			d.endpointTransmit(descCDCEndpointDataTx, xfer)
			acm.txHead += 1
			if acm.txHead >= descCDCTDCount {
				acm.txHead = 0
			}
			size -= int(acm.txFree)
			sent += int(acm.txFree)
			acm.txFree = 0
			d.timerStop(0)
		} else {
			_ = copy(buff, data[:size])
			acm.txFree -= uint16(size)
			sent += size
			size = 0
			d.timerOneShot(0)
		}
	}
	return sent, nil
}

func (d *dhw) cdcSync() {
	const autoFlushTx = true
	if !autoFlushTx {
		return
	}
	acm := &descCDC[d.cc.config-1]
	if 0 == acm.txFree {
		return
	}
	xfer := &acm.td[acm.txHead]
	buff := &acm.tx[uint16(acm.txHead)*descCDCTxSize]
	size := descCDCTxSize - acm.txFree
	d.transferPrepare(xfer, buff, size, 0)
	flushCache(uintptr(unsafe.Pointer(buff)), uintptr(size))
	d.endpointTransmit(descCDCEndpointDataTx, xfer)
	acm.txHead += 1
	if acm.txHead >= descCDCTDCount {
		acm.txHead = 0
	}
	acm.txFree = 0
}
