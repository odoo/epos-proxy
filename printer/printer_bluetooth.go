package printer

import (
	"epos-proxy/bluetooth"
	"epos-proxy/logger"
	"fmt"
	"time"
)

func newBlueToothPrinter(address string) *Printer {
	p := &Printer{
		connectionType:   PrinterTypeBluetooth,
		bluetoothAddress: address,
		jobs:             make(chan Job, QueueSize),
	}
	logger.Debugf("Created new Bluetooth printer instance for address: %s", address)
	go p.loop()
	return p
}

func (p *Printer) ensureOpenBluetoothLocked() error {
	if p.btConn != nil {
		logger.Debugf("BT printer %s already connected", p.idToString())
		return nil
	}

	conn, err := bluetooth.BTManager.Dial(p.bluetoothAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to BT printer %s: %w", p.bluetoothAddress, err)
	}

	p.btConn = conn

	if devPath, channel, ok := bluetooth.BTManager.GetCachedBinding(p.bluetoothAddress); ok {
		p.btDevPath = devPath
		logger.Infof("BT printer %s connected via %s (channel %d)",
			p.bluetoothAddress, devPath, channel)
	} else {
		logger.Infof("BT printer %s connected (raw socket)", p.bluetoothAddress)
	}

	return nil
}

func (p *Printer) writeBluetooth(data []byte) error {
	if err := p.btConn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
		p.closeDeviceLocked()
		return fmt.Errorf("failed to set write deadline for BT printer %s: %w", p.idToString(), err)
	}
	if _, err := p.btConn.Write(data); err != nil {
		p.closeDeviceLocked()
		return fmt.Errorf("failed to write to BT printer %s: %w", p.idToString(), err)
	}
	logger.Debugf("Successfully wrote to BT printer %s", p.idToString())
	return nil
}
