//go:build darwin

package bluetooth

import (
	"context"
	"errors"
	"net"
)

type ClassicTransport struct{}

func (t *ClassicTransport) Name() string {
	return "Classic"
}

func (t *ClassicTransport) IsAvailable() bool {
	return false
}

func (t *ClassicTransport) Dial(ctx context.Context, address string) (net.Conn, error) {
	return nil, errors.New("not implemented for darwin")
}

func (t *ClassicTransport) Scan(ctx context.Context) ([]BluetoothPrinterInfo, error) {
	return nil, errors.New("not implemented for darwin")
}

func CheckDependencies() []DependencyStatus {
	return []DependencyStatus{}
}
