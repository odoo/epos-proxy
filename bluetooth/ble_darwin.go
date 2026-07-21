//go:build darwin && cgo

package bluetooth

import (
	"context"
	"encoding/json"
	"epos-proxy/logger"
	"fmt"
	"os/exec"
	"strings"
	"time"

	cbgo "github.com/tinygo-org/cbgo"
	tinygoBT "tinygo.org/x/bluetooth"
)

// On Darwin+CGO,it uses the Properties() method
// exposed by our local fork of tinygo.org/x/bluetooth to pick the first
// characteristic that supports Write or WriteWithoutResponse.
func discoverPrinterCharacteristic(service tinygoBT.DeviceService) (*tinygoBT.DeviceCharacteristic, error) {
	chars, err := service.DiscoverCharacteristics(nil)
	if err != nil {
		return nil, err
	}
	if len(chars) == 0 {
		return nil, fmt.Errorf("BT/ble: service %s exposes no characteristics", service.UUID())
	}

	writeProps := cbgo.CharacteristicPropertyWrite | cbgo.CharacteristicPropertyWriteWithoutResponse

	var fallback *tinygoBT.DeviceCharacteristic
	for i := range chars {
		c := &chars[i]
		props := c.Properties()
		logger.Debugf("BT/ble: characteristic %s props=0x%x", c.UUID(), int(props))
		if props&writeProps != 0 {
			logger.Debugf("BT/ble: selected writable characteristic %s", c.UUID())
			return c, nil
		}
		if fallback == nil {
			fallback = c
		}
	}

	logger.Debugf("BT/ble: no writable characteristic found, using fallback %s", fallback.UUID())
	return fallback, nil
}

// Resolves a Classic MAC address to a BLE UUID on macOS by
// querying system_profiler for the device's Bluetooth name, then scanning for a
// BLE device with a matching or similar name.
func resolveMACToBLEUUID(mac string) (string, bool) {
	btName := lookupBluetoothName(mac)
	if btName == "" {
		return "", false
	}
	sanitizedTarget := sanitizeForCUName(btName)
	if sanitizedTarget == "" {
		return "", false
	}

	logger.Debugf("BT/darwin/classic: attempting to resolve MAC %s (%q) via BLE scan name-matching", mac, btName)

	ble := &BLETransport{}
	if !ble.IsAvailable() {
		return "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	devices, err := ble.Scan(ctx)
	if err != nil {
		return "", false
	}

	for _, dev := range devices {
		if strings.Contains(strings.ToLower(sanitizeForCUName(dev.Name)), sanitizedTarget) {
			logger.Debugf("BT/darwin/classic: matched BLE device %s (%q) for classic printer %s", dev.Address, dev.Name, mac)
			return dev.Address, true
		}
	}

	return "", false
}

func lookupBluetoothName(mac string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "system_profiler", "SPBluetoothDataType", "-json").Output()
	if ctx.Err() == context.DeadlineExceeded {
		logger.Warnf("BT/darwin: system_profiler timed out resolving BT name for %s", mac)
		return ""
	}
	if err != nil {
		logger.Warnf("BT/darwin: system_profiler failed, cannot resolve BT name for %s: %v", mac, err)
		return ""
	}

	var generic map[string]any
	if err := json.Unmarshal(out, &generic); err != nil {
		logger.Warnf("BT/darwin: failed to parse system_profiler JSON: %v", err)
		return ""
	}

	target := strings.ToLower(mac)
	var found string
	var walk func(v any)
	walk = func(v any) {
		if found != "" {
			return
		}
		switch t := v.(type) {
		case map[string]any:
			if addr, ok := t["device_address"].(string); ok && strings.ToLower(addr) == target {
				if name, ok := t["_name"].(string); ok {
					found = name
					return
				}
			}
			for _, val := range t {
				walk(val)
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(generic)

	if found == "" {
		logger.Warnf("BT/darwin: no system_profiler entry matched MAC %s", mac)
	} else {
		logger.Debugf("BT/darwin: resolved MAC %s -> Bluetooth name %q", mac, found)
	}
	return found
}

func sanitizeForCUName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}
