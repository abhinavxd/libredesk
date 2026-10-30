import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import App from './App.vue'
import api from './api/index.js'
import { resolveLanguage } from './utils/language.js'
import '@shared-ui/assets/styles/main.scss'

async function initWidget () {
    try {
        // Get `inbox_id` from URL params
        const urlParams = new URLSearchParams(window.location.search)
        const inboxID = urlParams.get('inbox_id')
        const hostLang = urlParams.get('lang')

        if (!inboxID) {
            throw new Error('`inbox_id` is missing in query parameters')
        }

        // Fetch widget settings to get language config
        const widgetSettingsResponse = await api.getWidgetSettings(inboxID)
        const widgetConfig = widgetSettingsResponse.data.data

        const fallbackLang = widgetConfig.fallback_language || 'en-US'
        let availableCodes = []
        if (widgetConfig.language === 'auto') {
            const availableResp = await api.getAvailableLanguages()
            availableCodes = availableResp.data.data.map(l => l.code)
        }
        const browserLanguages = [navigator.language || navigator.languages?.[0] || '']
        const language = resolveLanguage(widgetConfig, hostLang, browserLanguages, availableCodes)
        const lang = language.locale

        // Fetch language messages
        const langMessages = await api.getLanguage(lang)

        // Initialize i18n
        const i18nConfig = {
            legacy: false,
            locale: lang,
            fallbackLocale: fallbackLang,
            messages: {
                [lang]: langMessages.data
            }
        }

        const i18n = createI18n(i18nConfig)
        const app = createApp(App)
        const pinia = createPinia()

        app.use(pinia)
        app.use(i18n)
        // Store widget config globally for access in App.vue
        app.config.globalProperties.$widgetConfig = widgetConfig
        app.config.globalProperties.$widgetLanguagePreference = language.requestedLocale
        app.mount('#app')
    } catch (error) {
        console.error('Error initializing widget:', error)
    }
}

initWidget()
