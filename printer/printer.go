package printer

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"epos-proxy/logger"

	"github.com/google/gousb"
)

func newPrinter(rawPrinter RawPrinter) *Printer {
	// Check if this is a LAN printer
	if lanIP, ok := DecodeLANPrinterID(rawPrinter.PrinterIp); ok {
		p := &Printer{
			connectionType: PrinterTypeLAN,
			lanIP:          lanIP,
			jobs:           make(chan Job, QueueSize),
		}
		go p.loop()
		return p
	}

	var printerID *PrinterID
	if rawPrinter.PrinterIp != "" {
		var err error
		printerID, err = decodePrinterID(rawPrinter.PrinterIp)
		if err != nil {
			logger.Warnf("failed to decode printer ID %q: %v", rawPrinter.PrinterIp, err)
		}
	}

	var idName string
	if printerID != nil {
		idName = printerID.IdName
	}

	// USB printer
	p := &Printer{
		connectionType: PrinterTypeUSB,
		id:             printerID,
		jobs:           make(chan Job, QueueSize),
		idName:         idName,
		Category:       rawPrinter.Category,
	}

	if p.Category == PrinterOffice && idName == "" {
		if err := p.fetchSystemPrinterName(); err != nil {
			logger.Errorf("Error: %v", err)
		}
	}

	logger.Debugf("Created new USB printer instance for ID: %s", p.idToString())
	go p.loop()
	return p
}

func (p *Printer) Enqueue(fn JobFunc, reply chan JobResult) error {
	j := Job{run: fn, reply: reply}
	select {
	case p.jobs <- j:
		logger.Debugf("Enqueued print job for printer %s", p.idToString())
		return nil
	default:
		logger.Warnf("Printer queue full for printer %s", p.idToString())
		return ErrQueueFull
	}
}

func (p *Printer) Write(data []byte) error {
	if err := p.ensureOpen(); err != nil {
		return err
	}

	if p.Category == PrinterOffice {
		return p.printViaSystemPrinter(data)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	logger.Debugf("Writing %d bytes to printer %s", len(data), p.idToString())

	if p.connectionType == PrinterTypeLAN {
		if err := p.tcpConn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
			p.closeDeviceLocked()
			return fmt.Errorf("failed to set write deadline for LAN printer %s: %w", p.idToString(), err)
		}
		if _, err := p.tcpConn.Write(data); err != nil {
			p.closeDeviceLocked()
			return fmt.Errorf("failed to write to LAN printer %s: %w", p.idToString(), err)
		}
		logger.Debugf("Successfully wrote to LAN printer %s", p.idToString())
		return nil
	}

	// USB write
	ctx, cancel := context.WithTimeout(context.Background(), WriteTimeout)
	defer cancel()
	logger.Debugf("Writing to USB printer %s with timeout %v", p.idToString(), WriteTimeout)

	if _, err := p.outEndpoint.WriteContext(ctx, data); err != nil {
		p.closeDeviceLocked()
		return fmt.Errorf("failed to write to USB printer %s: %w", p.idToString(), err)
	}
	return nil
}

func (p *Printer) loop() {
	logger.Debugf("Printer loop started for %s with %d jobs", p.idToString(), len(p.jobs))
	for j := range p.jobs {
		result := j.run(p)
		if j.reply != nil {
			j.reply <- result
			close(j.reply)
		}
		if len(p.jobs) == 0 {
			p.close()
		}
	}
}

const pdfNetworkPrefix = "PDF_NETWORK_"

func (p *Printer) ensureOpen() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Category == PrinterOffice {
		if strings.HasPrefix(p.idName, pdfNetworkPrefix) {
			p.lanIP = strings.TrimPrefix(p.idName, pdfNetworkPrefix)
			if err := p.ensureOpenLANLocked(); err != nil {
				return fmt.Errorf("printer is not open: %w", err)
			}
		}
		return p.ensureSystemPrinterOpen()
	}

	if p.connectionType == PrinterTypeLAN {
		return p.ensureOpenLANLocked()
	}
	return p.ensureOpenUSBLocked()
}

func (p *Printer) ensureOpenLANLocked() error {
	if p.tcpConn != nil {
		logger.Debugf("LAN printer %s already connected", p.idToString())
		return nil // already connected
	}

	addr := fmt.Sprintf("%s:%d", p.lanIP, LANPort)
	logger.Debugf("Attempting to connect to LAN printer %s at %s", p.idToString(), addr)
	conn, err := net.DialTimeout("tcp", addr, LANConnectTimeout)
	if err != nil {
		logger.Errorf("Failed to connect to LAN printer %v at %s: %v", p, addr, err)
		return fmt.Errorf("failed to connect to LAN printer at %s: %w", addr, err)
	}

	p.tcpConn = conn
	return nil
}

func (p *Printer) ensureOpenUSBLocked() error {
	if p.device != nil {
		logger.Debugf("USB printer %s already connected", p.idToString())
		return nil // already connected
	}

	ctx := gousb.NewContext()

	var (
		eps     []EndpointInfo
		findAny = p.id == nil
	)

	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if findAny && len(eps) > 0 {
			return false
		}
		ep, ok := findPrinterEndpoint(desc)
		if ok {
			eps = append(eps, ep)
			return true
		}
		return false
	})
	if err != nil {
		_ = ctx.Close()
		return fmt.Errorf("failed to open USB device for printer %s: %w", p.idToString(), err)
	}
	if len(devices) == 0 {
		_ = ctx.Close()
		logger.Warnf("USB printer %s not found", p.idToString())
		return ErrNotFound
	}

	var (
		target   *gousb.Device
		targetEP *EndpointInfo
	)
	for i, d := range devices {
		serial, _ := d.SerialNumber()

		match := false
		if findAny {
			match = true
		} else if p.id.Serial != "" {
			match = serial == p.id.Serial
		} else if p.id.Path != "" {
			match = pathToString(d.Desc) == p.id.Path
		}

		if match && target == nil {
			target = d
			ep := eps[i]
			targetEP = &ep
		} else {
			_ = d.Close()
		}
	}
	if target == nil || targetEP == nil {
		_ = ctx.Close()
		return ErrNotFound
	}

	_ = target.SetAutoDetach(true)

	cfg, err := target.Config(targetEP.config)
	if err != nil {
		// Retry without auto-detach.
		_ = target.SetAutoDetach(false)
		cfg, err = target.Config(targetEP.config)
	}
	logger.Debugf("Configuring USB device %s", p.idToString())
	if err != nil {
		_ = target.Close()
		_ = ctx.Close()
		return err
	}

	iFace, err := cfg.Interface(targetEP.iFace, targetEP.alternateSetting)
	if err != nil {
		logger.Errorf("Failed to claim USB interface for printer %s: Error: %v", p.idToString(), err)
		_ = cfg.Close()
		_ = target.Close()
		_ = ctx.Close()
		return err
	}

	ep, err := iFace.OutEndpoint(targetEP.outEndpoint)
	if err != nil {
		logger.Errorf("Failed to get USB out endpoint for printer %s: Error: %v", p.idToString(), err)
		iFace.Close()
		_ = cfg.Close()
		_ = target.Close()
		_ = ctx.Close()
		return err
	}

	p.usbCtx = ctx
	p.device = target
	p.config = cfg
	p.iFace = iFace
	p.outEndpoint = ep
	return nil
}

func (p *Printer) close() {
	p.mu.Lock()
	logger.Debugf("Closing printer %s", p.idToString())
	defer p.mu.Unlock()
	p.closeDeviceLocked()
}

func (p *Printer) closeDeviceLocked() {
	if p.connectionType == PrinterTypeLAN {
		if p.tcpConn != nil {
			_ = p.tcpConn.Close()
			p.tcpConn = nil
			logger.Debugf("LAN printer %s connection closed", p.idToString())
		}
		return
	}

	// USB close
	if p.device == nil {
		return
	}
	p.iFace.Close()
	_ = p.config.Close()
	_ = p.device.Close()
	_ = p.usbCtx.Close()
	p.device = nil
	p.config = nil
	p.iFace = nil
	p.outEndpoint = nil
	p.usbCtx = nil
	logger.Debugf("USB printer %s device closed", p.idToString())
}

func (p *Printer) idToString() string {
	if p.connectionType == PrinterTypeLAN {
		return fmt.Sprintf("LAN:%s", p.lanIP)
	}
	if p.id != nil {
		return fmt.Sprintf("USB:%s, %s, %s", p.id.Serial, p.id.Path, p.idName)
	}
	return "USB:unknown"
}
