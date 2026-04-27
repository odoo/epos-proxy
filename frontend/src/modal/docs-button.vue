<template>
    <button @click="openDocs" name="Docs" aria-label="Documentation" title="Help" class="absolute top-3 right-3
               w-12 h-12 flex items-center justify-center
               rounded-full
               cursor-pointer
               bg-odoo text-white
               shadow-lg shadow-odoo/30
               hover:shadow-odoo/50 hover:scale-105
               active:scale-95
               transition-all duration-300 ease-out
               backdrop-blur-md">

        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" fill="currentColor" viewBox="0 0 16 16">
            <path
                d="m8.93 6.588-2.29.287-.082.38.45.083c.294.07.352.176.288.469l-.738 3.468c-.194.897.105 1.319.808 1.319.545 0 1.178-.252 1.465-.598l.088-.416c-.2.176-.492.246-.686.246-.275 0-.375-.193-.304-.533zM9 4.5a1 1 0 1 1-2 0 1 1 0 0 1 2 0" />
        </svg>
    </button>
    <StepModal v-model="showFixModal" :allSteps="fixSteps" />
</template>
<script setup>
import { computed, ref } from 'vue'
import { macSteps, linuxSteps, windowsSteps } from "./fix-step";
import StepModal from "./step-modal.vue";

const showFixModal = ref(false)
const props = defineProps({
    os: {
        type: String,
        default: null
    }
})

const fixSteps = computed(() => {
    if (!showFixModal.value) return {}
    if (isWindows()) return windowsSteps()
    if (isMac()) return macSteps()
    if (isLinux()) return linuxSteps()
    return {}
})

const openDocs = () => showFixModal.value = true
const isWindows = () => props.os && props.os.toLowerCase().includes('windows')
const isMac = () => props.os && props.os.toLowerCase().includes('darwin')
const isLinux = () => props.os && props.os.toLowerCase().includes('linux')

</script>
