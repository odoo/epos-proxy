<template>
  <div>
    <div
        class="w-full max-w-full sm:max-w-md md:max-w-lg lg:max-w-xl bg-white/85 rounded-2xl shadow-lg overflow-hidden px-4 sm:px-6 py-2 sm:py-4">

      <div v-if="printers.length || unavailablePrinters.length" class="p-6">
        <ul class="divide-y divide-gray-300">

          <li v-for="printer in printers" :key="printer.id" class="text-left first:pt-0 py-6 last:pb-0 relative">

            <div class="flex items-center gap-2">
              <span class="w-4 h-4 shrink-0" :class="getPrinterStatusClass(printer)" v-html="getPrinterIcon(printer)"></span>
              <span class="min-w-0 font-medium text-gray-900 break-all flex-1">{{ printer.name }}</span>
              <span
                  v-if="printer.isBT"
                  @click="removeBluetoothPrinter(printer)"
                  class="text-gray-600 hover:text-danger cursor-pointer text-xl font-bold"
                  title="Remove Bluetooth printer"
              >×</span>
              <span
                  v-else-if="printer.isLAN"
                  @click="removeLanPrinter(printer)"
                  class="text-gray-600 hover:text-danger cursor-pointer text-xl font-bold"
                  title="Remove printer"
              >×</span>
            </div>
            <div class="text-slate-600 mt-2 text-sm break-all">{{ printer.ip }}</div>
            <PrinterActions :printer="printer" />
          </li>

          <li v-for="printer in unavailablePrinters" :key="printer.name"
              class="text-left first:pt-0 py-6 last:pb-0 relative">
            <div class="flex items-center gap-2">
              <span class="w-4 h-4 shrink-0 text-danger" v-html="usbIcon"></span>
              <span class="min-w-0 font-medium text-gray-900">{{ printer.name }}</span>
            </div>
            <div class="text-danger mt-1 text-wrap">Unable to communicate with this printer: {{
                printer.errorMsg
              }}
            </div>
            <div v-if="hasLibUsbErrorFix(printer.errorMsg)" class="flex gap-2 mt-4 flex-wrap">
              <button
                  class="flex-1 border bg-odoo text-white hover:bg-odoo-dark rounded-lg px-4 py-2 text-center cursor-pointer"
                  @click="openFixModal(printer)"
              >{{ getFixErrorText(printer.errorMsg) }}
              </button>
            </div>
          </li>

        </ul>
      </div>

      <div v-if="loading" class="p-6">
        <div class="font-medium text-lg text-center">Searching for printers...</div>
      </div>
      <div v-else-if="!printers.length && !unavailablePrinters.length" class="p-6">
        <div class="font-medium text-lg text-center">No printers found</div>
        <div class="mt-2 text-gray-600 text-center">Make sure your printer is powered on and connected via USB, Network, or Bluetooth.</div>
      </div>

      <div v-if="errorMsg">
        <div class="text-red-700 mt-4 text-center">Error: {{ errorMsg }}</div>
      </div>

      <StepModal v-model="showFixModal" :steps="fixSteps"/>

    </div>
  </div>
  <div class="mt-6 text-center flex flex-row gap-2">
    <div
        @click="showAddDialog = true"
        class="flex-1 border-2 border-dashed border-gray-300 bg-gray-50 rounded-lg px-4 py-3 text-gray-600 hover:border-gray-400 hover:bg-gray-100 cursor-pointer flex items-center justify-center gap-2 transition-colors"
    >
        <span class="w-4 h-4 shrink-0" v-html="wifiIcon"></span>
        Add Network Printer
    </div>
    <BluetoothDialog @refresh="updatePrinters"/>
  </div>

  <NetworkIpDialog :show="showAddDialog" @close="onNetworkDialogClose"/>
</template>

<script setup>
import {computed, onMounted, onUnmounted, ref} from 'vue'
import { getPrinterIcon, usbIcon, wifiIcon } from './components/printer-icons.js'
import { CheckLANPrinterStatus, CheckBluetoothPrinterStatus, ConfirmRemoveLANPrinter, ConfirmRemoveBluetoothPrinter, Status } from '../wailsjs/go/main/App'
import {brewSteps, linuxSteps, zadigSteps} from "./modal/fix-step";
import StepModal from "./modal/step-modal.vue";
import NetworkIpDialog from "./modal/network-ip-dialog.vue";
import PrinterActions from './components/printer-actions.vue'
import { useToast } from './hooks/useToast.js'
import BluetoothDialog from "./modal/bluetooth-dialog.vue";

const { notify } = useToast()

const printers = ref([])
const unavailablePrinters = ref([])
const errorMsg = ref(null)
const loading = ref(true)
const lanStatus = ref({})
const pendingChecks = ref(new Set())
const showFixModal = ref(false)
const fixPrinterName = ref(null)
const os = ref(null)
const showAddDialog = ref(false)

let intervalId = null
let isUpdating = false

const handleVisibilityChange = () => document.hidden ? stopPolling() : startPolling()
const handleFocus = () => startPolling();
const handleBlur = () => stopPolling();

async function updatePrinters() {
  if (isUpdating) return

  isUpdating = true
  try {
    const res = await Status()
    printers.value = res.printers
    unavailablePrinters.value = res.unavailablePrinters
    errorMsg.value = res.errorMsg
    os.value = res.os
    loading.value = false

    // Check status for each LAN and Bluetooth printer
    for (const printer of res.printers) {
      if (printer.isLAN && printer.lanIp) {
        checkLanPrinterStatus(printer.lanIp)
      }
      if (printer.isBT && printer.btMac) {
        checkBtPrinterStatus(printer.btMac)
      }
    }

  } catch (error) {
    console.error('Failed to update printers:', error)
    errorMsg.value = 'Failed to retrieve printer status. Please try again.'
  } finally {
    isUpdating = false
  }
}

function checkLanPrinterStatus(ip) {
  if (pendingChecks.value.has(ip)) return

  pendingChecks.value.add(ip)
  if (lanStatus.value[ip] === undefined) {
    lanStatus.value[ip] = 'loading'
  }
  CheckLANPrinterStatus(ip).then((online) => {
    lanStatus.value[ip] = online ? 'online' : 'offline'
  }).finally(() => {
    pendingChecks.value.delete(ip)
  })
}

const btStatus = ref({})

function checkBtPrinterStatus(mac) {
  const key = `bt:${mac}`
  if (pendingChecks.value.has(key)) return

  pendingChecks.value.add(key)
  if (btStatus.value[mac] === undefined) {
    btStatus.value[mac] = 'loading'
  }
  CheckBluetoothPrinterStatus(mac).then((online) => {
    btStatus.value[mac] = online ? 'online' : 'offline'
  }).finally(() => {
    pendingChecks.value.delete(key)
  })
}

function getPrinterStatusClass(printer) {
  if (printer.isBT) {
    const status = btStatus.value[printer.btMac]
    if (status === 'online') return 'text-success'
    if (status === 'offline') return 'text-danger'
    return 'text-warning'
  }
  if (!printer.isLAN) {
    return printer.online ? 'text-success' : 'text-danger'
  }
  const status = lanStatus.value[printer.lanIp]
  if (status === 'online') return 'text-success'
  if (status === 'offline') return 'text-danger'
  return 'text-warning'
}

onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('focus', handleFocus)
  window.addEventListener('blur', handleBlur)

  if (!document.hidden) startPolling()
})

onUnmounted(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('focus', handleFocus)
  window.removeEventListener('blur', handleBlur)
})

const startPolling = () => {
  if (intervalId) return
  updatePrinters()
  intervalId = setInterval(updatePrinters, 5000)
}

const stopPolling = () => {
  if (!intervalId) return
  clearInterval(intervalId)
  intervalId = null
}

const fixSteps = computed(() => {
  if (!showFixModal.value) {
    return []
  }

  if (isWindows()) return zadigSteps(fixPrinterName.value)
  if (isMac()) return brewSteps(fixPrinterName.value)
  if (isLinux()) return linuxSteps(fixPrinterName.value)
  return []
})

function hasLibUsbErrorFix(error="") {
  return true
  return error.toLowerCase().includes('libusb')
}


function isWindows() {
  return os.value && os.value.toLowerCase().includes('windows')
}

function isMac() {
  return os.value && os.value.toLowerCase().includes('darwin')
}

function isLinux() {
  return os.value && os.value.toLowerCase().includes('linux')
}

function getFixErrorText() {

  if (isWindows()) {
    return 'Fix - Install WinUSB driver'
  }

  if (isMac() || isLinux()) {
    return 'Fix - Install libusb'
  }

}

function openFixModal(printer) {
  fixPrinterName.value = printer.name
  showFixModal.value = true
}

async function removeLanPrinter(printer) {
  if (!printer.lanIp) return

  try {
    const removed = await ConfirmRemoveLANPrinter(printer.lanIp)
    if (removed) {
      updatePrinters()
      notify('Printer removed successfully', 'success')
    }
  } catch (err) {
    console.error('Failed to remove LAN printer:', err)
    notify('Failed to remove printer', 'danger')
  }
}

async function removeBluetoothPrinter(printer) {
  if (!printer.btMac) return

  try {
    const removed = await ConfirmRemoveBluetoothPrinter(printer.btMac)
    if (removed) {
      updatePrinters()
      notify('Printer removed successfully', 'success')
    }
  } catch (err) {
    console.error('Failed to remove Bluetooth printer:', err)
    notify('Failed to remove printer', 'danger')
  }
}

function onNetworkDialogClose(shouldRefresh) {
  showAddDialog.value = false
  if (shouldRefresh) updatePrinters()
}

</script>
