import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@widget/api/index.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'

const parentOrigin = () => new URLSearchParams(window.location.search).get('parent_origin')
const postToParent = message => {
  const target = parentOrigin()
  if (target) window.parent.postMessage(message, target)
}

export const useProactiveStore = defineStore('proactive', () => {
  const invitation = ref(null)
  const pending = ref(null)
  const widget = useWidgetStore()
  const chat = useChatStore()
  let browserKey = ''
  let sessionKey = ''
  const setSessionKey = key => { if (typeof key === 'string' && key.length === 36) sessionKey = key }
  const setBrowserKey = key => { if (typeof key === 'string' && key.length === 36) browserKey = key }
  let busy = false
  let generation = 0
  const reset = () => {
    generation++
    invitation.value = null
    pending.value = null
    busy = false
    postToParent({ type: 'CLEAR_CAMPAIGN' })
  }
  // The user opened the invitation and then left the chat view without answering it.
  const abandon = () => {
    pending.value = null
  }
  const next = async (context) => {
    if (!browserKey || busy || invitation.value || pending.value || !widget.config.has_campaigns || widget.isOpen || chat.getConversations.some(item => item.unread_message_count > 0)) return
    busy = true
    const request = generation
    try {
      const response = await api.nextCampaign({ ...context, browser_key: browserKey, session_key: sessionKey })
      if (request !== generation || !response.data.data || widget.isOpen) return
      const delivery = response.data.data
      invitation.value = delivery
      postToParent({ type: 'SHOW_CAMPAIGN', delivery })
    } catch { return false }
    finally { if (request === generation) busy = false }
  }
  const event = async (name, id) => {
    if (!invitation.value || invitation.value.id !== id) return
    const delivery = invitation.value
    if (name === 'opened') {
      pending.value = delivery
      invitation.value = null
      chat.setCurrentConversation(null)
      widget.navigateToChat()
    } else if (name === 'dismissed') invitation.value = null
    else if (name === 'dropped') {
      invitation.value = null
      return
    }
    try { await api.campaignEvent({ delivery_id: id, browser_key: browserKey, event: name }) } catch { return false }
  }
  const replyPayload = () => pending.value ? { delivery_id: pending.value.id, browser_key: browserKey } : {}
  const replied = () => { pending.value = null; invitation.value = null; postToParent({ type: 'CLEAR_CAMPAIGN' }) }
  return { setSessionKey, setBrowserKey, invitation, pending, next, event, reset, abandon, replyPayload, replied }
})
