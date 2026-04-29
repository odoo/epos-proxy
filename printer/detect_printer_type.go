package printer

import (
	"epos-proxy/logger"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/gousb"
)

var vidPidTypeMap = map[string]PrinterType{
	"2aaf:6015": TypeTHERMAL, // Essae thermal
	"04b8:0e32": TypeTHERMAL, // Epson thermal
	"04b8:0202": TypeTHERMAL, // Epson thermal
	"04b8:0203": TypeTHERMAL, // Epson thermal
	"2d84:c7c8": TypeTHERMAL, // Zhuhai Poskey Technology
	"4b43:3830": TypeTHERMAL, // Caysn CN811-UWB
}

var nonAlphaRegex = regexp.MustCompile(`[^A-Z]+`)
var officeCmds = map[string]struct{}{"PCL": {}, "PCLC": {}, "PCLXL": {}, "POSTSCRIPT": {}}
var thermalCmds = map[string]struct{}{"ESCPOS": {}, "TSPL": {}, "ZPL": {}}

func isPrinterDevice(device *gousb.Device) (PrinterType, bool) {
	deviceID, isPrinter, _ := getPrinterDeviceID(device)

	printerType := detectPrinterTypeFromCmds(deviceID)
	if isPrinter || strings.Contains(strings.ToUpper(deviceID["CLS"]), "PRINTER") {
		return printerType, true
	}

	vidPid := fmt.Sprintf("%04X:%04X", uint16(device.Desc.Vendor), uint16(device.Desc.Product))
	if t := detectByVidPid(vidPid); t != TypeANY {
		return t, true
	}

	if printerType != TypeANY {
		return printerType, true
	}

	return TypeANY, false
}

func detectPrinterTypeFromCmds(deviceId DeviceID) PrinterType {
	cmds := extractCmds(deviceId)
	for _, c := range cmds {
		if _, ok := officeCmds[c]; ok {
			return TypeOFFICE
		}
	}

	for _, c := range cmds {
		if _, ok := thermalCmds[c]; ok {
			return TypeTHERMAL
		}
	}

	logger.Debugf("CMD: %v, ID: %v", cmds, deviceId)
	return TypeANY
}

func extractCmds(id DeviceID) []string {
	var result []string
	seen := make(map[string]bool)

	raw := id["CMD"]
	if raw == "" {
		return result
	}

	for _, c := range strings.Split(raw, ",") {
		n := nonAlphaRegex.ReplaceAllString(
			strings.ToUpper(strings.TrimSpace(c)),
			"",
		)

		if n != "" && !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}

	return result
}

func detectByVidPid(vidPid string) PrinterType {
	if t, ok := vidPidTypeMap[strings.ToLower(vidPid)]; ok {
		return t
	}

	logger.Debugf("Set any for VID:PID (%s)", vidPid)
	return TypeANY
}
