import { computed } from 'vue'

import { useI18n } from 'vue-i18n'
import { Download, Link2, MailOpen, MessageSquare, PenLine, StickyNote } from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'

import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS, CONVERSATION_ACTIONS } from '@main/constants/emitterEvents'
import { permissions as perms } from '@main/constants/permissions'

import { SECTIONS } from '../sections'

export function useConversationCommands() {
  const { t } = useI18n()
  const emitter = useEmitter()
  const conversationStore = useConversationStore()

  const section = SECTIONS.CONVERSATION

  const composeCommands = () => [
    {
      id: 'conv.reply',
      label: t('command.switchToReply'),
      keywords: [t('globals.terms.reply')],
      section,
      icon: MessageSquare,
      permission: perms.MESSAGES_WRITE,

      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, 'reply')
    },
    {
      id: 'conv.private-note',
      label: t('command.switchToPrivateNote'),
      keywords: [t('globals.terms.privateNote'), t('globals.terms.note')],
      section,
      icon: StickyNote,
      permission: perms.MESSAGES_WRITE_PRIVATE,

      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, 'private_note')
    },
    {
      id: 'conv.focus-reply',
      label: t('command.focusReplyBox'),
      keywords: [t('globals.terms.reply'), 'editor', 'compose'],
      section,
      icon: PenLine,
      permission: [perms.MESSAGES_WRITE, perms.MESSAGES_WRITE_PRIVATE],
      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_FOCUS)
    }
  ]

  const copyLink = async () => {
    await navigator.clipboard.writeText(window.location.href)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.copied') })
  }

  const miscCommands = (conv) => [
    {
      id: 'conv.mark-unread',
      label: t('globals.messages.markAsUnread'),
      keywords: ['unread'],
      section,
      icon: MailOpen,
      run: () => conversationStore.markAsUnread(conv.uuid)
    },
    {
      id: 'conv.copy-link',
      label: t('globals.messages.copyLink'),
      keywords: [t('globals.terms.copy'), t('globals.terms.link'), 'url'],
      section,
      icon: Link2,
      run: copyLink
    },
    {
      id: 'conv.transcript',
      label: t('conversation.downloadTranscript'),
      keywords: ['export'],
      section,
      icon: Download,
      run: () =>
        emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.DOWNLOAD_TRANSCRIPT)
    }
  ]

  return computed(() => {
    if (!conversationStore.isConversationOpen) return []
    const conv = conversationStore.current
    if (!conv) return []
    return [...miscCommands(conv), ...composeCommands()]
  })
}
