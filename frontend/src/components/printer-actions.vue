<template>
  <div class="flex gap-2 mt-4 flex-wrap">
    <button @click="onCopy" class="flex-1 border text-sm rounded-lg px-3 py-2 cursor-pointer whitespace-nowrap" :class="copiedIds?.[printer.id]?.ip
      ? 'bg-success text-white'
      : 'bg-odoo text-white hover:bg-odoo-dark'">
      {{ copiedIds?.[printer.id]?.ip ? '✓ Copied!' : 'Copy IP' }}
    </button>

    <button @click="onTest" :disabled="testPrintIds?.[printer.id]"
      class="flex-1 border rounded-lg text-sm px-3 py-2 cursor-pointer border-stone-300 text-stone-600 hover:bg-stone-50 hover:border-stone-400">
      {{ testPrintIds?.[printer.id] ? 'Printing...' : 'Test' }}
    </button>
  </div>
</template>
<script setup>
const props = defineProps({
  printer: Object,
  copiedIds: Object,
  testPrintIds: Object
})

const emit = defineEmits(['copy', 'test'])

function onCopy() { emit('copy', props.printer) }

function onTest() { emit('test', props.printer, props.printer.type) }

</script>
