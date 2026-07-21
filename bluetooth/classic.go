package bluetooth

import (
	"net"
	"os"
	"sync"
	"time"
)

// btConnectTimeout is the maximum time allowed for a single RFCOMM connect attempt.
const btConnectTimeout = 3 * time.Second

// rfcommBinding records the state of a bound (or candidate) RFCOMM device.
// On Linux the DevPath refers to an actual /dev/rfcommX node;
// on Darwin/Windows it is used only as a value holder.
type rfcommBinding struct {
	DevPath string // e.g. "/dev/rfcomm0"
	Channel int    // RFCOMM channel number
	Index   int    // numeric index (0 = rfcomm0, 1 = rfcomm1, …)
}

type rfcommCache struct {
	mu      sync.RWMutex
	entries map[string]*rfcommBinding
}

func (c *rfcommCache) get(address string) (*rfcommBinding, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, ok := c.entries[address]
	return b, ok
}

func (c *rfcommCache) set(address string, b *rfcommBinding) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[address] = b
}

func (c *rfcommCache) delete(address string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, address)
}

func (bm *BluetoothManager) setBinding(address string, b *rfcommBinding) {
	oldBinding, ok := bm.cache.get(address)
	if ok && oldBinding.Channel == b.Channel && oldBinding.DevPath == b.DevPath {
		return
	}
	bm.cache.set(address, b)
}

type serialConn struct {
	f    *os.File
	path string
}

type serialAddr struct{ path string }

func (a serialAddr) Network() string { return "rfcomm-serial" }
func (a serialAddr) String() string  { return a.path }

func (c *serialConn) Read(b []byte) (int, error)  { return c.f.Read(b) }
func (c *serialConn) Write(b []byte) (int, error) { return c.f.Write(b) }
func (c *serialConn) Close() error                { return c.f.Close() }

func (c *serialConn) LocalAddr() net.Addr {
	return netAddrPlaceholder{net: "rfcomm-serial", addr: c.path}
}
func (c *serialConn) RemoteAddr() net.Addr {
	return netAddrPlaceholder{net: "rfcomm-serial", addr: c.path}
}

func (c *serialConn) SetDeadline(t time.Time) error      { return nil }
func (c *serialConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *serialConn) SetWriteDeadline(t time.Time) error { return nil }

type netAddrPlaceholder struct {
	net  string
	addr string
}

func (a netAddrPlaceholder) Network() string { return a.net }
func (a netAddrPlaceholder) String() string  { return a.addr }
