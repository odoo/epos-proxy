package printer

import (
	"encoding/base64"
	"epos-proxy/logger"
	"errors"
	"fmt"
	"strings"
)

func encodePrinterID(libUsbPrinter *LibUsbPrinter) (string, error) {
	var parts []string

	if libUsbPrinter.Serial != "" {
		parts = append(parts, "s:"+libUsbPrinter.Serial)
	} else {
		if libUsbPrinter.VidPid != "" {
			parts = append(parts, "vp:"+libUsbPrinter.VidPid)
		}

		if libUsbPrinter.Path != "" {
			parts = append(parts, "p:"+libUsbPrinter.Path)
		}

		if len(parts) == 0 {
			return "", fmt.Errorf("cannot encode printer ID: no identifier provided")
		}
	}

	base := strings.Join(parts, "|")
	id := base64.RawURLEncoding.EncodeToString([]byte(base))
	logger.Infof("LibUsbPrinter: %v | base: %s | encoded id: %s", libUsbPrinter, base, id)
	return id, nil
}

var ErrInvalidPrinterID = errors.New("invalid printer ID format")

func decodePrinterID(id string) (*PrinterID, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return nil, ErrInvalidPrinterID
	}

	var (
		serial string
		VidPid string
		path   string
	)

	raw := string(decoded)
	for _, part := range strings.Split(raw, "|") {
		switch {
		case strings.HasPrefix(part, "s:"):
			serial = strings.TrimPrefix(part, "s:")

		case strings.HasPrefix(part, "vp:"):
			VidPid = strings.TrimPrefix(part, "vp:")

		case strings.HasPrefix(part, "p:"):
			path = strings.TrimPrefix(part, "p:")
		}
	}

	if serial == "" && path == "" && VidPid == "" {
		return nil, ErrInvalidPrinterID
	}

	logger.Infof("Decoded printer ID: %s {serial: %s, VidPid: %s, path: %s}", id, serial, VidPid, path)
	return &PrinterID{
		Serial: serial,
		VidPid: VidPid,
		Path:   path,
	}, nil
}

func EncodeLANPrinterID(ip string) string {
	return base64.RawURLEncoding.EncodeToString([]byte("l:" + ip))
}

func DecodeLANPrinterID(id string) (string, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", false
	}

	if len(decoded) < 3 || decoded[1] != ':' {
		return "", false
	}

	if decoded[0] != 'l' {
		return "", false
	}

	return string(decoded[2:]), true
}

func EncodeBluetoothPrinterID(mac string) string {
	return base64.RawURLEncoding.EncodeToString([]byte("b:" + mac))
}

func DecodeBluetoothPrinterID(id string) (string, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return "", false
	}
	if len(decoded) < 3 || decoded[1] != ':' || decoded[0] != 'b' {
		return "", false
	}
	return string(decoded[2:]), true
}
