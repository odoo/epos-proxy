package printer

import (
	"epos-proxy/logger"
	"fmt"
	"strings"

	"github.com/google/gousb"
)

var keyAliases = map[string]string{
	// Command set aliases
	"CMD":         "CMD",
	"COMMAND SET": "CMD",
	"COMMANDSET":  "CMD",
	"COMMAND":     "CMD",
	"COMMANDS":    "CMD",

	// Manufacturer aliases
	"MFG":          "MFG",
	"MANUFACTURER": "MFG",

	// Model aliases
	"MDL":   "MDL",
	"MODEL": "MDL",
}

func getPrinterDeviceID(dev *gousb.Device) (DeviceID, bool, error) {
	buf := make([]byte, 1024)
	isPrinter := false
	for _, cfg := range dev.Desc.Configs {
		for _, iFace := range cfg.Interfaces {
			for _, alt := range iFace.AltSettings {

				if alt.Class != gousb.ClassPrinter && alt.Class != gousb.ClassVendorSpec {
					continue
				}

				n, err := dev.Control(
					0xA1, // IN | CLASS | INTERFACE
					0x00, // GET_DEVICE_ID
					0x00,
					uint16(iFace.Number),
					buf,
				)

				if err != nil || n < 2 {
					logger.Debugf("USB control transfer failed for interface %d: err=%v, n=%d", iFace.Number, err, n)
					continue
				}

				totalLen := int(buf[0])<<8 | int(buf[1])
				if totalLen <= 2 {
					continue
				}

				strLen := totalLen - 2
				if strLen > n-2 {
					strLen = n - 2
				}

				raw := string(buf[2 : 2+strLen])
				deviceID := _parseDeviceID(raw)
				if len(deviceID) == 0 {
					continue
				}

				if alt.Class == gousb.ClassPrinter {
					isPrinter = true
				}

				logger.Debugf("parsed device ID from interface %d: %v", iFace.Number, deviceID)
				return deviceID, isPrinter, nil
			}
		}
	}

	return nil, isPrinter, fmt.Errorf("device id not found")
}

func _parseDeviceID(raw string) DeviceID {
	result := make(DeviceID)

	parts := strings.Split(raw, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key := _normalizeKey(kv[0])
		val := strings.TrimSpace(kv[1])

		// Merge values if same canonical key appears multiple times
		if existing, ok := result[key]; ok {
			result[key] = existing + "," + val
		} else {
			result[key] = val
		}
	}

	return result
}

func _normalizeKey(key string) string {
	key = strings.ToUpper(strings.TrimSpace(key))

	if v, ok := keyAliases[key]; ok {
		return v
	}
	return key
}
