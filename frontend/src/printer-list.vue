<template>
  <div>
    <div
        class="w-full max-w-full sm:max-w-md md:max-w-lg lg:max-w-xl bg-white/85 rounded-2xl shadow-lg overflow-hidden px-4 sm:px-6 py-2 sm:py-4">

      <div v-if="printers.length || unavailablePrinters.length" class="p-6">
        <ul class="divide-y divide-gray-300">

          <li v-for="printer in printers" :key="printer.id" class="text-left first:pt-0 py-6 last:pb-0 relative">

            <div class="flex items-center gap-2">
              <span class="w-3 h-3 rounded-full shrink-0" :class="getPrinterStatusClass(printer)"></span>
              <span class="min-w-0 font-medium text-gray-900 break-all flex-1">{{ printer.name }}</span>
              <span
                  v-if="printer.isLAN"
                  @click="removeLanPrinter(printer)"
                  class="text-gray-600 hover:text-danger cursor-pointer text-xl font-bold"
                  title="Remove printer"
              >×</span>
            </div>
            <div class="text-slate-600 mt-2 text-sm break-all">{{ printer.ip }}</div>
             <PrinterActions 
              :printer="printer"
              :copiedIds="copiedIds"
              :testPrintIds="testPrintIds"
              @copy="copyField"
              @test="testPrint" />
          </li>

          <li v-for="printer in unavailablePrinters" :key="printer.name"
              class="text-left first:pt-0 py-6 last:pb-0 relative">
            <div class="flex items-center gap-2">
              <span class="w-3 h-3 rounded-full shrink-0 bg-danger"></span>
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
        <div class="mt-2 text-gray-600 text-center">Make sure your printer is powered on and connected via USB.</div>
      </div>

      <div v-if="errorMsg">
        <div class="text-red-700 mt-4 text-center">Error: {{ errorMsg }}</div>
      </div>

      <StepModal v-model="showFixModal" :steps="fixSteps"/>

    </div>
  </div>
  <div class="mt-6 text-center">
    <div
        @click="showAddDialog = true"
        class="border-2 border-dashed border-gray-300 bg-gray-50 rounded-lg px-4 py-3 text-gray-600 hover:border-gray-400 hover:bg-gray-100 cursor-pointer"
    >+ Add Network Printer
    </div>
  </div>

  <NetworkIpDialog :show="showAddDialog" @close="onNetworkDialogClose"/>

  <teleport to="body">
    <transition
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="opacity-0 translate-x-4"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 translate-x-4"
    >
      <div
          v-if="toast.show"
          class="fixed top-4 right-4 z-50 px-4 py-3 rounded-lg shadow-lg text-white text-sm max-w-xs"
          :class="toast.type === 'success' ? 'bg-success' : 'bg-danger'"
      >
        {{ toast.message }}
      </div>
    </transition>
  </teleport>
</template>

<script setup>
import {computed, onMounted, onUnmounted, ref} from 'vue'
import {CheckLANPrinterStatus, ConfirmRemoveLANPrinter, Status} from '../wailsjs/go/main/App'
import {brewSteps, linuxSteps, zadigSteps} from "./modal/fix-step";
import StepModal from "./modal/step-modal.vue";
import NetworkIpDialog from "./modal/network-ip-dialog.vue";
import PrinterActions from './components/printer-actions.vue'
import { copyPrinterFieldValue, handleTestPrint } from "./components/printer-actions.js";

const printers = ref([])
const unavailablePrinters = ref([])
const errorMsg = ref(null)
const loading = ref(true)
const copiedIds = ref({})
const lanStatus = ref({})
const pendingChecks = ref(new Set())
const showFixModal = ref(false)
const fixPrinterName = ref(null)
const testPrintIds = ref({})
const os = ref(null)
const showAddDialog = ref(false)
const toast = ref({ show: false, message: '', type: 'success' })

let toastTimeout = null
let intervalId = null
let isTabVisible = true
let isUpdating = false

const copyField = (printer, field) => copyPrinterFieldValue(printer, field, { copiedIds, showToast })
const testPrint = (printer) => handleTestPrint(printer, { testPrintIds, showToast })

const handleVisibilityChange = () => {
  isTabVisible = !document.hidden
  if (isTabVisible) updatePrinters()
}

function updatePrinters() {
  if (isUpdating) return

  isUpdating = true
  Status().then((res) => {
    printers.value = res.printers
    unavailablePrinters.value = res.unavailablePrinters
    errorMsg.value = res.errorMsg
    os.value = res.os
    loading.value = false

    // Check status for each LAN printer
    for (const printer of res.printers) {
      if (printer.isLAN && printer.lanIp) {
        checkLanPrinterStatus(printer.lanIp)
      }
    }
  }).finally(() => {
    isUpdating = false
  })
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

function getPrinterStatusClass(printer) {
  if (!printer.isLAN) {
    return printer.online ? 'bg-success' : 'bg-danger'
  }
  const status = lanStatus.value[printer.lanIp]
  if (status === 'online') return 'bg-success'
  if (status === 'offline') return 'bg-danger'
  return 'bg-warning'
}

onMounted(() => {
  isTabVisible = true
  document.addEventListener('visibilitychange', handleVisibilityChange)
  updatePrinters()
  intervalId = setInterval(() => {
    if (isTabVisible) updatePrinters()
  }, 5000)
})

onUnmounted(() => {
  clearInterval(intervalId)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})

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

function showToast(message, type = 'success') {
  if (toastTimeout) clearTimeout(toastTimeout)
  toast.value = { show: true, message, type }

  toastTimeout = setTimeout(() => {
    toast.value.show = false
  }, type === 'success' ? 2000: 3000)
}

async function removeLanPrinter(printer) {
  if (!printer.lanIp) return

  try {
    const removed = await ConfirmRemoveLANPrinter(printer.lanIp)
    if (removed) updatePrinters()
  } catch (err) {
    console.error('Failed to remove LAN printer:', err)
  }
}

function onNetworkDialogClose(shouldRefresh) {
  showAddDialog.value = false
  if (shouldRefresh) updatePrinters()
}
</script>
