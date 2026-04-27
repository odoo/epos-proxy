<template>
  <div>
    <div
        class="w-full max-w-full sm:max-w-md md:max-w-lg lg:max-w-xl bg-white/65 rounded-2xl shadow-lg overflow-hidden px-4 sm:px-6 py-2 sm:py-4">

      <PrinterFilter v-model="activeFilter" @refresh="updatePrinters" />
      <DocsButton :os="os"/>
      <div v-if="printers.length" class="p-6 overflow-y-auto max-h-[60vh]">
        <ul class="divide-y divide-gray-300">

          <li v-for="printer in printers" :key="printer.id" class="text-left first:pt-0 py-6 last:pb-0 relative">

            <div class="flex items-center gap-2">
              <span class="w-3 h-3 rounded-full shrink-0" :class="getPrinterStatusClass(printer)"></span>
              <div class="flex items-center gap-2 w-full">
                <span class="font-medium text-gray-900 truncate">
                  {{ printer.name }}
                </span>
                <button
                  @click="copyField(printer, 'name')"
                  class="flex-shrink-0 text-xs px-1 py-1 rounded cursor-pointer transition-colors duration-200 whitespace-nowrap"
                  :class="copiedIds[printer.id]?.name 
                    ? 'text-green-700 bg-green-100' 
                    : 'text-blue-800 border-blue-800 hover:text-blue-600 hover:bg-blue-100'"
                >
                  {{ copiedIds[printer.id]?.name ? 'Copied ✓' : 'Copy' }}
                </button>
              </div>
              <span v-if="printer.label" class="px-2 py-1 text-xs font-semibold rounded" :class="printer.type === 'ANY'
                ? 'bg-yellow-500 text-gray-800'
                : 'bg-green-500 text-gray-800'">
                {{ printer.label }}
              </span>
              <span
                  v-if="printer.isLAN || String(printer.name).startsWith('PDF_NETWORK_')"
                  @click="removeLanPrinter(printer)"
                  class="text-gray-600 hover:text-danger cursor-pointer text-xl font-bold"
                  title="Remove printer"
              >×</span>
            </div>
            <div class="mt-2 text-sm break-all transition-colors duration-200 flex items-center gap-2">
              <span class="select-all">{{ printer.ip }}</span>
            </div>
            <PrinterActions 
              :printer="printer"
              :copiedIds="copiedIds"
              :testPrintIds="testPrintIds"
              @copy="copyField"
              @test="testPrint" />
          </li>
        </ul>
      </div>

      <div v-if="loading" class="p-6">
        <div class="font-medium text-lg text-center">Searching for printers...</div>
      </div>
      <div v-else-if="!printers.length" class="p-6">
        <div class="font-medium text-lg text-center">No printers found</div>
        <div class="mt-2 text-gray-600 text-center">Make sure your printer is powered on, properly connected
          (USB/Wi-Fi) or <b>change the category.</b>
        </div>
      </div>

      <div v-if="errorMsg">
        <div class="text-red-700 mt-4 text-center">Error: {{ errorMsg }}</div>
      </div>
    </div>
  </div>
  <PrinterTypeModal v-if="showTypeSelect && selectedPrinter" v-model="showTypeSelect" :selectedPrinter="selectedPrinter"
    @select="selectType" />
  <div class="mt-6 text-center">
    <div
        @click="showAddDialog = true"
        class="border-2 border-dashed border-gray-300 bg-gray-50 rounded-lg px-4 py-3 text-gray-600 hover:border-gray-400 hover:bg-gray-100 cursor-pointer"
    >+ Add Network Printer
    </div>
  </div>

  <NetworkIpDialog :show="showAddDialog" @close="onNetworkDialogClose" :showToast="showToast"/>

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
import { onMounted, onUnmounted, ref } from 'vue'
import { CheckLANPrinterStatus, ConfirmRemoveLANPrinter, Status, ConfirmRemoveSystemPrinter } from '../wailsjs/go/main/App'
import NetworkIpDialog from "./modal/network-ip-dialog.vue";
import PrinterActions from './components/printer-actions.vue'
import { copyPrinterFieldValue, handleTestPrint } from "./components/printer-actions.js";
import PrinterFilter from './components/top-bar.vue'
import PrinterTypeModal from "./modal/printer-type-modal.vue";
import DocsButton from "./modal/docs-button.vue";

const printers = ref([])
const errorMsg = ref(null)
const loading = ref(true)
const copiedIds = ref({})
const lanStatus = ref({})
const pendingChecks = ref(new Set())
const testPrintIds = ref({})
const os = ref(null)
const showAddDialog = ref(false)
const toast = ref({ show: false, message: '', type: 'success' })
const activeFilter = ref('THERMAL')
const showTypeSelect = ref(false)
const selectedPrinter = ref(null)

let toastTimeout = null
let intervalId = null
let isUpdating = false

const copyField = (printer, field) => copyPrinterFieldValue(printer, field, { copiedIds, showToast })
const testPrint = (printer, type) => handleTestPrint(printer, type, { testPrintIds, selectedPrinter, showTypeSelect, showToast })

const handleFocus = () => startPolling();
const handleBlur = () => stopPolling();

function selectType(type) {
  showTypeSelect.value = false
  if (selectedPrinter.value) {
    testPrint(selectedPrinter.value, type)
  }
}

async function updatePrinters() {
  if (isUpdating) return

  isUpdating = true
  try {
    const res = await Status(activeFilter.value)
    printers.value = res.printers.filter(p => p.type === activeFilter.value)
    errorMsg.value = res.errorMsg
    os.value = res.os
    loading.value = false

    // Check status for each LAN printer
    for (const printer of res.printers) {
      if (printer.isLAN && printer.lanIp) {
        checkLanPrinterStatus(printer.lanIp)
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
  window.addEventListener('focus', handleFocus)
  window.addEventListener('blur', handleBlur)

  if (document.hasFocus()) startPolling();
})

onUnmounted(() => {
  stopPolling()
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

function showToast(message, type = 'success') {
  if (toastTimeout) clearTimeout(toastTimeout)
  toast.value = { show: true, message, type }

  toastTimeout = setTimeout(() => {
    toast.value.show = false
  }, type === 'success' ? 2000: 3000)
}

async function removeLanPrinter(printer) {
  if (!printer.lanIp && !String(printer.name).startsWith("PDF_NETWORK_")) return

  try {
    const removed = printer.lanIp ? await ConfirmRemoveLANPrinter(printer.lanIp) : await ConfirmRemoveSystemPrinter(printer.name);
    if (removed) {
      updatePrinters()
      showToast('Printer removed successfully')
    } else {
      showToast('Failed to remove printer', 'danger')
    }
  } catch (err) {
    showToast('Failed to remove printer', 'danger')
    console.error('Failed to remove LAN printer:', err)
  }
}

function onNetworkDialogClose(shouldRefresh) {
  showAddDialog.value = false
  if (shouldRefresh) updatePrinters()
}
</script>
