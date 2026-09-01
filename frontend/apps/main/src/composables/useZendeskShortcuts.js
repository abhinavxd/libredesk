import { onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { useTicketActions } from '@main/composables/useTicketActions'
import { useZendeskTabs, TICKET_ROUTE_NAMES } from '@main/composables/useZendeskTabs'
import { useUiLayout, UI_LAYOUT_ZENDESK } from '@main/composables/useUiLayout'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { permissions as perms } from '@main/constants/permissions'

const CONVERSATION_ROUTES = new Set([
  'inbox-conversation',
  'team-inbox-conversation',
  'view-inbox-conversation'
])
const LIST_ROUTES = new Set(['inbox', 'team-inbox', 'view-inbox'])

function isTypingTarget (el) {
  if (!el) return false
  const tag = el.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable
}

function isOverlayOpen () {
  return Boolean(
    document.querySelector('[role="dialog"][data-state="open"], [role="alertdialog"][data-state="open"]')
  )
}

/**
 * Zendesk-style list/ticket keyboard shortcuts:
 *  j / k     next / previous ticket
 *  Enter     open the first (or highlighted) ticket from a list
 *  x         select / deselect the current ticket
 *  =         assign to me
 *  r         reply    n  private note    m  macros
 *  c         new ticket
 *  /         search    ?  shortcut help
 *  #         close
 *  s then o/p/s/c   submit as Open / Pending / Resolved / Closed
 *  [ / ]     previous / next ticket tab (Zendesk layout)
 * Ctrl/Cmd+Enter to submit a reply is handled by the editor itself.
 */
export function useZendeskShortcuts () {
  const route = useRoute()
  const router = useRouter()
  const conversationStore = useConversationStore()
  const userStore = useUserStore()
  const emitter = useEmitter()
  const actions = useTicketActions()
  const { tabs, activeUuid, selectTab, openNewTab } = useZendeskTabs()
  const { layout } = useUiLayout()

  let chord = null
  let chordTimer = null

  const startChord = (key) => {
    clearTimeout(chordTimer)
    chord = key
    chordTimer = setTimeout(() => {
      chord = null
    }, 800)
  }

  const consumeChord = () => {
    const next = chord
    chord = null
    clearTimeout(chordTimer)
    return next
  }

  const conversationRouteName = () => {
    if (route.params.teamID) return 'team-inbox-conversation'
    if (route.params.viewID) return 'view-inbox-conversation'
    return 'inbox-conversation'
  }

  const paramsFor = (uuid) => ({
    uuid,
    ...(route.params.teamID && { teamID: route.params.teamID }),
    ...(route.params.viewID && { viewID: route.params.viewID }),
    ...(!route.params.teamID && !route.params.viewID && { type: route.params.type || 'assigned' })
  })

  const openIndex = (idx) => {
    const list = conversationStore.conversationsList
    if (idx < 0 || idx >= list.length) return
    router.push({ name: conversationRouteName(), params: paramsFor(list[idx].uuid) })
  }

  const currentIndex = () =>
    conversationStore.conversationsList.findIndex((c) => c.uuid === route.params.uuid)

  const activeUUID = () => route.params.uuid || conversationStore.current?.uuid || ''

  const matchStatus = (letter) => {
    const statuses = conversationStore.statusOptionsNoSnooze.map((s) => s.label)
    const aliases = {
      o: ['open'],
      p: ['pending', 'snoozed'],
      s: ['solved', 'resolved'],
      c: ['closed'],
      h: ['hold', 'on-hold', 'on hold', 'on_hold']
    }
    const wanted = aliases[letter] || []
    return (
      statuses.find((s) => wanted.includes(s.toLowerCase())) ||
      statuses.find((s) => s.toLowerCase().startsWith(letter))
    )
  }

  const applyStatus = (letter) => {
    if (!userStore.can(perms.CONVERSATIONS_UPDATE_STATUS)) return
    const uuid = activeUUID()
    const status = matchStatus(letter)
    if (uuid && status) actions.setStatus(uuid, status)
  }

  const shiftTab = (dir) => {
    if (layout.value !== UI_LAYOUT_ZENDESK || !tabs.value.length) return
    const idx = tabs.value.findIndex((t) => t.uuid === activeUuid.value)
    const next = idx < 0 ? 0 : (idx + dir + tabs.value.length) % tabs.value.length
    selectTab(tabs.value[next])
  }

  const onKeydown = (e) => {
    if (isTypingTarget(e.target) || isOverlayOpen()) {
      consumeChord()
      return
    }
    if (e.metaKey || e.ctrlKey || e.altKey) return

    const inConversation = CONVERSATION_ROUTES.has(route.name)
    const inList = LIST_ROUTES.has(route.name)
    const inTicketChrome = TICKET_ROUTE_NAMES.has(route.name)
    if (!inConversation && !inList && !inTicketChrome) return

    if (chord === 's') {
      e.preventDefault()
      const letter = e.key.toLowerCase()
      consumeChord()
      if (letter !== 'escape') applyStatus(letter)
      return
    }

    if (e.key === 'j' || e.key === 'k') {
      e.preventDefault()
      const idx = currentIndex()
      if (idx === -1) {
        if (e.key === 'j') openIndex(0)
        return
      }
      openIndex(e.key === 'j' ? idx + 1 : idx - 1)
      return
    }

    if (e.key === 'Enter' && inList) {
      e.preventDefault()
      openIndex(0)
      return
    }

    if (e.key === 'x') {
      const uuid = activeUUID() || conversationStore.conversationsList[0]?.uuid
      if (!uuid) return
      e.preventDefault()
      actions.toggleSelect(uuid)
      return
    }

    if (e.key === '=' && userStore.can(perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE)) {
      const uuid = activeUUID()
      if (!uuid) return
      e.preventDefault()
      actions.assignToMe(uuid)
      return
    }

    if (e.key === 'r' && inConversation) {
      e.preventDefault()
      actions.focusComposer(activeUUID(), 'reply')
      return
    }

    if (e.key === 'n' && inConversation) {
      e.preventDefault()
      actions.focusComposer(activeUUID(), 'private_note')
      return
    }

    if (e.key === 'm' && inConversation) {
      e.preventDefault()
      emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, {
        command: 'apply-macro-to-existing-conversation',
        open: true
      })
      return
    }

    if (e.key === 'c') {
      e.preventDefault()
      if (layout.value === UI_LAYOUT_ZENDESK) {
        openNewTab()
      } else {
        emitter.emit(EMITTER_EVENTS.OPEN_CREATE_CONVERSATION)
      }
      return
    }

    if (e.key === '/') {
      e.preventDefault()
      emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, { command: null, open: true })
      return
    }

    if (e.key === '?') {
      e.preventDefault()
      emitter.emit(EMITTER_EVENTS.SHOW_KEYBOARD_SHORTCUTS)
      return
    }

    if (e.key === '#') {
      e.preventDefault()
      applyStatus('c')
      return
    }

    if (e.key === 's' && (inConversation || activeUUID())) {
      e.preventDefault()
      startChord('s')
      return
    }

    if (e.key === '[' && inTicketChrome) {
      e.preventDefault()
      shiftTab(-1)
      return
    }

    if (e.key === ']' && inTicketChrome) {
      e.preventDefault()
      shiftTab(1)
    }
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => {
    window.removeEventListener('keydown', onKeydown)
    clearTimeout(chordTimer)
  })
}
