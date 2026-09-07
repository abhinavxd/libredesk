import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import api, { saveSession } from '@widget/api/index.js'

// initConversation calls the chat-init endpoint and applies the response to the stores - shared
// by every place that can start a conversation (typing the first message, submitting the
// pre-chat form, or a guided-form inbox starting proactively with no message at all).
export async function initConversation (payload) {
  const chatStore = useChatStore()
  const userStore = useUserStore()

  const resp = await api.initChatConversation(payload)
  const {
    conversation,
    session_token,
    user,
    messages,
    business_hours_id,
    working_hours_utc_offset,
    guided_form_allow_skip
  } = resp.data.data
  conversation.business_hours_id = business_hours_id
  conversation.working_hours_utc_offset = working_hours_utc_offset
  conversation.guided_form_allow_skip = guided_form_allow_skip

  if (!userStore.userSessionToken && session_token) {
    saveSession(session_token, user, userStore, true)
  }

  chatStore.addConversationToList(conversation)
  chatStore.setCurrentConversation(conversation)
  chatStore.replaceMessages(messages)

  return conversation
}
