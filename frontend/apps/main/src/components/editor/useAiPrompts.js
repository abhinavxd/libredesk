import { computed, ref } from 'vue'
import { useAiPromptStore } from '@main/stores/aiPrompt'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

// Rewrites the editor's HTML in place with a DB-stored prompt; disabled editors expose no prompts.
export function useAiPrompts({ enabled, htmlContent }) {
  const store = useAiPromptStore()
  const emitter = useEmitter()
  const isGenerating = ref(false)

  if (enabled) store.fetchPrompts()

  const aiPrompts = computed(() => (enabled ? store.prompts : []))

  const runAiPrompt = async (key) => {
    if (isGenerating.value) return
    isGenerating.value = true
    try {
      const resp = await api.aiCompletion({ prompt_key: key, content: htmlContent.value })
      htmlContent.value = resp.data.data || ''
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isGenerating.value = false
    }
  }

  return { aiPrompts, isGenerating, runAiPrompt }
}
