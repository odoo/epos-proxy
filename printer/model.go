package printer

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/google/gousb"
)

type PrinterConnectionType int

const (
	PrinterTypeUSB PrinterConnectionType = iota
	PrinterTypeLAN
)

const (
	QueueSize    = 100
	WriteTimeout = 30 * time.Second
)

var ErrNotFound = errors.New("printer not found")
var ErrQueueFull = errors.New("printer queue is full")

type JobResult struct {
	OK  bool
	Err error
}

type JobFunc func(p *Printer) JobResult

type Job struct {
	run   JobFunc
	reply chan JobResult
}

type Printer struct {
	connectionType PrinterConnectionType
	id             *PrinterID
	lanIP          string
	mu             sync.Mutex
	// USB fields
	usbCtx      *gousb.Context
	device      *gousb.Device
	config      *gousb.Config
	iFace       *gousb.Interface
	outEndpoint *gousb.OutEndpoint
	// LAN fields
	tcpConn net.Conn
	jobs    chan Job
}

type PrinterID struct {
	Serial string
	Path   string
}

type Info struct {
	ProductName string
	VendorName  string
	Serial      string
	Id          string
	Path        string
}

type UnavailableInfo struct {
	Name  string
	Error string
}

type EndpointInfo struct {
	config           int
	iFace            int
	alternateSetting int
	outEndpoint      int
}

type Printers struct {
	Available   []Info
	Unavailable []UnavailableInfo
}
