package bluetooth

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var UuidRegexp = regexp.MustCompile(`^(?i)[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}$`)
var MacRegexp = regexp.MustCompile(`^(?i)([0-9A-F]{2}[:\-]){5}[0-9A-F]{2}$`)

// parseMACToBytes converts "AA:BB:CC:DD:EE:FF" → [6]byte in reversed order
// (little-endian as required by the BlueZ sockaddr_rc).
func ParseMACToBytes(macAddress string) ([6]byte, error) {
	parts := strings.Split(strings.ToUpper(macAddress), ":")
	if len(parts) != 6 {
		return [6]byte{}, fmt.Errorf("invalid MAC: %s", macAddress)
	}
	var b [6]byte
	for i, p := range parts {
		v, err := strconv.ParseUint(p, 16, 8)
		if err != nil {
			return [6]byte{}, fmt.Errorf("invalid MAC octet %q: %w", p, err)
		}
		b[5-i] = byte(v)
	}
	return b, nil
}

func NormalizeAddress(address string) string {
	address = strings.ToUpper(strings.TrimSpace(address))
	// If it is a UUID, preserve it as-is
	if UuidRegexp.MatchString(address) {
		return address
	}
	address = strings.ReplaceAll(address, "-", ":")
	return address
}

func ValidateAddress(address string) error {
	address = strings.TrimSpace(address)
	if MacRegexp.MatchString(address) {
		return nil
	}
	if UuidRegexp.MatchString(address) {
		return nil
	}
	return fmt.Errorf("invalid MAC address format: %s (expected format: AA:BB:CC:DD:EE:FF or UUID)", address)
}
