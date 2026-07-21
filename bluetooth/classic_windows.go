//go:build windows

package bluetooth

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"epos-proxy/logger"

	"golang.org/x/sys/windows"
)

const (
	afBTH          = 32 // AF_BTH
	bthProtoRFCOMM = 3  // BTHPROTO_RFCOMM

	soSndtimeo = 0x1005 // SO_SNDTIMEO
	soRcvtimeo = 0x1006 // SO_RCVTIMEO

	sockaddrBTHSize = 30
)

var (
	ws2_32          = windows.NewLazySystemDLL("ws2_32.dll")
	procSocket      = ws2_32.NewProc("socket")
	procConnect     = ws2_32.NewProc("connect")
	procSend        = ws2_32.NewProc("send")
	procRecv        = ws2_32.NewProc("recv")
	procCloseSocket = ws2_32.NewProc("closesocket")
	procSetsockopt  = ws2_32.NewProc("setsockopt")

	bluetoothAPIs        = windows.NewLazySystemDLL("BluetoothAPIs.dll")
	procBTFindFirst      = bluetoothAPIs.NewProc("BluetoothFindFirstDevice")
	procBTFindNext       = bluetoothAPIs.NewProc("BluetoothFindNextDevice")
	procBTFindClose      = bluetoothAPIs.NewProc("BluetoothFindDeviceClose")
	procBTFindFirstRadio = bluetoothAPIs.NewProc("BluetoothFindFirstRadio")
	procBTFindRadioClose = bluetoothAPIs.NewProc("BluetoothFindRadioClose")
)

type btDeviceSearchParams struct {
	dwSize               uint32
	fReturnAuthenticated uint32
	fReturnRemembered    uint32
	fReturnUnknown       uint32
	fReturnConnected     uint32
	fIssueInquiry        uint32
	cTimeoutMultiplier   uint8
	_                    [3]byte
	hRadio               uintptr
}

type winSYSTEMTIME struct {
	Year, Month, DayOfWeek, Day uint16
	Hour, Minute, Second, Ms    uint16
}

type btDeviceInfo struct {
	dwSize          uint32
	_               [4]byte
	Address         uint64
	ulClassOfDevice uint32
	fConnected      uint32
	fRemembered     uint32
	fAuthenticated  uint32
	stLastSeen      winSYSTEMTIME
	stLastUsed      winSYSTEMTIME
	szName          [248]uint16
}

type ClassicTransport struct{}

func (t *ClassicTransport) Name() string {
	return "Classic"
}

func (t *ClassicTransport) IsAvailable() bool {
	return isBluetoothAdapterActive()
}

func (t *ClassicTransport) Dial(ctx context.Context, address string) (net.Conn, error) {
	channel := BTManager.GetCachedRFCOMMChannel(address)
	return dialRFCOMMPlatform(address, channel)
}

func (t *ClassicTransport) Scan(ctx context.Context) ([]BluetoothPrinterInfo, error) {
	return scanPairedPrinters()
}

func scanPairedPrinters() ([]BluetoothPrinterInfo, error) {
	logger.Debug("BT: scanning for Bluetooth printers on windows")
	if err := bluetoothAPIs.Load(); err != nil {
		return nil, fmt.Errorf("BluetoothAPIs.dll unavailable: %w", err)
	}

	params := btDeviceSearchParams{}
	params.dwSize = uint32(unsafe.Sizeof(params))
	params.fReturnAuthenticated = 1
	params.fReturnRemembered = 1
	params.fReturnConnected = 1

	var info btDeviceInfo
	info.dwSize = uint32(unsafe.Sizeof(info))

	handle, _, e := procBTFindFirst.Call(
		uintptr(unsafe.Pointer(&params)),
		uintptr(unsafe.Pointer(&info)),
	)
	const invalidHandle = ^uintptr(0)
	if handle == invalidHandle || handle == 0 {
		if e == windows.ERROR_NO_MORE_ITEMS {
			return nil, nil
		}
		return nil, fmt.Errorf("BluetoothFindFirstDevice failed: %w", e)
	}
	defer procBTFindClose.Call(handle)

	var devices []BluetoothPrinterInfo

	for {
		mac := btAddrToMAC(info.Address)
		name := utf16ToString(info.szName[:])
		if name == "" {
			name = mac
		}

		logger.Debugf("BT/Windows: found device %s (%s)", name, mac)
		devices = append(devices, BluetoothPrinterInfo{Address: NormalizeAddress(mac), Name: name, Device: "printer"})

		info = btDeviceInfo{}
		info.dwSize = uint32(unsafe.Sizeof(info))
		r, _, _ := procBTFindNext.Call(handle, uintptr(unsafe.Pointer(&info)))
		if r == 0 {
			break
		}
	}

	return devices, nil
}

func isBluetoothAdapterActive() bool {
	if err := bluetoothAPIs.Load(); err != nil {
		return false
	}

	var params struct {
		dwSize uint32
	}
	params.dwSize = 4
	var hRadio syscall.Handle
	hFind, _, _ := procBTFindFirstRadio.Call(
		uintptr(unsafe.Pointer(&params)),
		uintptr(unsafe.Pointer(&hRadio)),
	)

	const invalidHandle = ^uintptr(0)
	if hFind == 0 || hFind == invalidHandle {
		return false
	}

	_ = syscall.CloseHandle(hRadio)
	_, _, _ = procBTFindRadioClose.Call(hFind)
	return true
}

func CheckDependencies() []DependencyStatus {
	return []DependencyStatus{}
}

func macToWindowsBTHAddr(mac string) (uint64, error) {
	parts := strings.Split(strings.ToUpper(mac), ":")
	if len(parts) != 6 {
		return 0, fmt.Errorf("invalid MAC: %s", mac)
	}
	var addr uint64
	for _, p := range parts {
		v, err := parseHexByte(p)
		if err != nil {
			return 0, fmt.Errorf("invalid MAC octet %q: %w", p, err)
		}
		addr = (addr << 8) | uint64(v)
	}
	return addr, nil
}

func parseHexByte(s string) (byte, error) {
	var v uint64
	for _, c := range s {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= uint64(c - '0')
		case c >= 'A' && c <= 'F':
			v |= uint64(c-'A') + 10
		case c >= 'a' && c <= 'f':
			v |= uint64(c-'a') + 10
		default:
			return 0, fmt.Errorf("invalid hex char %q", c)
		}
	}
	return byte(v), nil
}

func makeSockaddrBTH(btAddr uint64, channel uint32) [sockaddrBTHSize]byte {
	var sa [sockaddrBTHSize]byte
	binary.LittleEndian.PutUint16(sa[0:2], afBTH)
	binary.LittleEndian.PutUint64(sa[2:10], btAddr)
	binary.LittleEndian.PutUint32(sa[26:30], channel)
	return sa
}

func btAddrToMAC(addr uint64) string {
	b := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		b[i] = byte(addr & 0xFF)
		addr >>= 8
	}
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", b[0], b[1], b[2], b[3], b[4], b[5])
}

func utf16ToString(s []uint16) string {
	for i, v := range s {
		if v == 0 {
			return windows.UTF16ToString(s[:i])
		}
	}
	return windows.UTF16ToString(s)
}

type windowsBTConn struct {
	sock syscall.Handle
	mac  string
	ch   int
}

func (c *windowsBTConn) LocalAddr() net.Addr {
	return netAddrPlaceholder{net: "rfcomm", addr: fmt.Sprintf("%s/%d", c.mac, c.ch)}
}
func (c *windowsBTConn) RemoteAddr() net.Addr {
	return netAddrPlaceholder{net: "rfcomm", addr: fmt.Sprintf("%s/%d", c.mac, c.ch)}
}

func (c *windowsBTConn) Read(b []byte) (int, error) {
	r, _, err := procRecv.Call(
		uintptr(c.sock),
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(len(b)),
		0,
	)
	if int32(r) < 0 {
		return 0, fmt.Errorf("recv failed: %w", err)
	}
	return int(r), nil
}

func (c *windowsBTConn) Write(b []byte) (int, error) {
	total := 0
	for len(b) > 0 {
		r, _, err := procSend.Call(
			uintptr(c.sock),
			uintptr(unsafe.Pointer(&b[0])),
			uintptr(len(b)),
			0,
		)
		if int32(r) < 0 {
			return total, fmt.Errorf("send failed: %w", err)
		}
		n := int(r)
		total += n
		b = b[n:]
	}
	return total, nil
}

func (c *windowsBTConn) Close() error {
	r, _, err := procCloseSocket.Call(uintptr(c.sock))
	if r != 0 {
		return fmt.Errorf("closesocket failed: %w", err)
	}
	return nil
}

func (c *windowsBTConn) setTimeoutMS(optname int32, ms int32) error {
	r, _, err := procSetsockopt.Call(
		uintptr(c.sock),
		uintptr(windows.SOL_SOCKET),
		uintptr(optname),
		uintptr(unsafe.Pointer(&ms)),
		uintptr(4),
	)
	if r != 0 {
		return fmt.Errorf("setsockopt failed: %w", err)
	}
	return nil
}

func (c *windowsBTConn) SetDeadline(t time.Time) error {
	_ = c.SetReadDeadline(t)
	return c.SetWriteDeadline(t)
}

func (c *windowsBTConn) SetReadDeadline(t time.Time) error {
	ms := int32(time.Until(t).Milliseconds())
	if ms < 0 {
		ms = 1
	}
	return c.setTimeoutMS(soRcvtimeo, ms)
}

func (c *windowsBTConn) SetWriteDeadline(t time.Time) error {
	ms := int32(time.Until(t).Milliseconds())
	if ms < 0 {
		ms = 1
	}
	return c.setTimeoutMS(soSndtimeo, ms)
}

func dialRFCOMM(mac string, channel int) (net.Conn, error) {
	mac = NormalizeAddress(mac)

	btAddr, err := macToWindowsBTHAddr(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid bluetooth MAC %q: %w", mac, err)
	}

	r, _, e := procSocket.Call(afBTH, windows.SOCK_STREAM, bthProtoRFCOMM)
	const invalidSocket = ^uintptr(0)
	if r == invalidSocket {
		return nil, fmt.Errorf("BT socket() failed: %w", e)
	}
	sock := syscall.Handle(r)

	cleanup := func() { procCloseSocket.Call(uintptr(sock)) }

	sa := makeSockaddrBTH(btAddr, uint32(channel))

	timeoutMS := int32(btConnectTimeout.Milliseconds())
	procSetsockopt.Call(
		uintptr(sock),
		uintptr(windows.SOL_SOCKET),
		uintptr(soSndtimeo),
		uintptr(unsafe.Pointer(&timeoutMS)),
		4,
	)

	rc, _, e := procConnect.Call(
		uintptr(sock),
		uintptr(unsafe.Pointer(&sa[0])),
		sockaddrBTHSize,
	)
	if rc != 0 {
		cleanup()
		return nil, fmt.Errorf("RFCOMM connect to %s channel %d failed: %w", mac, channel, e)
	}

	return &windowsBTConn{sock: sock, mac: mac, ch: channel}, nil
}

func dialRFCOMMPlatform(mac string, cachedChannel int) (net.Conn, error) {
	mac = NormalizeAddress(mac)
	logger.Debugf("BT/Windows: dialling %s (cached channel %d)", mac, cachedChannel)

	if cachedChannel > 0 {
		if conn, err := dialRFCOMM(mac, cachedChannel); err == nil {
			return conn, nil
		}
	}

	for _, ch := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		if ch == cachedChannel {
			continue
		}
		logger.Debugf("BT/Windows: probing channel %d for %s", ch, mac)
		if conn, err := dialRFCOMM(mac, ch); err == nil {
			logger.Debugf("BT/Windows: channel %d succeeded for %s", ch, mac)
			BTManager.setBinding(mac, &rfcommBinding{DevPath: "", Channel: ch, Index: -1})
			return conn, nil
		}
	}

	return nil, fmt.Errorf("BT/Windows: no working RFCOMM channel found for %s", mac)
}
