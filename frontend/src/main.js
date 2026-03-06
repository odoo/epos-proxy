import './app.css';
import PrinterList from "./printer-list.vue";
import {createApp} from 'vue'

const app = createApp(PrinterList)
app.mount('#app')