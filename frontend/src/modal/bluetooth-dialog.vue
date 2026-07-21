<template>
  <div @click="showBluetoothDialog = true"
    class="flex-1 border-2 border-dashed border-gray-300 bg-gray-50 rounded-lg px-4 py-3 text-gray-600 hover:border-gray-400 hover:bg-gray-100 cursor-pointer flex items-center justify-center gap-2 transition-colors">
    <span class="w-4 h-4 shrink-0" v-html="bluetoothIcon"></span> Add Bluetooth Printer
  </div>
  <teleport to="body">
    <transition enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0"
      enter-to-class="opacity-100" leave-active-class="transition duration-150 ease-in" leave-from-class="opacity-100"
      leave-to-class="opacity-0">
      <div v-if="showBluetoothDialog" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/75" @click="close" />
        <div class="relative bg-white rounded-2xl w-full max-w-sm shadow-xl overflow-hidden p-6">

          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <span class="w-5 h-5 shrink-0" v-html="bluetoothIcon"></span>
              <div class="text-lg font-medium">Add Bluetooth Printer</div>
            </div>
            <CloseButton @click="close" />
          </div>

          <!-- Dependencies Warning -->
          <div v-if="dependencies.length"
            class="mb-4 bg-amber-50 border border-amber-200 rounded-lg p-3 text-xs text-amber-800">
            <div class="font-semibold flex items-center gap-1.5 mb-1.5 text-amber-950">
              <span class="w-4 h-4 shrink-0 inline-block" v-html="warningIcon"></span> Missing Bluetooth Dependencies
            </div>
            <div class="space-y-2">
              <div v-for="dep in dependencies" :key="dep.name"
                class="border-t border-amber-200/50 pt-1.5 first:border-0 first:pt-0">
                <div class="font-semibold text-amber-950">{{ dep.name }}</div>
                <div class="text-amber-700 mb-1.5">{{ dep.description }}</div>
                <div
                  class="bg-amber-950/5 text-amber-900 px-2 py-1 rounded font-mono select-all flex justify-between items-center gap-1">
                  <span class="truncate">{{ dep.installCmd }}</span>
                  <span
                    class="text-[10px] uppercase text-amber-600 font-sans tracking-wider font-semibold cursor-pointer hover:text-amber-800 shrink-0"
                    @click="copyCmd(dep.installCmd)">Copy</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Scan Button -->
          <button @click="scan" :disabled="scanning"
            class="w-full border rounded-lg px-4 py-2 mb-4 text-sm cursor-pointer flex items-center justify-center gap-2 border-stone-300 text-stone-700 hover:bg-stone-50 disabled:opacity-50 disabled:cursor-not-allowed">
            <span v-if="scanning"
              class="animate-spin inline-block w-4 h-4 border-2 border-current border-t-transparent rounded-full"></span>
            <span v-else class="w-4 h-4 shrink-0 inline-block" v-html="searchIcon"></span>
            <span>{{ scanning ? 'Scanning...' : 'Scan for Devices' }}</span>
          </button>

          <!-- Scan error -->
          <div v-if="scanError" class="text-danger text-xs mb-3 text-center">{{ scanError }}</div>

          <!-- Device list from scan -->
          <div v-if="devices.length" class="mb-4 border border-gray-200 rounded-lg overflow-hidden">
            <div class="text-xs text-gray-500 px-3 pt-2 pb-1 font-medium uppercase tracking-wide bg-gray-50">
              Paired
            </div>
            <ul class="divide-y divide-gray-100 max-h-44 overflow-y-auto">
              <li v-for="device in devices" :key="device.address" @click="selectDevice(device)"
                class="flex items-center gap-3 px-3 py-2.5 cursor-pointer hover:bg-blue-50 transition-colors"
                :class="selectedMac === device.address ? 'bg-blue-50 ring-1 ring-inset ring-blue-400' : ''">
                <span class="w-4 h-4 shrink-0"
                  v-html="device.device === 'printer' ? printerIcon : bluetoothIcon"></span>
                <div class="flex-1 min-w-0">
                  <div class="text-sm font-medium text-gray-800 truncate">{{ device.name }}</div>
                  <div class="text-xs text-gray-400 font-mono">{{ device.address }}</div>
                </div>
                <span v-if="selectedMac === device.address" class="w-4 h-4 shrink-0 text-blue-500"
                  v-html="checkIcon"></span>
              </li>
            </ul>
          </div>
          <div v-if="error" class="text-danger text-sm mb-3">{{ error }}</div>

          <button @click="submit" :disabled="loading || (!macInput.trim() && !selectedMac)"
            class="w-full border rounded-lg px-4 py-2 cursor-pointer text-sm bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors">{{
              loading ? 'Connecting...' : 'Add Printer' }}
          </button>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import CloseButton from './close-button.vue'
import { bluetoothIcon, warningIcon, searchIcon, printerIcon, checkIcon } from '../components/printer-icons.js'
import { AddBluetoothPrinter, ScanBluetoothPrinters, CheckBluetoothDependencies } from '../../wailsjs/go/main/App'
import { useToast } from '../hooks/useToast.js'

const emit = defineEmits(['refresh'])
const { notify } = useToast()

const showBluetoothDialog = ref(false)
const macInput = ref('')
const nameInput = ref('')
const error = ref(null)
const loading = ref(false)
const scanning = ref(false)
const scanError = ref(null)
const devices = ref([])
const selectedMac = ref('')
const inputRef = ref(null)
const dependencies = ref([])

async function checkDeps() {
  try {
    const missing = await CheckBluetoothDependencies()
    if (missing && missing.length > 0) {
      dependencies.value = missing
      return
    }

    scan()
  } catch (err) {
    console.error("Failed to check Bluetooth dependencies:", err)
  }
}

function copyCmd(cmd) {
  navigator.clipboard.writeText(cmd)
}

watch(showBluetoothDialog, (val) => {
  if (val) {
    macInput.value = ''
    nameInput.value = ''
    error.value = null
    scanError.value = null
    devices.value = []
    selectedMac.value = ''
    dependencies.value = []
    checkDeps()
    nextTick(() => inputRef.value?.focus())
  }
})

function selectDevice(device) {
  selectedMac.value = device.address
  macInput.value = device.address
  nameInput.value = nameInput.value || device.name
}

async function scan() {
  scanning.value = true
  scanError.value = null
  devices.value = []
  try {
    const found = await ScanBluetoothPrinters()
    devices.value = found || []
    if (!devices.value.length) {
      scanError.value = 'No paired Bluetooth devices found. Pair your printer first via system Bluetooth settings.'
    }
  } catch (err) {
    scanError.value = err?.toString() || 'Scan failed'
  } finally {
    scanning.value = false
  }
}

function close(shouldRefresh = false) {
  error.value = null
  if (shouldRefresh) {
    emit("refresh")
  }
  showBluetoothDialog.value = false
}

async function submit() {
  const mac = (selectedMac.value || macInput.value).trim()
  if (!mac) {
    error.value = 'Please enter or scan a MAC address'
    return
  }

  loading.value = true
  error.value = null

  try {
    await AddBluetoothPrinter(mac, nameInput.value.trim())
    close(true)
    notify("Bluetooth printer added successfully", "success")
  } catch (err) {
    error.value = err?.toString() || 'Failed to add Bluetooth printer'
  } finally {
    loading.value = false
  }
}

</script>
