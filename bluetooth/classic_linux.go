//go:build linux

package bluetooth

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"epos-proxy/logger"

	"golang.org/x/sys/unix"
)

var rfcommBindDisabled = false
var rfcommBindDisabledMu sync.RWMutex

func isRFCOMMBindDisabled() bool {
	rfcommBindDisabledMu.RLock()
	defer rfcommBindDisabledMu.RUnlock()
	return rfcommBindDisabled
}

func disableRFCOMMBind() {
	rfcommBindDisabledMu.Lock()
	rfcommBindDisabled = true
	rfcommBindDisabledMu.Unlock()
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

// scanPairedPrinters lists paired Bluetooth printers via bluetoothctl.
func scanPairedPrinters() ([]BluetoothPrinterInfo, error) {
	logger.Debug("BT: scanning for Bluetooth printers on Linux")

	out, err := exec.Command("bluetoothctl", "devices").Output()
	if err != nil {
		return nil, fmt.Errorf("bluetoothctl devices failed: %w — is bluez installed?", err)
	}

	var devices []BluetoothPrinterInfo
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		// Format: "Device AA:BB:CC:DD:EE:FF Device Name"
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Device ") {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			continue
		}

		mac := NormalizeAddress(parts[1])
		if seen[mac] {
			continue
		}

		name := "Unknown"
		if len(parts) == 3 {
			name = strings.TrimSpace(parts[2])
		}

		infoOut, err := exec.Command("bluetoothctl", "info", mac).Output()
		if err != nil {
			logger.Warnf("BT: failed to get info for %s: %v", mac, err)
			continue
		}

		info := strings.ToLower(string(infoOut))
		hasPrinterIcon := strings.Contains(info, "icon: printer")
		hasSPP := strings.Contains(info, "uuid: serial port")
		if !hasPrinterIcon || !hasSPP {
			continue
		}

		seen[mac] = true
		devices = append(devices, BluetoothPrinterInfo{Address: NormalizeAddress(mac), Name: name, Device: "printer"})
	}

	logger.Debugf("BT: found %d Bluetooth printers", len(devices))
	return devices, nil
}

func isBluetoothAdapterActive() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "bluetoothctl", "show").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Powered: yes")
}

func CheckDependencies() []DependencyStatus {
	deps := []DependencyStatus{}

	// Check bluetoothctl (bluez)
	_, err := exec.LookPath("bluetoothctl")
	if err != nil {
		deps = append(deps, DependencyStatus{
			Name:        "bluez",
			InstallCmd:  "sudo apt-get install bluez",
			Description: "Required to scan for Bluetooth devices and manage connections",
		})
	}

	return deps
}

// Strategy: Try cached channel → SDP → probe channels 1–8 → raw socket fallback.
func dialRFCOMMPlatform(mac string, cachedChannel int) (net.Conn, error) {
	mac = NormalizeAddress(mac)

	// Step 0: Try cached channel first if it exists.
	if cachedChannel > 0 {
		logger.Debugf("BT/RFCOMM: trying cached channel %d for %s", cachedChannel, mac)
		if conn, err := tryRFCOMMDevice(mac, cachedChannel); err == nil {
			logger.Debugf("BT/RFCOMM: connected successfully on cached channel %d for %s", cachedChannel, mac)
			return conn, nil
		}
		logger.Warnf("BT/RFCOMM: connection on cached channel %d failed for %s", cachedChannel, mac)
	}

	// Step 1: Channel probing.
	for _, ch := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		if ch == cachedChannel {
			continue
		}

		logger.Debugf("BT/RFCOMM: probing channel %d for %s", ch, mac)
		if conn, err := tryRFCOMMDevice(mac, ch); err == nil {
			logger.Debugf("BT/RFCOMM: channel probe succeeded on channel %d for %s", ch, mac)
			return conn, nil
		} else {
			logger.Errorf("BT/RFCOMM: channel %d probe failed for %s: %v", ch, mac, err)
			BTManager.cache.delete(mac)
		}
	}

	// Step 2: Raw socket fallback.
	logger.Warnf("BT/RFCOMM: /dev/rfcommX approach exhausted for %s; falling back to raw RFCOMM socket", mac)
	fallbackCh := cachedChannel
	if fallbackCh == 0 {
		fallbackCh = 1
	}

	conn, err := dialRFCOMM(mac, fallbackCh)
	if err != nil {
		return nil, fmt.Errorf("BT/RFCOMM: all connection strategies failed for %s: %w", mac, err)
	}

	logger.Debugf("BT/RFCOMM: raw socket fallback succeeded for %s on channel %d", mac, fallbackCh)
	BTManager.setBinding(mac, &rfcommBinding{DevPath: "raw", Channel: fallbackCh, Index: -1})
	return conn, nil
}

func dialRFCOMM(mac string, channel int) (net.Conn, error) {
	mac = NormalizeAddress(mac)

	addr, err := ParseMACToBytes(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid bluetooth MAC %q: %w", mac, err)
	}

	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_STREAM, unix.BTPROTO_RFCOMM)
	if err != nil {
		return nil, fmt.Errorf("create RFCOMM socket failed: %w", err)
	}
	cleanup := func() { _ = unix.Close(fd) }

	unix.CloseOnExec(fd)
	if err := unix.SetNonblock(fd, true); err != nil {
		cleanup()
		return nil, fmt.Errorf("set nonblocking mode failed: %w", err)
	}

	_ = unix.SetsockoptLinger(fd, unix.SOL_SOCKET, unix.SO_LINGER, &unix.Linger{Onoff: 1, Linger: 1})

	sa := &unix.SockaddrRFCOMM{Addr: addr, Channel: uint8(channel)}
	err = unix.Connect(fd, sa)
	if err != nil && err != unix.EINPROGRESS && err != unix.EAGAIN {
		cleanup()
		return nil, fmt.Errorf("RFCOMM connect to %s channel %d failed: %w", mac, channel, err)
	}

	pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLOUT}}
	n, err := unix.Poll(pollFds, int(btConnectTimeout.Milliseconds()))
	if err != nil || n == 0 {
		cleanup()
		return nil, fmt.Errorf("RFCOMM poll failed or timeout to %s channel %d: %w", mac, channel, err)
	}

	if pollFds[0].Revents&(unix.POLLERR|unix.POLLHUP|unix.POLLNVAL) != 0 {
		soErr, _ := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
		cleanup()
		if soErr != 0 {
			return nil, fmt.Errorf("RFCOMM connect to %s channel %d failed: errno=%d (%s)",
				mac, channel, soErr, syscall.Errno(soErr).Error())
		}
		return nil, fmt.Errorf("RFCOMM connect to %s channel %d failed", mac, channel)
	}

	soErr, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
	if err != nil {
		return nil, fmt.Errorf("SO_ERROR check failed: %w", err)
	}
	if soErr != 0 {
		cleanup()
		return nil, fmt.Errorf("RFCOMM connect to %s channel %d failed: errno=%d (%s)", mac, channel, soErr, syscall.Errno(soErr).Error())
	}

	file := os.NewFile(uintptr(fd), fmt.Sprintf("rfcomm-%s-%d", mac, channel))
	if file == nil {
		cleanup()
		return nil, fmt.Errorf("failed to create os.File from RFCOMM socket")
	}

	return &serialConn{f: file, path: fmt.Sprintf("rfcomm-%s-%d", mac, channel)}, nil
}

func tryRFCOMMDevice(mac string, channel int) (net.Conn, error) {
	if b, ok := BTManager.cache.get(mac); ok {
		if b.Channel != channel {
			BTManager.cache.delete(mac)
			_ = releaseRFCOMM(b.Index)
		} else if b.DevPath == "raw" {
			logger.Debugf("BT/RFCOMM: cache hit for raw socket connection to %s on channel %d", mac, channel)
			conn, err := dialRFCOMM(mac, channel)
			if err != nil {
				logger.Warnf("BT/RFCOMM: cached raw socket connection failed: %v; clearing cache", err)
				BTManager.cache.delete(mac)
				return nil, err
			}
			return conn, nil
		}
	}

	b, err := ensureRFCOMMDevice(mac, channel)
	if err == nil {
		conn, err := openRFCOMMDevice(b)
		if err == nil {
			return conn, nil
		}
		logger.Warnf("BT/RFCOMM: failed to open bound device %s: %v", b.DevPath, err)
		_ = releaseRFCOMM(b.Index)
		BTManager.cache.delete(mac)
	} else {
		logger.Warnf("BT/RFCOMM: binding device failed: %v", err)
	}

	logger.Debugf("BT/RFCOMM: falling back to raw RFCOMM socket for channel %d", channel)
	conn, err := dialRFCOMM(mac, channel)
	if err == nil {
		BTManager.setBinding(mac, &rfcommBinding{DevPath: "raw", Channel: channel, Index: -1})
		return conn, nil
	}
	return nil, err
}

func releaseRFCOMM(index int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "rfcomm", "release", strconv.Itoa(index)).CombinedOutput()
	if err != nil {
		logger.Debugf("BT/RFCOMM: rfcomm release for index %d returned: %v; output: %s", index, err, strings.TrimSpace(string(out)))
		return err
	}
	logger.Debugf("BT/RFCOMM: released rfcomm%d", index)
	return nil
}

func ensureRFCOMMDevice(mac string, channel int) (*rfcommBinding, error) {
	mac = NormalizeAddress(mac)

	if isRFCOMMBindDisabled() {
		return nil, fmt.Errorf("BT/RFCOMM: rfcomm bind is disabled on this system")
	}

	if b, ok := BTManager.cache.get(mac); ok {
		logger.Debugf("BT/RFCOMM: cache hit for %s → %s (channel %d)", mac, b.DevPath, b.Channel)
		if b.DevPath == "raw" {
			return nil, fmt.Errorf("BT/RFCOMM: raw cache hit")
		}
		if _, err := os.Stat(b.DevPath); err == nil {
			return b, nil
		}
		logger.Warnf("BT/RFCOMM: cached device %s no longer exists; re-binding", b.DevPath)
	}

	if existing, ok := findExistingRFCOMMDevice(mac); ok {
		if channel > 0 {
			existing.Channel = channel
		}
		if existing.Channel == 0 {
			existing.Channel = 1
		}
		logger.Debugf("BT/RFCOMM: reusing existing device %s for %s (channel %d)", existing.DevPath, mac, existing.Channel)
		BTManager.setBinding(mac, existing)
		return existing, nil
	}

	if channel <= 0 {
		channel = 1
	}

	b, err := bindRFCOMM(mac, channel)
	if err != nil {
		return nil, err
	}

	BTManager.setBinding(mac, b)
	logger.Debugf("BT/RFCOMM: bound %s → %s (channel %d)", mac, b.DevPath, b.Channel)
	return b, nil
}

func openRFCOMMDevice(b *rfcommBinding) (net.Conn, error) {
	f, err := os.OpenFile(b.DevPath, os.O_RDWR, 0)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("BT/RFCOMM: cannot open %s — add user to 'dialout' or 'bluetooth' group: %w", b.DevPath, err)
		}
		return nil, fmt.Errorf("BT/RFCOMM: failed to open %s: %w", b.DevPath, err)
	}
	logger.Debugf("BT/RFCOMM: opened %s as serial connection", b.DevPath)
	return &serialConn{f: f, path: b.DevPath}, nil
}

func findExistingRFCOMMDevice(mac string) (*rfcommBinding, bool) {
	bindings, err := listRFCOMMBindings()
	if err != nil {
		logger.Warnf("BT/RFCOMM: could not list RFCOMM bindings: %v", err)
		return nil, false
	}

	mac = NormalizeAddress(mac)
	for idx, boundMAC := range bindings {
		if boundMAC == mac {
			devPath := fmt.Sprintf("/dev/rfcomm%d", idx)
			if _, err := os.Stat(devPath); err != nil {
				logger.Warnf("BT/RFCOMM: binding for %s found at index %d but %s does not exist: %v",
					mac, idx, devPath, err)
			}
			logger.Debugf("BT/RFCOMM: existing binding found for %s → %s", mac, devPath)
			return &rfcommBinding{DevPath: devPath, Index: idx, Channel: 0}, true
		}
	}

	logger.Debugf("BT/RFCOMM: no existing RFCOMM binding for %s", mac)
	return nil, false
}

func listRFCOMMBindings() (map[int]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "rfcomm", "-a").CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("rfcomm -a timed out")
	}
	if err != nil {
		return nil, fmt.Errorf("rfcomm -a failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	bindings := make(map[int]string)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		devName := strings.TrimSpace(line[:colonIdx])
		if !strings.HasPrefix(devName, "rfcomm") {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimPrefix(devName, "rfcomm"))
		if err != nil {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(line[colonIdx+1:]))
		if len(fields) < 1 {
			continue
		}
		bindings[idx] = NormalizeAddress(fields[0])
	}

	logger.Debugf("BT/RFCOMM: listRFCOMMBindings found %d bound device(s): %v", len(bindings), bindings)
	return bindings, nil
}

func bindRFCOMM(mac string, channel int) (*rfcommBinding, error) {
	mac = NormalizeAddress(mac)

	existing, err := listRFCOMMBindings()
	if err != nil {
		return nil, fmt.Errorf("BT/RFCOMM: cannot list bindings (is rfcomm installed?): %w", err)
	}

	for idx, boundMAC := range existing {
		if boundMAC == mac {
			devPath := fmt.Sprintf("/dev/rfcomm%d", idx)
			logger.Debugf("BT/RFCOMM: bind skipped — %s already bound as %s", mac, devPath)
			return &rfcommBinding{DevPath: devPath, Index: idx, Channel: channel}, nil
		}
	}

	idx := findFreeRFCOMMIndex(existing)
	if idx < 0 {
		return nil, fmt.Errorf("BT/RFCOMM: no free RFCOMM index available (all 0..31 are occupied)")
	}

	devPath := fmt.Sprintf("/dev/rfcomm%d", idx)
	args := []string{"bind", strconv.Itoa(idx), mac, strconv.Itoa(channel)}
	logger.Debugf("BT/RFCOMM: running: rfcomm %s", strings.Join(args, " "))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "rfcomm", args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("BT/RFCOMM: rfcomm bind timed out")
	}
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if strings.Contains(outStr, "Operation not permitted") ||
			strings.Contains(outStr, "permission denied") ||
			strings.Contains(err.Error(), "permission denied") {
			return nil, fmt.Errorf(
				"BT/RFCOMM: rfcomm bind failed — insufficient privileges "+
					"(run as root or add user to 'bluetooth'/'dialout' group): %w; output: %s",
				err, outStr)
		}
		return nil, fmt.Errorf("BT/RFCOMM: rfcomm bind failed: %w; output: %s", err, outStr)
	}

	logger.Debugf("BT/RFCOMM: rfcomm bind succeeded, waiting for %s to appear…", devPath)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(devPath); err == nil {
			logger.Debugf("BT/RFCOMM: device %s is ready", devPath)
			return &rfcommBinding{DevPath: devPath, Index: idx, Channel: channel}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	disableRFCOMMBind()
	return nil, fmt.Errorf("BT/RFCOMM: %s did not appear after rfcomm bind", devPath)
}

func findFreeRFCOMMIndex(existing map[int]string) int {
	for i := 0; i <= 31; i++ {
		if _, used := existing[i]; !used {
			return i
		}
	}
	return -1
}
