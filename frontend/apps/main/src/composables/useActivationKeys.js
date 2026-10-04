import { ref } from 'vue'
import api from '@main/api'

// Shared by every composer: whether Key Distribution is on, the games the key
// card offers, and the game this person last picked.
const info = ref({ enabled: false, max_keys_per_email: 1, apps: [], last_app_id: null })

export function useActivationKeys() {
  // Refetched whenever a composer opens, so a newly enabled feature or a
  // restocked game shows up without reloading the page.
  const refresh = async () => {
    try {
      const response = await api.getKeyComposer()
      info.value = response.data.data
    } catch {
      // The card stays as it was; the server still checks every send.
    }
  }

  const rememberApp = (appId) => {
    info.value = { ...info.value, last_app_id: appId }
    api.setKeyComposerApp(appId).catch(() => {})
  }

  return { info, refresh, rememberApp }
}
