import { createApp, reactive } from 'vue'
import ToastNotification from '../components/toast-notification.vue'

// Singleton reactive object — reactive() (not ref()) so it can be passed as a
// createApp prop and still be reactive in the child app without auto-unwrapping.
const toast = reactive({ show: false, message: '', type: 'success' })
let timer = null

function notify(message, type = 'success') {
  if (timer) clearTimeout(timer)
  toast.show = true
  toast.message = message
  toast.type = type
  timer = setTimeout(() => {
    toast.show = false
  }, type === 'success' ? 2000 : 3000)
}

/**
 * Vue plugin — register with `app.use(Toast)`.
 * Mounts <ToastNotification> into its own DOM node (always on top) and
 * makes notify() available to every component via useToast().
 */
export const Toast = {
  install(app) {
    const toastHost = document.createElement('div')
    document.body.appendChild(toastHost)
    // Pass the reactive object directly — its reference is stable so the
    // child app will observe property mutations correctly.
    createApp(ToastNotification, { toast }).mount(toastHost)
  },
}

/**
 * Call in any component to obtain the notify(message, type) function.
 */
export function useToast() {
  return { notify }
}
