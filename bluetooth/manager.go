package bluetooth

import (
	"context"
	"epos-proxy/config"
	"epos-proxy/logger"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
)

type BluetoothPrinterInfo struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Device  string `json:"device"`
}

type DependencyStatus struct {
	Name        string `json:"name"`
	InstallCmd  string `json:"installCmd"`
	Description string `json:"description"`
}

type BluetoothManager struct {
	Cfg   *config.Manager
	cache *rfcommCache
}

var BTManager *BluetoothManager

func InitBluetoothManager(cfg *config.Manager) {
	BTManager = &BluetoothManager{
		Cfg: cfg,
		cache: &rfcommCache{
			entries: make(map[string]*rfcommBinding),
		},
	}
}

func (bm *BluetoothManager) GetCachedRFCOMMChannel(address string) int {
	address = NormalizeAddress(address)
	if b, ok := bm.cache.get(address); ok {
		return b.Channel
	}
	return 0
}

func (bm *BluetoothManager) CheckBluetoothPrinter(address string) error {
	if !IsBluetoothAdapterActive() {
		return fmt.Errorf("BT/manager: Bluetooth adapter is not available")
	}

	conn, err := bm.Dial(address)
	if err != nil {
		return fmt.Errorf("bluetooth printer %s is unreachable: %w", address, err)
	}
	_ = conn.Close()
	return nil
}

func (bm *BluetoothManager) GetCachedBinding(address string) (string, int, bool) {
	address = NormalizeAddress(address)
	if b, ok := bm.cache.get(address); ok {
		return b.DevPath, b.Channel, true
	}
	return "", 0, false
}

func supportedTransportsByOS() []Transport {
	if runtime.GOOS == "darwin" {
		return []Transport{
			&BLETransport{},
		}
	}

	return []Transport{
		&ClassicTransport{},
	}
}

// Dial attempts to connect to the Bluetooth device at address using the platform's
// preferred transports in order, automatically falling back if a connection fails.
func (bm *BluetoothManager) Dial(address string) (net.Conn, error) {
	var lastErr error
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for _, t := range supportedTransportsByOS() {
		if !t.IsAvailable() {
			logger.Debugf("BT/manager: transport %s is not available, skipping", t.Name())
			continue
		}

		logger.Debugf("BT/manager: attempting connection via %s to %s", t.Name(), address)
		conn, err := t.Dial(ctx, address)
		if err == nil {
			logger.Debugf("BT/manager: connected via %s to %s", t.Name(), address)
			return conn, nil
		}
		logger.Warnf("BT/manager: connection via %s to %s failed: %v", t.Name(), address, err)
		lastErr = err
	}

	if lastErr == nil {
		return nil, fmt.Errorf("bluetooth/manager: no available Bluetooth transports")
	}
	return nil, fmt.Errorf("bluetooth/manager: all connection strategies failed: %w", lastErr)
}

// ScanBluetoothPrinters queries all available transports for devices and merges the results.
func ScanBluetoothPrinters() ([]BluetoothPrinterInfo, error) {
	logger.Debug("BT/manager: starting Bluetooth printer scan across all available transports")

	var allDevices []BluetoothPrinterInfo
	seen := make(map[string]bool)
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, t := range supportedTransportsByOS() {
		if !t.IsAvailable() {
			continue
		}
		wg.Add(1)
		go func(trans Transport) {
			defer wg.Done()
			devices, err := trans.Scan(ctx)
			if err != nil {
				logger.Warnf("BT/manager: transport %s scan failed: %v", trans.Name(), err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, d := range devices {
				norm := NormalizeAddress(d.Address)
				if !seen[norm] {
					seen[norm] = true
					d.Address = norm
					allDevices = append(allDevices, d)
				}
			}
		}(t)
	}

	wg.Wait()
	logger.Debugf("BT/manager: scan complete, found %d unique printer(s)", len(allDevices))
	return allDevices, nil
}

func IsBluetoothAdapterActive() bool {
	for _, t := range supportedTransportsByOS() {
		if t.IsAvailable() {
			return true
		}
	}
	return false
}
