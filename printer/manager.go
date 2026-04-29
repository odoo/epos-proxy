package printer

import (
	"epos-proxy/logger"
	"fmt"
	"sync"
)

type Manager struct {
	mu       sync.Mutex
	printers map[string]*Printer
}

func NewManager() *Manager {
	return &Manager{printers: make(map[string]*Printer)}
}

func (m *Manager) Get(rawPrinter RawPrinter) (*Printer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.printers[rawPrinter.PrinterIp]; ok {
		p.PrintType = rawPrinter.PrintType
		logger.Debugf("Reusing existing printer instance for ID: %s", rawPrinter)
		return p, nil
	}

	logger.Debugf("Creating new printer instance for ID: %v", rawPrinter)
	p := newPrinter(rawPrinter)
	if err := p.ensureOpen(); err != nil {
		return nil, fmt.Errorf("failed to open new printer instance for ID %v: %w", rawPrinter, err)
	}

	m.printers[rawPrinter.PrinterIp] = p
	logger.Debugf("Registered new printer instance for ID: %v", rawPrinter)
	return p, nil
}

func (m *Manager) WriteAsync(data []byte, rawPrinter RawPrinter) (<-chan JobResult, error) {
	p, err := m.Get(rawPrinter)
	if err != nil {
		return nil, fmt.Errorf("failed to get printer for ID %v: %w", rawPrinter.PrinterIp, err)
	}

	reply := make(chan JobResult, 1)
	err = p.Enqueue(func(p *Printer) JobResult {
		logger.Debugf("Executing print job for printer %v", rawPrinter)
		if err := p.Write(data); err != nil {
			return JobResult{Err: fmt.Errorf("print job failed for printer %v: %w", rawPrinter, err)}
		}
		logger.Debugf("Print job completed for printer %v", rawPrinter)
		return JobResult{OK: true}
	}, reply)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue print job for printer %v: %w", rawPrinter, err)
	}

	return reply, nil
}
