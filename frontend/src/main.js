import './app.css'
import { createApp } from 'vue'
import PrinterList from './printer-list.vue'
import { Toast } from './hooks/useToast.js'

const app = createApp(PrinterList)
app.use(Toast)
app.mount('#app')
