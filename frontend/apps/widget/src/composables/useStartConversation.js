import { computed } from 'vue'
import { useChatStore } from '@widget/store/chat.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { useUserStore } from '@widget/store/user.js'

export function useStartConversation () {
  const chatStore = useChatStore()
  const widgetStore = useWidgetStore()
  const userStore = useUserStore()

  const audienceConfig = computed(() =>
    userStore.isVisitor ? widgetStore.config?.visitors : widgetStore.config?.users
  )

  const configuredLabel = computed(() => audienceConfig.value?.start_conversation_button_text || '')

  // Mirrors the server check, else the button is shown and the send fails.
  const canStartNewConversation = computed(() => {
    if (!audienceConfig.value?.allow_start_conversation) return false
    return (
      audienceConfig.value?.prevent_multiple_conversations !== true || !chatStore.hasConversations
    )
  })

  const startNewConversation = () => {
    chatStore.setCurrentConversation(null)
    chatStore.clearMessages()
    widgetStore.navigateToChat()
  }

  return { configuredLabel, canStartNewConversation, startNewConversation }
}
