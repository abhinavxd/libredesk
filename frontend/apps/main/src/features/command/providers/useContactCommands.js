import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
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
  const { t } = useI18n()
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
        label: t('command.startConversationWithContact'),
        keywords: [t('conversation.newConversation'), t('globals.messages.create')],
        section,
        icon: MessageSquarePlus,
        permission: perms.CONVERSATIONS_WRITE,
        run: () => emitter.emit(EMITTER_EVENTS.OPEN_CREATE_CONVERSATION, { contact })
      },
      {
        id: 'contact.add-note',
        label: t('contact.addNote'),
        keywords: [t('globals.terms.note')],
        section,
        icon: StickyNote,
        permission: perms.CONTACT_NOTES_WRITE,
        run: contactAction(CONTACT_ACTIONS.ADD_NOTE)
      },
      {
        id: 'contact.toggle-block',
        label: t(contact.enabled ? 'contact.blockContact' : 'contact.unblockContact'),
        keywords: [t('globals.messages.block'), t('globals.messages.unblock')],
        section,
        icon: contact.enabled ? ShieldOff : ShieldCheck,
        permission: perms.CONTACTS_BLOCK,
        run: contactAction(CONTACT_ACTIONS.TOGGLE_BLOCK)
      },
      {
        id: 'contact.export',
        label: t('globals.messages.exportData'),
        keywords: ['download', 'json'],
        section,
        icon: Download,
        permission: perms.CONTACTS_EXPORT,
        run: contactAction(CONTACT_ACTIONS.EXPORT)
      },
      {
        id: 'contact.delete',
        label: t('contact.deleteContact'),
        keywords: [t('globals.messages.delete')],
        section,
        icon: Trash2,
        permission: perms.CONTACTS_DELETE,
        destructive: true,
        run: contactAction(CONTACT_ACTIONS.DELETE)
      }
    ]
  })
}
