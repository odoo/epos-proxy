package printer

import (
	"epos-proxy/logger"
	"sync"
)

type Manager struct {
	mu       sync.Mutex
	printers map[string]*Printer
}

func NewManager() *Manager {
	return &Manager{printers: make(map[string]*Printer)}
}

func (m *Manager) Get(id string) (*Printer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.printers[id]; ok {
		logger.Log.Debugf("Reusing existing printer instance for ID: %s", id)
		return p, nil
	}

	logger.Log.Debugf("Creating new printer instance for ID: %s", id)
	p := newPrinter(id)
	if err := p.ensureOpen(); err != nil {
		logger.Log.Errorf("Failed to open new printer instance for ID %s: %v", id, err)
		return nil, err
	}

	m.printers[id] = p
	logger.Log.Infof("Registered new printer instance for ID: %s", id)
	return p, nil
}

func (m *Manager) WriteAsync(printerId string, data []byte) (<-chan JobResult, error) {
	p, err := m.Get(printerId)
	if err != nil {
		return nil, err
	}

	reply := make(chan JobResult, 1)
	err = p.Enqueue(func(p *Printer) JobResult {
		logger.Log.Debugf("Executing print job for printer %s", printerId)
		if err := p.Write(data); err != nil {
			logger.Log.Errorf("Print job failed for printer %s: %v", printerId, err)
			return JobResult{Err: err}
		}
		logger.Log.Debugf("Print job completed for printer %s", printerId)
		return JobResult{OK: true}
	}, reply)
	if err != nil {
		logger.Log.Errorf("Failed to enqueue print job for printer %s: %v", printerId, err)
		return nil, err
	}

	return reply, nil
}
