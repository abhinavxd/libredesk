import { defineStore } from 'pinia'

export const useAppSettingsStore = defineStore('settings', {
    state: () => ({
        settings: {}
    }),
    actions: {
        setSettings (newSettings) {
            this.settings = newSettings
        }
    }
})
