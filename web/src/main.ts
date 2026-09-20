import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerSW } from 'virtual:pwa-register'
import '@fontsource-variable/manrope'
import '@fontsource-variable/sora'
import App from './App.vue'
import { i18n } from './i18n'
import { router } from './router'
import './styles.css'

registerSW({ immediate: true })

createApp(App).use(createPinia()).use(i18n).use(router).mount('#app')
