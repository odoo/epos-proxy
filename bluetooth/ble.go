//go:build !darwin || cgo

package bluetooth

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"epos-proxy/logger"
	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter
var adapterOnce sync.Once
var adapterEnableErr error

const defaultBLEWriteChunk = 180

func enableAdapter() error {
	adapterOnce.Do(func() {
		adapterEnableErr = adapter.Enable()
		if adapterEnableErr != nil {
			logger.Errorf("BT/ble: failed to enable bluetooth adapter: %v", adapterEnableErr)
		} else {
			logger.Debugf("BT/ble: bluetooth adapter enabled successfully")
		}
	})
	return adapterEnableErr
}

type BLETransport struct{}

func (t *BLETransport) Name() string {
	return "BLE"
}

func (t *BLETransport) IsAvailable() bool {
	return enableAdapter() == nil
}

func (t *BLETransport) Dial(ctx context.Context, address string) (net.Conn, error) {
	address = NormalizeAddress(address)
	// macOS CoreBluetooth identifies BLE peripherals by UUID rather than exposing
	// their Bluetooth MAC address. Linux and Windows expose a Bluetooth address for
	// BLE devices, but connections are still established through BLE peripheral
	// discovery rather than directly dialing an address.

	dialAddress := address
	if !UuidRegexp.MatchString(address) && runtime.GOOS == "darwin" {
		resolved, ok := resolveMACToBLEUUID(address)
		if ok {
			logger.Debugf("BT/ble: resolved MAC %s to BLE UUID %s", address, resolved)
			dialAddress = resolved
		} else {
			return nil, fmt.Errorf("BT/ble: cannot dial MAC address %s directly on macOS without UUID resolution", address)
		}
	}

	return dialBLE(ctx, dialAddress)
}

func (t *BLETransport) Scan(ctx context.Context) ([]BluetoothPrinterInfo, error) {
	return scanLiveBLEPrinters(ctx, 3*time.Second)
}

// Scans for BLE devices and returns a list of discovered printers.
func scanLiveBLEPrinters(ctx context.Context, timeout time.Duration) ([]BluetoothPrinterInfo, error) {
	logger.Debugf("BT/ble: starting live BLE scan for %v", timeout)

	var mu sync.Mutex
	var devices []BluetoothPrinterInfo
	seen := make(map[string]bool)

	scanDone := make(chan error, 1)
	go func() {
		err := adapter.Scan(func(a *bluetooth.Adapter, result bluetooth.ScanResult) {
			name := result.LocalName()
			if name == "" {
				return
			}
			addrStr := result.Address.String()

			mu.Lock()
			defer mu.Unlock()
			if seen[addrStr] {
				return
			}

			seen[addrStr] = true
			devices = append(devices, BluetoothPrinterInfo{
				Address: addrStr,
				Name:    name,
				Device:  getDeviceType(name),
			})
		})
		scanDone <- err
	}()

	select {
	case <-ctx.Done():
		_ = adapter.StopScan()
		return nil, ctx.Err()
	case <-time.After(timeout):
		_ = adapter.StopScan()
	}

	select {
	case err := <-scanDone:
		if err != nil {
			logger.Errorf("BT/ble: scan failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		logger.Warnf("BT/ble: scan goroutine did not exit within backstop")
	}

	mu.Lock()
	defer mu.Unlock()
	return devices, nil
}

type bleConn struct {
	device       bluetooth.Device
	char         *bluetooth.DeviceCharacteristic
	address      string
	writeTimeout time.Duration
	readDeadline time.Time
	connected    bool
}

func (c *bleConn) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func (c *bleConn) Write(b []byte) (int, error) {
	if c.char == nil {
		return 0, fmt.Errorf("BT/ble: connection closed")
	}

	logger.Debugf("BT/ble: writing %d bytes to %s", len(b), c.address)

	// Split write into chunks because BLE has MTU limit.
	// Safe MTU chunk size is 180 bytes.
	totalWritten := 0

	for totalWritten < len(b) {
		end := totalWritten + defaultBLEWriteChunk
		if end > len(b) {
			end = len(b)
		}
		chunk := b[totalWritten:end]

		n, err := c.char.Write(chunk)
		if err != nil {
			n, err = c.char.WriteWithoutResponse(chunk)
			if err != nil {
				return totalWritten, fmt.Errorf("BT/ble: write failed: %w", err)
			}
			time.Sleep(15 * time.Millisecond)
		}
		totalWritten += n
	}

	return totalWritten, nil
}

func (c *bleConn) Close() error {
	if c.connected {
		err := c.device.Disconnect()
		c.connected = false
		c.char = nil
		return err
	}
	return nil
}

func (c *bleConn) LocalAddr() net.Addr {
	return netAddrPlaceholder{net: "ble", addr: "local-ble"}
}

func (c *bleConn) RemoteAddr() net.Addr {
	return netAddrPlaceholder{net: "ble", addr: c.address}
}

func (c *bleConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *bleConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

func (c *bleConn) SetWriteDeadline(t time.Time) error {
	if t.IsZero() {
		c.writeTimeout = 0
	} else {
		c.writeTimeout = time.Until(t)
	}
	return nil
}

func dialBLE(ctx context.Context, address string) (net.Conn, error) {
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := dialBLEInternal(address)
		ch <- result{conn, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.conn, r.err
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("BT/ble: connect to %s timed out after 15s", address)
	}
}

func dialBLEInternal(address string) (conn net.Conn, err error) {
	logger.Debugf("BT/ble: connecting to %s", address)

	var addr bluetooth.Address
	addr.Set(address)

	device, err := adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, fmt.Errorf("BT/ble: failed to connect to %s: %w", address, err)
	}

	success := false
	defer func() {
		if !success {
			_ = device.Disconnect()
		}
	}()

	services, err := device.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("BT/ble: failed to discover services: %w", err)
	}

	var char *bluetooth.DeviceCharacteristic
	for _, service := range services {
		char, err = discoverPrinterCharacteristic(service)
		if err != nil {
			logger.Debugf("BT/ble: skipping service %s: %v", service.UUID(), err)
			continue
		}
		if char != nil {
			break
		}
	}

	if char == nil {
		return nil, fmt.Errorf("BT/ble: no writable characteristic found")
	}

	logger.Debugf("BT/ble: connected to %s using characteristic %s", address, char.UUID())

	success = true
	return &bleConn{
		device:       device,
		char:         char,
		address:      address,
		writeTimeout: 10 * time.Second,
		connected:    true,
	}, nil
}

var printerKeywords = []string{"print", "printer", "pos", "epson", "star", "thermal", "58", "80"}

func getDeviceType(name string) string {
	name = strings.ToLower(name)

	for _, keyword := range printerKeywords {
		if strings.Contains(name, keyword) {
			return "printer"
		}
	}

	return "other"
}
