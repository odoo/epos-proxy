package printer

import (
	"sort"
	"strings"
	"sync"
)

type printerCache struct {
	mu                sync.Mutex
	lastSnapshot      string
	cachedPrinters    []Info
	cachedUnavailable []UnavailableInfo
}

var usbCache = &printerCache{}

func (c *printerCache) HasChanged(keyMap map[string]struct{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	snap := buildSnapshot(keyMap)
	return snap != c.lastSnapshot
}

func (c *printerCache) Get() ([]Info, []UnavailableInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cachedPrinters, c.cachedUnavailable
}

func (c *printerCache) Update(keyMap map[string]struct{}, printers []Info, unavailablePrinters []UnavailableInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastSnapshot = buildSnapshot(keyMap)
	c.cachedPrinters = printers
	c.cachedUnavailable = unavailablePrinters
}

func buildSnapshot(keyMap map[string]struct{}) string {
	keys := make([]string, 0, len(keyMap))
	for k := range keyMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}
