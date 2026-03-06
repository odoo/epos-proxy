package printer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/gousb"
)

type PrinterType int

const (
	PrinterTypeUSB PrinterType = iota
	PrinterTypeLAN
)

const (
	QueueSize    = 100
	WriteTimeout = 5 * time.Second
)

var ErrNotFound = errors.New("printer not found")
var ErrQueueFull = errors.New("printer queue is full")

type JobResult struct {
	OK  bool
	Err error
}

type JobFunc func(p *Printer) JobResult

type job struct {
	run   JobFunc
	reply chan JobResult
}

type Printer struct {
	printerType PrinterType
	id          *PrinterID
	lanIP       string
	mu          sync.Mutex
	// USB fields
	usbCtx      *gousb.Context
	device      *gousb.Device
	config      *gousb.Config
	iFace       *gousb.Interface
	outEndpoint *gousb.OutEndpoint
	// LAN fields
	tcpConn net.Conn
	jobs    chan job
}

func newPrinter(id string) *Printer {
	// Check if this is a LAN printer
	if lanIP, ok := DecodeLANPrinterID(id); ok {
		p := &Printer{
			printerType: PrinterTypeLAN,
			lanIP:       lanIP,
			jobs:        make(chan job, QueueSize),
		}
		go p.loop()
		return p
	}

	// USB printer
	var printerID *PrinterID = nil
	if id != "" {
		printerID, _ = decodePrinterID(id)
	}

	p := &Printer{
		printerType: PrinterTypeUSB,
		id:          printerID,
		jobs:        make(chan job, QueueSize),
	}

	go p.loop()
	return p
}

func (p *Printer) Enqueue(fn JobFunc, reply chan JobResult) error {
	j := job{run: fn, reply: reply}
	select {
	case p.jobs <- j:
		return nil
	default:
		return ErrQueueFull
	}
}

func (p *Printer) Write(data []byte) error {
	if err := p.ensureOpen(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.printerType == PrinterTypeLAN {
		if err := p.tcpConn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
			p.closeDeviceLocked()
			return err
		}
		if _, err := p.tcpConn.Write(data); err != nil {
			p.closeDeviceLocked()
			return err
		}
		return nil
	}

	// USB write
	ctx, cancel := context.WithTimeout(context.Background(), WriteTimeout)
	defer cancel()

	if _, err := p.outEndpoint.WriteContext(ctx, data); err != nil {
		p.closeDeviceLocked()
		return err
	}
	return nil
}

func (p *Printer) loop() {
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
func (p *Printer) ensureOpen() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.printerType == PrinterTypeLAN {
		return p.ensureOpenLANLocked()
	}
	return p.ensureOpenUSBLocked()
}

func (p *Printer) ensureOpenLANLocked() error {
	if p.tcpConn != nil {
		return nil // already connected
	}

	addr := fmt.Sprintf("%s:%d", p.lanIP, LANPort)
	conn, err := net.DialTimeout("tcp", addr, LANConnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to LAN printer at %s: %w", addr, err)
	}

	p.tcpConn = conn
	return nil
}

func (p *Printer) ensureOpenUSBLocked() error {
	if p.device != nil {
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
		return err
	}
	if len(devices) == 0 {
		_ = ctx.Close()
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
		} else if p.id.ProductID != 0 {
			match = d.Desc.Vendor == p.id.VendorID && d.Desc.Product == p.id.ProductID
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
	if err != nil {
		_ = target.Close()
		_ = ctx.Close()
		return err
	}

	iFace, err := cfg.Interface(targetEP.iFace, targetEP.alternateSetting)
	if err != nil {
		_ = cfg.Close()
		_ = target.Close()
		_ = ctx.Close()
		return err
	}

	ep, err := iFace.OutEndpoint(targetEP.outEndpoint)
	if err != nil {
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
	defer p.mu.Unlock()
	p.closeDeviceLocked()
}

func (p *Printer) closeDeviceLocked() {
	if p.printerType == PrinterTypeLAN {
		if p.tcpConn != nil {
			_ = p.tcpConn.Close()
			p.tcpConn = nil
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
}
