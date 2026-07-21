<template>
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
defineProps({
  toast: {
    type: Object,
    required: true,
    shape: { show: Boolean, message: String, type: 'success' | 'danger' }
  },
})

</script>
