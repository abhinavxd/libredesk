import { defineStore } from 'pinia'
import api from '@/api'

export const DEFAULT_SITE_TITLE = 'Fernmail'

// Signed-in settings win; the public config covers the login screen.
const brandSetting = (state, key) => state.settings[key] ?? state.public_config[key]

export const useAppSettingsStore = defineStore('settings', {
    state: () => ({
        settings: {},
        public_config: {}
    }),
    getters: {
        // A blank title or logo means the Fernmail default.
        siteTitle: (state) => brandSetting(state, 'app.site_name')?.trim() || DEFAULT_SITE_TITLE,
        siteLogo: (state) => brandSetting(state, 'app.logo_url') || ''
    },
    actions: {
        async fetchSettings (key = 'general') {
            try {
                const response = await api.getSettings(key)
                this.settings = response?.data?.data || {}
                return this.settings
            } catch (error) {
                // Pass
            }
        },
        async fetchPublicConfig () {
            try {
                const response = await api.getConfig()
                this.public_config = response?.data?.data || {}
                return this.public_config
            } catch (error) {
                // Pass
            }
        },
        setSettings (newSettings) {
            this.settings = newSettings
        },
        setPublicConfig (newPublicConfig) {
            this.public_config = newPublicConfig
        }
    }
})
