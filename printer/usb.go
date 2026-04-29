package printer

import (
	"fmt"
	"runtime"
	"strings"

	"epos-proxy/logger"
	"epos-proxy/util"

	"github.com/google/gousb"
)

var ignoredSystemPrefixes = []string{
	"PDF_NETWORK_",
}

func ListUSBPrinters() (*Printers, error) {
	logger.Debug("Starting USB printer detection")

	systemUsbPrinters, err := listSystemPrinters()
	if err != nil {
		return nil, fmt.Errorf("failed to get system printers: %w", err)
	}

	libusbPrinters, err := listLibUsbPrinters()
	if err != nil {
		return nil, fmt.Errorf("failed to get USB printers: %w", err)
	}

	result, err := mergePrinters(systemUsbPrinters, libusbPrinters)
	if err != nil {
		return nil, fmt.Errorf("failed to merge printer list: %w", err)
	}
	return result, nil
}

func listLibUsbPrinters() ([]LibUsbPrinter, error) {
	ctx := gousb.NewContext()
	defer ctx.Close()

	current := make(map[string]struct{})

	// First list all  without opening devices, to avoid permission errors on some platforms
	_, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if _, supported := findPrinterEndpoint(desc); supported {
			key := fingerprintKey(desc)
			current[key] = struct{}{}
		}
		return false
	})

	if err != nil {
		return nil, fmt.Errorf("failed to enumerate USB devices: %w", err)
	}

	if !usbCache.HasChanged(current) {
		logger.Debugf("USB unchanged → using cache")
		return usbCache.Get(), nil
	}

	logger.Infof("USB changed → rescanning devices")

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		_, supported := findPrinterEndpoint(desc)
		return supported
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open USB devices: %w", err)
	}

	var printers []LibUsbPrinter

	for _, device := range devs {
		desc := device.Desc
		printerType, ok := isPrinterDevice(device)
		if !ok {
			device.Close()
			continue
		}

		info := LibUsbPrinter{}

		productName, _ := device.Product()
		vendorName, _ := device.Manufacturer()
		serial, _ := device.SerialNumber()

		productName = util.Ternary(productName == "", fmt.Sprintf("PID: %04X", uint16(desc.Product)), productName)
		vendorName = util.Ternary(vendorName == "", fmt.Sprintf("VID: %04X", uint16(desc.Vendor)), vendorName)

		info.Name = fmt.Sprintf("%s %s", vendorName, productName)
		info.Serial = serial
		info.Path = pathToString(desc)
		info.Type = printerType
		info.VidPid = fmt.Sprintf("%04X:%04X", uint16(desc.Vendor), uint16(desc.Product))

		logger.Debugf("USB printer: %s (Serial: %s)", info.Name, info.Serial)

		printers = append(printers, info)

		device.Close()
	}

	usbCache.Update(current, printers)
	return printers, nil
}

func fingerprintKey(desc *gousb.DeviceDesc) string {
	return fmt.Sprintf("%d-%d-%04X:%04X",
		desc.Bus,
		desc.Address,
		desc.Vendor,
		desc.Product,
	)
}

func findPrinterEndpoint(dev *gousb.DeviceDesc) (EndpointInfo, bool) {
	for cfgNum, cfg := range dev.Configs {
		for _, iFace := range cfg.Interfaces {
			for _, alt := range iFace.AltSettings {
				if ep, ok := matchBulkOutEndpoint(alt); ok {
					return EndpointInfo{
						config:           cfgNum,
						iFace:            iFace.Number,
						alternateSetting: alt.Alternate,
						outEndpoint:      ep,
					}, true
				}
			}
		}
	}
	return EndpointInfo{}, false
}

func matchBulkOutEndpoint(alt gousb.InterfaceSetting) (int, bool) {
	if alt.Class != gousb.ClassPrinter && alt.Class != gousb.ClassVendorSpec {
		return 0, false
	}
	for _, ep := range alt.Endpoints {
		if ep.Direction == gousb.EndpointDirectionOut &&
			ep.TransferType == gousb.TransferTypeBulk {
			return ep.Number, true
		}
	}
	return 0, false
}

func mergePrinters(systemPrinters []SystemUsbPrinter, libusbPrinters []LibUsbPrinter) (*Printers, error) {
	result := &Printers{
		Available: make([]Info, 0),
	}

	matchedUSB := make([]bool, len(libusbPrinters))
	matchedSystemPrinterName := make([]string, 0, len(systemPrinters))
	for _, sysUsb := range systemPrinters {
		found := false

		if runtime.GOOS != "windows" {
			for i, libUsb := range libusbPrinters {
				logger.Debugf("Matching USB[%d]: Serial=%v Path=%v with CUPS Serial=%v Name=%s",
					i, libUsb.Serial, libUsb.Path, sysUsb.Serial, sysUsb.IdName)

				// Serial match
				if libUsb.Serial != "" && sysUsb.Serial != "" && libUsb.Serial == sysUsb.Serial {
					logger.Debugf("Matched by SERIAL: %s ↔ %s", libUsb.Serial, sysUsb.Serial)
					if id, err := encodePrinterID(libUsb.Serial, libUsb.Path, sysUsb.IdName); err == nil {
						result.Available = append(result.Available, Info{
							Id:      id,
							Name:    sysUsb.IdName,
							Variant: string(TypeANY),
							Type:    libUsb.Type,
							Label:   sysUsb.Label,
						})
					} else {
						logger.Errorf("failed to encode printer ID: %v", err)
					}
					matchedUSB[i] = true
					found = true
					break
				}
			}
		}

		// No USB match → standalone System printer
		if !found {
			logger.Debugf("No USB match for CUPS printer: %s", sysUsb.IdName)

			// store libusb cache and remove those only
			if sysUsb.Serial != "" {
				continue
			}

			id, err := encodePrinterID("", "", sysUsb.IdName)
			if err != nil {
				logger.Errorf("failed to encode printer ID: %v", err)
				continue
			}

			if !hasIgnoredPrefix(sysUsb.IdName) {
				matchedSystemPrinterName = append(matchedSystemPrinterName, sysUsb.IdName)
			}

			result.Available = append(result.Available, Info{
				Id:      id,
				Name:    sysUsb.IdName,
				Type:    sysUsb.Type,
				Variant: string(TypeOFFICE),
				IsLAN:   sysUsb.IsLAN,
				IP:      sysUsb.IP,
				Label:   sysUsb.Label,
			})
		}
	}
	appendLibusbEposPrinterOnly(libusbPrinters, matchedUSB, result, matchedSystemPrinterName)
	return result, nil
}

func appendLibusbEposPrinterOnly(libusbPrinters []LibUsbPrinter, matchedUSB []bool, result *Printers, matchedSystemPrinterName []string) {
	for i, libUsb := range libusbPrinters {
		if matchedUSB[i] {
			continue
		}

		// libUsb.Type detected by COMMANDS supported by printer
		if libUsb.Type == TypeOFFICE {
			continue
		}

		matched := false
		// skip those which are normal standard printer
		for _, name := range matchedSystemPrinterName {
			if name == "" {
				continue
			}
			if util.IsMatch(libUsb.Name, name) {
				matched = true
				logger.Infof("Printer matched by fuzzy name: %s, %s ", libUsb.Name, name)
				break
			}
		}
		if matched {
			continue
		}
		logger.Debugf("USB-only printer detected: %s", libUsb.Name)

		id, err := encodePrinterID(libUsb.Serial, libUsb.Path, "")
		if err != nil {
			logger.Errorf("failed to encode printer ID: %v", err)
			continue
		}

		result.Available = append(result.Available, Info{
			Id:      id,
			Name:    libUsb.Name,
			Variant: string(TypeTHERMAL),
			Type:    libUsb.Type,
			Label:   "USB",
		})
	}
}

func hasIgnoredPrefix(name string) bool {
	for _, prefix := range ignoredSystemPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
