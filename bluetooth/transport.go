package bluetooth

import (
	"context"
	"net"
)

// Transport defines the interface that all Bluetooth connection types (Classic/RFCOMM, BLE) must implement.
type Transport interface {
	// Name returns the display name of the transport.
	Name() string

	// Dial opens a connection to the given address.
	Dial(ctx context.Context, address string) (net.Conn, error)

	// Scan performs a scan for devices matching this transport.
	Scan(ctx context.Context) ([]BluetoothPrinterInfo, error)

	// IsAvailable returns true if the adapter for this transport is active and available.
	IsAvailable() bool
}
