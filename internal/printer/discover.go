package printer

import (
	"fmt"

	"epos-proxy/internal/config"
	"epos-proxy/internal/logger"
)

// Device describes a discovered printer (USB or LAN).
type Device struct {
	Ip         string `json:"ip"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsLAN      bool   `json:"isLAN"`
	LANIp      string `json:"lanIp,omitempty"`
	Online     bool   `json:"online"`
}

type UnavailableDevice struct {
	Name     string `json:"name"`
	ErrorMsg string `json:"errorMsg"`
	IsLAN    bool   `json:"isLAN"`
	LANIp    string `json:"lanIp,omitempty"`
}

// DiscoveryResult holds the full result of a printer scan across all transports.
type DiscoveryResult struct {
	Printers            []Device            `json:"printers"`
	UnavailablePrinters []UnavailableDevice `json:"unavailablePrinters"`
	ErrorMsg            string              `json:"errorMsg"`
}

// DiscoverAllPrinters queries both USB and configured LAN printers.
// The returned Device.Ip field is always empty; callers that need an HTTP
// endpoint (e.g. the app layer) should populate it after the call.
func DiscoverAllPrinters(cfg *config.Manager) DiscoveryResult {
	available := make([]Device, 0)
	unavailable := make([]UnavailableDevice, 0)
	var scanErr string

	// 1. USB printers
	usbPrinters, err := ListUSBPrinters()
	if err != nil {
		scanErr = err.Error()
		logger.Errorf("USB printer detection failed: %v", err)
	} else if usbPrinters != nil {
		logger.Debugf("Detected %d available USB printers", len(usbPrinters.Available))
		for _, info := range usbPrinters.Available {
			available = append(available, Device{
				Identifier: info.Id,
				Name:       info.Name,
				Type:       string(info.Type),
				IsLAN:      false,
				Online:     true,
			})
		}

		for _, u := range usbPrinters.Unavailable {
			unavailable = append(unavailable, UnavailableDevice{
				Name:     u.Name,
				ErrorMsg: u.Error,
			})
			logger.Warnf("USB printer unavailable: %s (%s)", u.Name, u.Error)
		}
	}

	// 2. LAN printers
	lanPrinters := ListLANPrinters(cfg)
	for _, lan := range lanPrinters {
		available = append(available, Device{
			Identifier: lan.Id,
			Name:       fmt.Sprintf("Network - %s", lan.IP),
			Type:       string(TypeReceipt),
			IsLAN:      true,
			LANIp:      lan.IP,
			Online:     true,
		})
	}

	return DiscoveryResult{
		Printers:            available,
		UnavailablePrinters: unavailable,
		ErrorMsg:            scanErr,
	}
}
