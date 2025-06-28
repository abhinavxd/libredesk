import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { useAppSettingsStore } from './stores/appSettings'
import router from './router'
import mitt from 'mitt'
import api from './api'
import './assets/styles/main.scss'
import './utils/strings.js'
import VueDOMPurifyHTML from 'vue-dompurify-html'
import Root from './Root.vue'

const setFavicon = (url) => {
  let link = document.createElement("link")
  link.rel = "icon"
  document.head.appendChild(link)
  link.href = url
}

async function initApp () {
  const settings = (await api.getSettings('general')).data.data
  const emitter = mitt()
  const lang = settings['app.lang'] || 'en'
  const langMessages = await api.getLanguage(lang)

  // Set favicon.
  if (settings['app.favicon_url'])
    setFavicon(settings['app.favicon_url'])

  // Initialize i18n.
  const i18nConfig = {
    legacy: false,
    locale: lang,
    fallbackLocale: 'en',
    messages: {
      [lang]: langMessages.data
    }
  }

  const i18n = createI18n(i18nConfig)
  const app = createApp(Root)
  const pinia = createPinia()
  app.use(pinia)

  // Store app settings in Pinia
  const settingsStore = useAppSettingsStore()
  settingsStore.setSettings(settings)

  // Add emitter to global properties.
  app.config.globalProperties.emitter = emitter

  app.use(router)
  app.use(i18n)
  app.use(VueDOMPurifyHTML)
  app.mount('#app')
}

initApp().catch((error) => {
  console.error('Error initializing app: ', error)
})