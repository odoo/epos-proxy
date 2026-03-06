package printer

import "sync"

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
		return p, nil
	}

	p := newPrinter(id)
	if err := p.ensureOpen(); err != nil {
		return nil, err
	}

	m.printers[id] = p
	return p, nil
}

func (m *Manager) WriteAsync(printerId string, data []byte) (<-chan JobResult, error) {
	p, err := m.Get(printerId)
	if err != nil {
		return nil, err
	}

	reply := make(chan JobResult, 1)
	err = p.Enqueue(func(p *Printer) JobResult {
		if err := p.Write(data); err != nil {
			return JobResult{Err: err}
		}
		return JobResult{OK: true}
	}, reply)
	if err != nil {
		return nil, err
	}

	return reply, nil
}
