import { watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import { getTextFromHTML } from '@shared-ui/utils/string.js'

const parentOrigin = () => new URLSearchParams(window.location.search).get('parent_origin')

export function useReplyPreviews() {
  const widget = useWidgetStore()
  const chat = useChatStore()
  const user = useUserStore()
  const { t } = useI18n()
  watch(() => [chat.getConversations, user.userID, widget.config.previews, widget.config.dark_mode, widget.isOpen], async () => {
    await nextTick()
    const element = document.querySelector('.libredesk-widget-app')
    if (!element) return
    const style = getComputedStyle(element)
    const previews = chat.getConversations.filter(conversation => conversation.unread_message_count > 0 && ['agent', 'ai_assistant'].includes(conversation.last_message?.author?.type)).map(conversation => {
      const message = conversation.last_message
      const author = message.author || {}
      return {
        conversation: conversation.uuid,
        key: `${conversation.uuid}:${message.created_at}`,
        name: [author.first_name, author.last_name].filter(Boolean).join(' ') || widget.config.brand_name,
        avatar: author.avatar_url || '',
        text: widget.config.previews?.content === 'generic' ? t('widget.newReply') : getTextFromHTML(message.content || '').slice(0, 240) || t('globals.terms.attachment'),
        image: widget.config.previews?.content === 'generic' ? '' : message.attachments?.find(attachment => attachment.content_type?.startsWith('image/'))?.thumbnail_url || '',
      }
    })
    const target = parentOrigin()
    if (!target) return
    window.parent.postMessage({ type: 'REPLY_PREVIEWS', identity: String(user.userID || 'visitor'), previews, config: { ...widget.config.previews },
      labels: { dismiss: t('globals.terms.dismiss'), dismissAll: t('widget.dismissPreviews'), open: t('widget.openConversation') },
      theme: { background: style.backgroundColor, foreground: style.color, border: `hsl(${style.getPropertyValue('--border')})` },
    }, target)
  }, { deep: true, immediate: true })
}
