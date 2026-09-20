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
  watch(
    () => [
      chat.getConversations,
      user.userID,
      widget.config.previews,
      widget.config.dark_mode,
      widget.isOpen
    ],
    async () => {
      await nextTick()
      const element = document.querySelector('.libredesk-widget-app')
      if (!element) return
      const style = getComputedStyle(element)
      const previews = chat.getConversations
        .filter((conversation) => conversation.unread_message_count > 0)
        .flatMap((conversation) =>
          (conversation.unread_messages?.length
            ? conversation.unread_messages
            : [conversation.last_message]
          ).map((message) => ({ conversation, message }))
        )
        .filter(({ message }) => ['agent', 'ai_assistant'].includes(message?.author?.type))
        .sort((a, b) => new Date(b.message.created_at) - new Date(a.message.created_at))
        .slice(0, 3)
        .map(({ conversation, message }) => {
          const author = message.author || {}
          return {
            conversation: conversation.uuid,
            key: message.uuid || `${conversation.uuid}:${message.created_at}`,
            name:
              [author.first_name, author.last_name].filter(Boolean).join(' ') ||
              widget.config.brand_name,
            avatar: author.avatar_url || '',
            text:
              widget.config.previews?.content === 'generic'
                ? t('widget.newReply')
                : getTextFromHTML(message.text_content || message.content || '').slice(0, 240) ||
                  t('globals.terms.attachment'),
            image:
              widget.config.previews?.content === 'generic'
                ? ''
                : message.attachments?.find((attachment) =>
                    attachment.content_type?.startsWith('image/')
                  )?.thumbnail_url || ''
          }
        })
      const target = parentOrigin()
      if (!target || target === 'null') return
      window.parent.postMessage(
        {
          type: 'REPLY_PREVIEWS',
          identity: String(user.userID || 'visitor'),
          previews,
          config: { ...widget.config.previews },
          labels: {
            dismiss: t('globals.terms.dismiss'),
            dismissAll: t('widget.dismissPreviews'),
            open: t('globals.messages.openConversation')
          },
          theme: {
            background: style.backgroundColor,
            foreground: style.color,
            muted: `hsl(${style.getPropertyValue('--muted-foreground')})`,
            border: `hsl(${style.getPropertyValue('--border')})`
          }
        },
        target
      )
    },
    { deep: true, immediate: true }
  )
}
