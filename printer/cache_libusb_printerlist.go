package printer

import (
	"sort"
	"strings"
	"sync"
)

type printerCache struct {
	mu             sync.Mutex
	lastSnapshot   string
	cachedPrinters []LibUsbPrinter
}

var usbCache = &printerCache{}

func (c *printerCache) HasChanged(keyMap map[string]struct{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	snap := buildSnapshot(keyMap)
	return snap != c.lastSnapshot
}

func (c *printerCache) Get() []LibUsbPrinter {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cachedPrinters
}

func (c *printerCache) Update(keyMap map[string]struct{}, printers []LibUsbPrinter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastSnapshot = buildSnapshot(keyMap)
	c.cachedPrinters = printers
}

func buildSnapshot(keyMap map[string]struct{}) string {
	keys := make([]string, 0, len(keyMap))
	for k := range keyMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}
