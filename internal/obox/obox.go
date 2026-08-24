package obox

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"epos-proxy/internal/config"
	"epos-proxy/internal/logger"
)

type StatusListener func()

type Module struct {
	appID string
	port  int
	cfg   *config.Manager

	credMu sync.RWMutex
	dbURL  string
	token  string
	dbUUID string

	wsStatus  atomic.Pointer[string]
	lanStatus atomic.Pointer[string]

	lastContactTime atomic.Int64

	lanMu    sync.Mutex
	lanTimer *time.Timer

	listenersMu sync.RWMutex
	listeners   []StatusListener

	workerMu     sync.Mutex
	workerCancel context.CancelFunc

	ctx    context.Context
	cancel context.CancelFunc
}

func NewModule(cfg *config.Manager, port int) *Module {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Module{
		cfg:    cfg,
		port:   port,
		appID:  cfg.GetAppID(),
		ctx:    ctx,
		cancel: cancel,
	}

	m.setWsStatus("disconnected")
	m.setLANStatus("disconnected")

	if !cfg.HasOdooCredentials() {
		return m
	}

	odooCfg := cfg.GetOdooConfig()
	m.setCredentials(odooCfg.DbURL, odooCfg.Token, odooCfg.DbUUID)

	m.setWsStatus("connecting")
	m.setLANStatus("connecting")
	m.startQueueHandler()

	return m
}

func (m *Module) startQueueHandler() {
	m.workerMu.Lock()
	defer m.workerMu.Unlock()

	if m.workerCancel != nil || m.ctx.Err() != nil {
		return
	}

	dbURL, token := m.GetCredentials()
	if dbURL == "" || token == "" {
		return
	}

	ctx, cancel := context.WithCancel(m.ctx)
	m.workerCancel = cancel
	go m.oboxQueueHandler(ctx)
}

func (m *Module) stopQueueHandler() {
	m.workerMu.Lock()
	defer m.workerMu.Unlock()

	if m.workerCancel == nil {
		return
	}

	m.workerCancel()
	m.workerCancel = nil
}

func (m *Module) SetCredentials(dbURL, token, dbUUID string) {
	m.setCredentials(dbURL, token, dbUUID)
	if err := m.cfg.SetOdooCredentials(dbURL, token, dbUUID); err != nil {
		logger.Warnf("[obox] Failed to save Odoo credentials to storage: %v", err)
	}

	m.startQueueHandler()
}

func (m *Module) setCredentials(dbURL, token, dbUUID string) {
	m.credMu.Lock()
	m.dbURL = dbURL
	m.token = token
	m.dbUUID = dbUUID
	m.credMu.Unlock()
}

func (m *Module) GetCredentials() (string, string) {
	m.credMu.RLock()
	defer m.credMu.RUnlock()

	return m.dbURL, m.token
}

func (m *Module) ClearCredentials() {
	m.setCredentials("", "", "")

	if err := m.cfg.ClearOdooConfig(); err != nil {
		logger.Warnf("[obox] Failed to clear Odoo credentials from storage: %v", err)
	}
}

func (m *Module) GetDbURL() string {
	m.credMu.RLock()
	defer m.credMu.RUnlock()

	return m.dbURL
}

func (m *Module) GetWebsocketStatus() string {
	if status := m.wsStatus.Load(); status != nil {
		return *status
	}
	return "disconnected"
}

func (m *Module) GetLANStatus() string {
	if status := m.lanStatus.Load(); status != nil {
		return *status
	}
	return "disconnected"
}

func (m *Module) cancelLANTimer() {
	m.lanMu.Lock()
	defer m.lanMu.Unlock()

	if m.lanTimer != nil {
		m.lanTimer.Stop()
		m.lanTimer = nil
	}

	m.setLANStatusValue("disconnected")
}

func (m *Module) Disconnect() {
	logger.Infof("[obox] Disconnect triggered")

	m.stopQueueHandler()
	m.ClearCredentials()
	m.setWsStatus("disconnected")
	m.cancelLANTimer()
}

func (m *Module) Stop() {
	m.stopQueueHandler()
	if m.cancel != nil {
		m.cancel()
	}

	m.cancelLANTimer()
}

func (m *Module) OnStatusChange(listener StatusListener) {
	m.listenersMu.Lock()
	defer m.listenersMu.Unlock()

	m.listeners = append(m.listeners, listener)
}

func (m *Module) notifyStatusChange() {
	m.listenersMu.RLock()
	listeners := append([]StatusListener(nil), m.listeners...)
	m.listenersMu.RUnlock()

	for _, listener := range listeners {
		listener()
	}
}

func (m *Module) setWsStatus(status string) {
	prev := m.wsStatus.Load()
	m.wsStatus.Store(&status)

	if prev == nil || *prev != status {
		m.notifyStatusChange()
	}
}

func (m *Module) setLANStatus(status string) {
	m.lanMu.Lock()
	defer m.lanMu.Unlock()

	m.setLANStatusValue(status)
	if m.lanTimer != nil {
		m.lanTimer.Stop()
		m.lanTimer = nil
	}

	var timeout time.Duration
	var next string

	switch status {
	case "connecting":
		timeout, next = 30*time.Second, "disconnected"
	case "connected":
		timeout, next = 30*time.Second, "connecting"
	default:
		return
	}

	m.lanTimer = time.AfterFunc(timeout, func() {
		m.setLANStatus(next)
	})
}

func (m *Module) setLANStatusValue(status string) {
	prev := m.lanStatus.Load()
	m.lanStatus.Store(&status)

	if prev == nil || *prev != status {
		m.notifyStatusChange()
	}
}
