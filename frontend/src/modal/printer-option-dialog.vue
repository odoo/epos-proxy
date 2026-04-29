<template>
  <teleport to="body">
    <div v-if="modelValue" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-80 shadow-lg relative">

        <div class="flex items-center justify-between mb-4">
          <div class="text-lg font-semibold text-center">Select Print Mode</div>
          <CloseButton @click="close" />
        </div>

        <div class="flex gap-3">
          <button class="flex-1 bg-gray-200 rounded-lg py-2 hover:bg-odoo hover:text-white cursor-pointer"
            @click="select('single')">
            Single Side
          </button>

          <button class="flex-1 bg-gray-200 rounded-lg py-2 hover:bg-odoo hover:text-white cursor-pointer"
            @click="select('duplex')">
            Duplex
          </button>
        </div>

      </div>
    </div>
  </teleport>
</template>

<script setup>
import CloseButton from './close-button.vue'

const props = defineProps({ modelValue: Boolean })
const emit = defineEmits(['update:modelValue', 'select'])

const close = () => emit('update:modelValue', false)
const select = (mode) => {
  emit('select', mode)
  close()
}
</script>
