import { onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { useKeyboardShortcutsDialog } from '@main/composables/useKeyboardShortcutsDialog'
import { useBulkActionPermissions } from '@main/composables/useBulkActionPermissions'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { permissions as perms } from '@main/constants/permissions'
import { CONVERSATION_DEFAULT_STATUSES, MACRO_CONTEXT } from '@main/constants/conversation'
import { MACROS_COMMAND, useCommandPalette } from './useCommandPalette'
import { SNOOZE_COMMAND } from './providers/useConversationCommands'

const isMod = (event) => event.ctrlKey || event.metaKey
const onlyAlt = (event) => event.altKey && !event.ctrlKey && !event.metaKey && !event.shiftKey
const dialogIsOpen = () => Boolean(document.querySelector('[role="dialog"][data-state="open"]'))

// Alt combos work while typing in the editor, so no shortcut depends on where focus is.
export function useGlobalShortcuts() {
  const route = useRoute()
  const router = useRouter()
  const emitter = useEmitter()
  const conversationStore = useConversationStore()
  const userStore = useUserStore()
  const palette = useCommandPalette()
  const shortcutsDialog = useKeyboardShortcutsDialog()
  const { canBulkAct } = useBulkActionPermissions()

  const conversationOpen = () => conversationStore.isConversationOpen && conversationStore.current

  const openMacros = () => {
    const inNewConversation = palette.macroContext.value === MACRO_CONTEXT.NEW_CONVERSATION
    if (!inNewConversation && !conversationOpen()) return false
    palette.openPalette({ parent: MACROS_COMMAND })
    return true
  }

  const openGroup = (parent, permission) => () => {
    if (!conversationOpen() || !userStore.can(permission)) return false
    palette.openPalette({ parent })
    return true
  }

  const stepConversation = (step) => () => {
    if (!conversationOpen() || !String(route.name).endsWith('-conversation')) return false
    const list = conversationStore.conversationsList
    const index = list.findIndex((c) => c.uuid === route.params.uuid)
    if (index === -1) return false
    const next = list[index + step]
    if (!next) return false
    router.push({ name: route.name, params: { ...route.params, uuid: next.uuid } })
    return true
  }

  const setStatus = (status) => () => {
    if (!conversationOpen() || !userStore.can(perms.CONVERSATIONS_UPDATE_STATUS)) return false
    if (conversationStore.current.status !== status) conversationStore.updateStatus(status)
    return true
  }

  const compose = (type, permission) => () => {
    if (!conversationOpen() || !userStore.can(permission)) return false
    emitter.emit(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, type)
    emitter.emit(EMITTER_EVENTS.REPLY_BOX_FOCUS)
    return true
  }

  const toggleSelectCurrent = () => {
    if (!conversationOpen() || !canBulkAct.value) return false
    conversationStore.toggleSelect(conversationStore.current.uuid, false)
    return true
  }

  const newConversation = () => {
    if (!userStore.can(perms.CONVERSATIONS_WRITE)) return false
    emitter.emit(EMITTER_EVENTS.OPEN_CREATE_CONVERSATION, {})
    return true
  }

  const showShortcuts = () => {
    palette.closePalette()
    shortcutsDialog.show()
    return true
  }

  // Keyed by event.code: on macOS Option+letter changes event.key to a symbol.
  const altKeys = {
    KeyJ: stepConversation(-1),
    KeyK: stepConversation(1),
    KeyZ: openGroup(SNOOZE_COMMAND, perms.CONVERSATIONS_UPDATE_STATUS),
    KeyP: openGroup('conv.priority', perms.CONVERSATIONS_UPDATE_PRIORITY),
    KeyA: openGroup('conv.assign-agent', perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE),
    KeyR: compose('reply', perms.MESSAGES_WRITE),
    KeyN: compose('private_note', perms.MESSAGES_WRITE_PRIVATE),
    KeyX: toggleSelectCurrent,
    KeyC: newConversation,
    KeyE: setStatus(CONVERSATION_DEFAULT_STATUSES.RESOLVED),
    KeyO: setStatus(CONVERSATION_DEFAULT_STATUSES.OPEN)
  }

  const onKeydown = (event) => {
    if (event.isComposing || event.repeat) return
    let handled = false
    if (isMod(event) && !event.altKey) {
      const key = event.key.toLowerCase()
      // Layouts like German put "/" on a shifted key, so it cannot require an unshifted event.
      if (event.key === '/') handled = showShortcuts()
      else if (event.shiftKey) handled = false
      else if (key === 'k') handled = (palette.togglePalette(), true)
      else if (key === 'm') handled = openMacros()
    } else if (onlyAlt(event) && altKeys[event.code] && !dialogIsOpen()) {
      handled = altKeys[event.code]()
    }
    if (handled) event.preventDefault()
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}
