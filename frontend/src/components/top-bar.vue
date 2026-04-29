<template>
  <div class="bg-white border-b border-gray-200">
    <div class="px-6 flex items-center justify-evenly">
      <div class="flex -mb-px">
        <button v-for="tab in tabs" :key="tab.value" @click="setFilter(tab.value)" :class="tabClass(tab.value)">
          {{ tab.label }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
const emit = defineEmits(['refresh'])
const modelValue = defineModel({ type: String, default: 'THERMAL' })
const tabs = [
  { label: 'Receipt/Label', value: 'THERMAL' },
  { label: 'Office', value: 'OFFICE' },
  { label: 'Other', value: 'ANY' }
]

function setFilter(value) {
  modelValue.value = value
  emit('refresh')
}

const tabClass = (value) => {
  return [
    'px-8 py-4 text-sm transition-all relative cursor-pointer',
    modelValue.value === value
      ? 'text-odoo-dark font-bold border-b-2 border-odoo-dark'
      : 'text-gray-500 font-medium hover:text-gray-700 border-transparent hover:border-gray-300'
  ]
}
</script>
