import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import '@shared-ui/assets/styles/main.scss'
import './assets/widget.css'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.mount('#app')
