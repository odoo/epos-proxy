package printer

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"epos-proxy/config"
	"epos-proxy/logger"
)

const (
	LANPort           = 9100
	LANConnectTimeout = 3 * time.Second
)

type LANPrinterInfo struct {
	IP string
	Id string
}

func CheckLANPrinter(ip string) error {
	addr := fmt.Sprintf("%s:%d", ip, LANPort)
	conn, err := net.DialTimeout("tcp", addr, LANConnectTimeout)
	if err != nil {
		logger.Log.Debugf("LAN printer %s is offline or unreachable: %v", ip, err)
		return err
	}
	_ = conn.Close()
	logger.Log.Debugf("Successfully connected to LAN printer %s", ip)
	return nil
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

func ListLANPrinters(cfg *config.Manager) []LANPrinterInfo {
	ips := cfg.GetLANPrinters()
	logger.Log.Debugf("Listing %d configured LAN printers", len(ips))
	result := make([]LANPrinterInfo, len(ips))

	for i, ip := range ips {
		result[i] = LANPrinterInfo{
			IP: ip,
			Id: EncodeLANPrinterID(ip),
		}
	}

	return result
}

func ValidateIPAddress(ip string) (string, error) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		logger.Log.Warn("Attempted to validate empty IP address")
		return "", fmt.Errorf("IP address cannot be empty")
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		logger.Log.Warnf("Invalid IP address format for input: %s", ip)
		return "", fmt.Errorf("invalid IP address format")
	}

	return ip, nil
}
