import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { initI18n } from './i18n'
import { useAppSettingsStore } from './stores/appSettings'
import router from './router'
import mitt from 'mitt'
import api from './api'
import '@shared-ui/assets/styles/main.scss'
import '@shared-ui/utils/string.js'
import Root from './Root.vue'

const applyLocaleFont = async (lang) => {
  const normalized = (lang || '').toLowerCase()
  const isPersian = normalized === 'fa' || normalized.startsWith('fa-')
  document.documentElement.classList.toggle('is-fa', isPersian)
  if (isPersian) {
    const links = [
      {
        id: 'libredesk-font-vazir-400',
        href: 'https://cdn.jsdelivr.net/npm/@fontsource/vazir@4.5.4/farsi-digits-400.css'
      },
      {
        id: 'libredesk-font-vazir-700',
        href: 'https://cdn.jsdelivr.net/npm/@fontsource/vazir@4.5.4/farsi-digits-700.css'
      }
    ]

    links.forEach(({ id, href }) => {
      if (document.getElementById(id)) return
      const link = document.createElement('link')
      link.id = id
      link.rel = 'stylesheet'
      link.href = href
      document.head.appendChild(link)
    })
  }
}

const setFavicon = (url) => {
  let link = document.createElement("link")
  link.rel = "icon"
  document.head.appendChild(link)
  link.href = url
}

async function initApp () {
  const config = (await api.getConfig()).data.data
  const emitter = mitt()
  const lang = config['app.lang'] || 'en-US'
  const langMessages = await api.getLanguage(lang)

  await applyLocaleFont(lang)

  // Set favicon.
  if (config['app.favicon_url'])
    setFavicon(config['app.favicon_url'])

  // Initialize i18n.
  const i18nConfig = {
    legacy: false,
    locale: lang,
    fallbackLocale: 'en-US',
    messages: {
      [lang]: langMessages.data
    }
  }

  const i18n = initI18n(i18nConfig)
  const app = createApp(Root)
  const pinia = createPinia()
  app.use(pinia)

  // Fetch and store app settings in store (after pinia is initialized)
  const settingsStore = useAppSettingsStore()

  // Store the public config in the store
  settingsStore.setPublicConfig(config)

  try {
    await settingsStore.fetchSettings('general')
  } catch (error) {
    // Pass
  }

  // Add emitter to global properties.
  app.config.globalProperties.emitter = emitter

  app.use(router)
  app.use(i18n)
  app.mount('#app')
}

initApp().catch((error) => {
  console.error('Error initializing app: ', error)
})
