package config

func (cm *Manager) AddBluetoothPrinter(address, name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, existing := range cm.Data.BluetoothPrinters {
		if existing.Address == address {
			cm.Data.BluetoothPrinters[i].Name = name
			return cm.saveLocked()
		}
	}
	cm.Data.BluetoothPrinters = append(cm.Data.BluetoothPrinters, BluetoothPrinterConfig{
		Address: address,
		Name:    name,
	})
	return cm.saveLocked()
}

func (cm *Manager) RemoveBluetoothPrinter(address string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, existing := range cm.Data.BluetoothPrinters {
		if existing.Address == address {
			cm.Data.BluetoothPrinters = append(cm.Data.BluetoothPrinters[:i], cm.Data.BluetoothPrinters[i+1:]...)
			return cm.saveLocked()
		}
	}
	return nil
}

func (cm *Manager) GetBluetoothPrinters() []BluetoothPrinterConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.Data.BluetoothPrinters == nil {
		return []BluetoothPrinterConfig{}
	}
	result := make([]BluetoothPrinterConfig, len(cm.Data.BluetoothPrinters))
	copy(result, cm.Data.BluetoothPrinters)
	return result
}
