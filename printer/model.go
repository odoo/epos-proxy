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
	PrinterTypeBluetooth
)

const (
	QueueSize    = 100
	WriteTimeout = 5 * time.Second
	ChunkSize    = 8 * 1024 // 8 KB
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
	// Bluetooth fields
	bluetoothAddress string
	btDevPath        string // cached /dev/rfcommX path (Linux only; empty on other platforms)
	btConn           net.Conn
}

type PrinterType string

const (
	PrinterTypeReceipt PrinterType = "receipt"
	PrinterTypeLabel   PrinterType = "label"
)

type Info struct {
	Id   string
	Name string
	Type PrinterType
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

type PrinterID struct {
	Serial string
	VidPid string
	Path   string
}

type Printers struct {
	Available   []Info
	Unavailable []UnavailableInfo
}

type DeviceID map[string]string

type LibUsbPrinter struct {
	Serial   string
	Path     string
	Name     string
	VidPid   string
	DeviceId DeviceID
}
