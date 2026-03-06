import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import vuePlugin from "@vitejs/plugin-vue";
export default defineConfig({
    plugins: [
        tailwindcss(),
        vuePlugin()
    ],
})