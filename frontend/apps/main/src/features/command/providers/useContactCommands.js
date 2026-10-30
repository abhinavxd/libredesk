


import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
import {
  Download,
  MessageSquarePlus,
  ShieldCheck,
  ShieldOff,
  StickyNote,
  Trash2
} from 'lucide-vue-next'
import { useContactStore } from '@main/stores/contact'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS, CONTACT_ACTIONS } from '@main/constants/emitterEvents'
import { permissions as perms } from '@main/constants/permissions'
import { SECTIONS } from '../sections'

export function useContactCommands() {
  const route = useRoute()
  const i18n = useI18n()
  const emitter = useEmitter()
  const contactStore = useContactStore()

  const section = SECTIONS.CONTACT
  const contactAction = (action) => () => emitter.emit(EMITTER_EVENTS.CONTACT_ACTION, action)

  return computed(() => {
    const contact = contactStore.current
    if (route.name !== 'contact-detail' || !contact) return []
    return [
      {
        id: 'contact.start-conversation',
        label: i18n._('Start conversation with contact'),
        keywords: [i18n._('New conversation'), i18n._('Create')],
        section,
        icon: MessageSquarePlus,
        permission: perms.CONVERSATIONS_WRITE,
        run: () => emitter.emit(EMITTER_EVENTS.OPEN_CREATE_CONVERSATION, { contact })
      },
      {
        id: 'contact.add-note',
        label: i18n._('Add note'),
        keywords: [i18n._('{count, plural, one {Note} other {Notes}}')],
        section,
        icon: StickyNote,
        permission: perms.CONTACT_NOTES_WRITE,
        run: contactAction(CONTACT_ACTIONS.ADD_NOTE)
      },
      {
        id: 'contact.toggle-block',
        label: contact.enabled ? i18n._('Block contact') : i18n._('Unblock contact'),
        keywords: [i18n._('Block'), i18n._('Unblock')],
        section,
        icon: contact.enabled ? ShieldOff : ShieldCheck,
        permission: perms.CONTACTS_BLOCK,
        run: contactAction(CONTACT_ACTIONS.TOGGLE_BLOCK)
      },
      {
        id: 'contact.export',
        label: i18n._('Export data'),
        keywords: ['download', 'json'],
        section,
        icon: Download,
        permission: perms.CONTACTS_EXPORT,
        run: contactAction(CONTACT_ACTIONS.EXPORT)
      },
      {
        id: 'contact.delete',
        label: i18n._('Delete contact'),
        keywords: [i18n._('Delete')],
        section,
        icon: Trash2,
        permission: perms.CONTACTS_DELETE,
        destructive: true,
        run: contactAction(CONTACT_ACTIONS.DELETE)
      }
    ]
  })
}
