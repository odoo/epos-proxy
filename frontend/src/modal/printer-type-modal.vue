<template>
  <teleport to="body">
    <div v-if="modelValue" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-80 shadow-lg relative">
        <div class="flex items-center justify-between mb-4">
            <div class="text-lg font-semibold text-center">Select Printer Type</div>
            <CloseButton @click="close"/>
        </div>
        <div class="mb-3 mt-3 text-sm text-yellow-700 bg-yellow-100 p-2 rounded"> ⚠️ Selecting the wrong type may result
          in garbage print or no output.</div>
        <div class="flex gap-3">
          <button
            class="flex-1 bg-gray-200 rounded-lg py-2 hover:bg-odoo hover:text-white cursor-pointer disabled:bg-gray-300 disabled:text-gray-500 disabled:cursor-not-allowed disabled:hover:bg-gray-300"
            @click="select('THERMAL')" :disabled="selectedPrinter.variant === 'OFFICE'">
            Receipt/Label
          </button>
          <button
            class="flex-1 bg-gray-200 rounded-lg py-2 hover:bg-odoo hover:text-white cursor-pointer disabled:bg-gray-300 disabled:text-gray-500 disabled:cursor-not-allowed disabled:hover:bg-gray-300"
            @click="select('OFFICE')" :disabled="selectedPrinter.variant === 'THERMAL'">
            Office (PDF)
          </button>
        </div>
      </div>
    </div>
  </teleport>
</template>

<script setup id="h9m2qp">
import CloseButton from './close-button.vue'

const props = defineProps({ modelValue: Boolean, selectedPrinter: Object })
const emit = defineEmits(['update:modelValue', 'select'])
function close() { emit('update:modelValue', false) }

function select(type) {
  emit('select', type);
  close();
}
</script>
