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
      <div v-if="modelValue" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4">
        <div class="absolute inset-0 bg-black/75" @click="close"/>
        <div class="relative bg-white rounded-2xl w-full max-w-md shadow-xl overflow-hidden">

          <div class="flex items-center justify-between px-5 pt-4">
            <p class="text-[12px] text-sm uppercase tracking-widest text-odoo">
              Step {{ currentStep + 1 }} of {{ steps.length }}
            </p>
            <CloseButton @click="close"/>
          </div>

          <div
              class="overflow-hidden transition-[height] duration-300 ease-in-out"
              :style="{ height: contentHeight }"
          >
            <transition
                enter-active-class="transition duration-300 ease-out"
                enter-from-class="opacity-0 translate-x-4"
                enter-to-class="opacity-100 translate-x-0"
                leave-active-class="transition duration-200 ease-in"
                leave-from-class="opacity-100 translate-x-0"
                leave-to-class="opacity-0 -translate-x-4"
                mode="out-in"
            >
              <div :key="currentStep" ref="contentRef" class="px-5 pt-2 pb-2">
                <h3 class="font-semibold text-lg text-gray-900 mb-1.5">{{ steps[currentStep].title }}</h3>
                <p class="text-gray-500 whitespace-pre-line">{{ steps[currentStep].desc }}</p>

                <a v-if="steps[currentStep].link" :href="steps[currentStep].link" target="_blank"
                   @click.prevent="BrowserOpenURL(steps[currentStep].link)"
                   class="inline-flex items-center mt-3 px-3 py-2 rounded-lg border border-stone-300 text-stone-600 hover:bg-stone-50 hover:border-stone-400"
                >
                  {{ steps[currentStep].linkLabel }}
                </a>
                <img
                    v-if="steps[currentStep].image"
                    :src="steps[currentStep].image"
                    :alt="steps[currentStep].title"
                    class="max-w-full mt-3 rounded-lg"
                />
                <div v-if="steps[currentStep].codes" v-for="(code, index) in steps[currentStep].codes" class="mt-3 relative group">
                  <pre class="bg-slate-800 text-emerald-500 text-sm rounded-lg px-4 py-3 overflow-x-auto font-mono">{{
                      code
                    }}</pre>
                  <button
                      class="absolute top-2.5 right-2 px-2 py-1 text-xs rounded-md bg-slate-700 text-slate-300 hover:bg-slate-600 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                      @click="copyCode(index,code)"
                  >
                    {{ codeCopied[index] ? '✓ Copied' : 'Copy' }}
                  </button>
                </div>
              </div>
            </transition>
          </div>

          <div class="px-5 pb-5 pt-3 flex gap-2">
            <button
                v-if="currentStep > 0"
                class="flex-1 py-2 rounded-lg border border-stone-300 text-stone-600 hover:bg-stone-50 hover:border-stone-400 cursor-pointer whitespace-nowrap transition-colors"
                @click="back">Back
            </button>
            <button
                v-if="currentStep < steps.length - 1"
                class="flex-1 py-2 rounded-lg bg-odoo text-white hover:bg-odoo-dark whitespace-nowrap cursor-pointer transition-colors"
                @click="next(steps.length)">Next
            </button>
            <button
                v-else
                class="flex-1 py-2 rounded-lg bg-emerald-500 text-white hover:bg-emerald-600 cursor-pointer transition-colors"
                @click="close">All done ✓
            </button>
          </div>

        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup>
import {nextTick, onUnmounted, ref, watch} from 'vue'
import {useStepModal} from './use-step-modal'
import {BrowserOpenURL} from "../../wailsjs/runtime"
import CloseButton from './close-button.vue'

const props = defineProps({
  modelValue: {type: Boolean, default: false},
  steps: {type: Array, default: () => []},
})

const emit = defineEmits(['update:modelValue'])
const {currentStep, next, back, close} = useStepModal(() => props.modelValue, emit)

const contentRef = ref(null)
const contentHeight = ref('auto')

const codeCopied = ref({})

async function copyCode(index,code) {
  await navigator.clipboard.writeText(code)
  codeCopied.value[index] = true
  setTimeout(() => codeCopied.value[index] = false, 2000)
}

const resizeObserver = new ResizeObserver(() => {
  if (contentRef.value) {
    contentHeight.value = `${contentRef.value.offsetHeight}px`
  }
})

watch(contentRef, (el) => {
  resizeObserver.disconnect()
  if (el) resizeObserver.observe(el)
})

watch(currentStep, () => {
  if (contentRef.value) {
    contentHeight.value = `${contentRef.value.offsetHeight}px`
  }
})

watch(() => props.modelValue, (val) => {
  if (val) nextTick(() => {
    if (contentRef.value) resizeObserver.observe(contentRef.value)
  })
})

onUnmounted(() => resizeObserver.disconnect())
</script>