package printer

import (
	"fmt"

	"epos-proxy/logger"
	"epos-proxy/util"

	"github.com/google/gousb"
)

func ListUSBPrinters() (*Printers, error) {
	logger.Debug("Starting USB printer detection")
	ctx := gousb.NewContext()
	defer func(ctx *gousb.Context) {
		_ = ctx.Close()

	}(ctx)

	current := make(map[string]struct{})

	// First list all  without opening devices, to avoid permission errors on some platforms
	var descriptors []gousb.DeviceDesc
	_, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if _, supported := findPrinterEndpoint(desc); supported {
			descriptors = append(descriptors, *desc)
			key := fingerprintKey(desc)
			current[key] = struct{}{}
		}
		return false
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open USB devices for listing: %w", err)
	}

	if !usbCache.HasChanged(current) {
		logger.Debugf("USB unchanged → using cache")
		availablePrinters, unavailablePrinters := usbCache.Get()
		return &Printers{
			Available:   availablePrinters,
			Unavailable: unavailablePrinters,
		}, nil
	}

	logger.Infof("USB changed → rescanning devices")

	result := &Printers{
		Available:   make([]Info, 0),
		Unavailable: make([]UnavailableInfo, 0),
	}
	for _, desc := range descriptors {
		info, err := GetPrinterInfo(ctx, &desc)
		if err != nil {
			// Device is not accessible, likely due to permissions / drivers.
			vid := fmt.Sprintf("%04X", uint16(desc.Vendor))
			pid := fmt.Sprintf("%04X", uint16(desc.Product))
			result.Unavailable = append(result.Unavailable, UnavailableInfo{
				Name:  getPrinterFriendlyName(vid, pid),
				Error: err.Error(),
			})
		} else if info != nil {
			logger.Debugf("Found available USB printer: %s (Serial: %s)", info.ProductName, info.Serial)
			result.Available = append(result.Available, *info)
		}
	}

	usbCache.Update(current, result.Available, result.Unavailable)

	return result, nil
}

func GetPrinterInfo(ctx *gousb.Context, descToFind *gousb.DeviceDesc) (*Info, error) {
	logger.Debugf("Attempting to get info for USB device: Bus %d, Address %d, Vendor %04X, Product %04X", descToFind.Bus, descToFind.Address, uint16(descToFind.Vendor), uint16(descToFind.Product))
	var found bool
	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if found {
			return false
		}
		if descToFind.Bus != desc.Bus || descToFind.Address != desc.Address ||
			descToFind.Vendor != desc.Vendor || descToFind.Product != desc.Product {
			return false
		}
		found = true
		return true
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open USB device for info retrieval: %w", err)
	}

	if len(devices) == 0 {
		return nil, nil
	}

	defer func() {
		for _, d := range devices {
			_ = d.Close()
		}
	}()

	device := devices[0]
	info := &Info{}
	productName, _ := device.Product()
	vendorName, _ := device.Manufacturer()

	info.ProductName = util.Ternary(productName == "", fmt.Sprintf("PID: %04X", uint16(descToFind.Product)), productName)
	info.VendorName = util.Ternary(vendorName == "", fmt.Sprintf("VID: %04X", uint16(descToFind.Vendor)), vendorName)
	info.Serial, _ = device.SerialNumber()
	info.Path = pathToString(descToFind)
	id, err := encodePrinterID(info.Serial, info.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to encode printer ID: %w", err)
	}
	info.Id = id
	return info, nil

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
				if alt.Class != gousb.ClassPrinter {
					continue
				}
				for _, ep := range alt.Endpoints {
					if ep.Direction == gousb.EndpointDirectionOut &&
						ep.TransferType == gousb.TransferTypeBulk {
						return EndpointInfo{
							config:           cfgNum,
							iFace:            iFace.Number,
							alternateSetting: alt.Alternate,
							outEndpoint:      ep.Number,
						}, true
					}
				}
			}
		}
	}
	return EndpointInfo{}, false
}
