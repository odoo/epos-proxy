<template>
  <teleport to="body">
    <transition
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="opacity-0"
        enter-to-class="opacity-100"
        leave-active-class="transition duration-150 ease-in"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0"
    >
      <div v-if="show" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/75" @click="close"/>
        <div class="relative bg-white rounded-2xl w-full max-w-sm shadow-xl overflow-hidden p-6">

          <div class="flex items-center justify-between mb-4">
            <div class="text-lg font-medium">Add Network Printer</div>
            <CloseButton @click="close"/>
          </div>
          <input
              v-model="ipInput"
              type="text"
              placeholder="IP Address (e.g. 192.168.1.100)"
              class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-odoo-light focus:border-transparent mb-3"
              @keyup.enter="submit"
              ref="inputRef"
          />
          <div class="mb-4">
          <div class="text-sm font-medium text-gray-700 mb-2">Printer Type</div>
          <div class="flex gap-3">
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                value="THERMAL"
                v-model="printerType"
                class="accent-odoo"
              />
              <span class="text-sm">Receipt/Label</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                value="OFFICE"
                v-model="printerType"
                class="accent-odoo"
              />
              <span class="text-sm">Office (PDF)</span>
            </label>
          </div>
        </div>
          <div v-if="error" class="text-danger text-sm mb-3">{{ error }}</div>
          <button
              @click="submit"
              :disabled="loading"
              class="w-full border rounded-lg px-4 py-2 cursor-pointer text-sm bg-odoo text-white hover:bg-odoo-dark disabled:opacity-50 disabled:cursor-not-allowed"
          >{{ loading ? 'Adding...' : 'Add' }}</button>

        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup>
import {ref, watch, nextTick} from 'vue'
import CloseButton from './close-button.vue'
import {AddLANPrinter} from '../../wailsjs/go/main/App'

const props = defineProps({
  show: {type: Boolean, default: false},
  showToast: Function,
})

const emit = defineEmits(['close'])

const showToast = props.showToast
const ipInput = ref('')
const error = ref(null)
const loading = ref(false)
const inputRef = ref(null)
const printerType = ref('THERMAL')

watch(() => props.show, (val) => {
  if (val) {
    ipInput.value = ''
    error.value = null
    printerType.value = 'THERMAL'
    nextTick(() => inputRef.value?.focus())
  }
})

function close(shouldRefresh = false) {
  error.value = null
  emit('close', shouldRefresh)
}

async function submit() {
  const ip = ipInput.value.trim()
  if (!ip) {
    error.value = 'Please enter an IP address'
    return
  }

  loading.value = true
  error.value = null

  try {
    await AddLANPrinter(ip, printerType.value)
    showToast('Printer added successfully')
    close(true)
  } catch (err) {
    console.log(err)
    showToast('Failed to add printer', 'danger')
    error.value = err || 'Failed to add printer'
  } finally {
    loading.value = false
  }
}
</script>
