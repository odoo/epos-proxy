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

type PrinterCategory int

const (
	PrinterThermal PrinterCategory = iota
	PrinterOffice
)

type RawPrinter struct {
	PrinterIp string
	Category  PrinterCategory
}

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
	Category       PrinterCategory
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
	// For office printer (e.g CupsName(Linux/Mac), Name(Windows))
	idName string
}

type PrinterID struct {
	Serial string
	Path   string
	IdName string
}

type PrinterType string

const (
	TypeTHERMAL PrinterType = "THERMAL"
	TypeOFFICE  PrinterType = "OFFICE"
	TypeANY     PrinterType = "ANY"
)

type SystemUsbPrinter struct {
	Serial   string
	IdName   string
	DeviceID string
	Type     PrinterType
	CupsUri  string // linux
	Label    string // win
	IsLAN    bool
	IP       string
}

type LibUsbPrinter struct {
	Serial string
	Path   string
	Name   string
	Type   PrinterType
	VidPid string
}

type Info struct {
	Id      string
	Name    string
	Type    PrinterType // Based on supported command languages
	Variant string      // Determined during record creation
	IsLAN   bool
	IP      string
	Label   string
}

type EndpointInfo struct {
	config           int
	iFace            int
	alternateSetting int
	outEndpoint      int
}

type Printers struct {
	Available []Info
}

type DeviceFingerprint struct {
	Bus     int
	Address int
	VidPid  string
}

type DeviceID map[string]string
