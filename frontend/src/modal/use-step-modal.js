import {ref, watch} from 'vue'

export function useStepModal(modelValue, emit) {
    const currentStep = ref(0)
    const transitioning = ref(false)
    let clickTimeout = null

    watch(modelValue, (val) => {
        if (val) currentStep.value = 0
    })

    function onNextStep() {
        transitioning.value = true
        clearTimeout(clickTimeout)
        clickTimeout = setTimeout(() => {
            transitioning.value = false
        }, 600)
    }

    function next(stepsLength) {
        if (transitioning.value) return
        if (currentStep.value < stepsLength - 1) {
            currentStep.value++
            onNextStep()
        }
    }

    function back() {
        if (transitioning.value) return
        currentStep.value--
        onNextStep()
    }

    function close() {
        if (transitioning.value) return
        emit('update:modelValue', false)
    }

    return {currentStep, transitioning, next, back, close}
}