<template>
  <div class="space-y-3">
    <Card
      class="rounded-xl border-border/80 shadow-sm transition-[background-color,box-shadow] can-hover:hover:bg-accent can-hover:hover:shadow-md"
    >
      <button
        type="button"
        class="block w-full rounded-xl text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        @click="continueConversation"
      >
        <CardContent class="p-4">
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <div class="text-sm font-medium">
                  {{ i18n._('Continue your conversation') }}
                </div>
              </div>
              <div class="flex gap-2 items-start">
                <div class="text-sm text-foreground line-clamp-2 flex-1 min-w-0">
                  {{ lastMessagePreview(conversation.last_message, i18n) }}
                </div>
                <UnreadCountBadge
                  :count="conversation.unread_message_count"
                  class="flex-shrink-0"
                />
              </div>
              <div class="text-xs text-muted-foreground mt-1">
                <span>{{ authorDisplayName }}</span>
                <span class="mx-1">•</span>
                <span>{{ getRelativeTime(new Date(conversation.last_message.created_at)) }}</span>
              </div>
            </div>
            <ArrowRight class="w-4 h-4 text-muted-foreground ml-2 flex-shrink-0 self-center" />
          </div>
        </CardContent>
      </button>
    </Card>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowRight } from 'lucide-vue-next'
import { Card, CardContent } from '@shared-ui/components/ui/card'
import UnreadCountBadge from '@widget/components/UnreadCountBadge.vue'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { useChatStore } from '@widget/store/chat.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { lastMessagePreview } from '@widget/utils/messagePreview.js'
import { useI18n } from '@shared-ui/i18n'

const props = defineProps({
  conversation: {
    type: Object,
    required: true
  }
})

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const i18n = useI18n()

const authorDisplayName = computed(() => {
  const author = props.conversation.last_message.author
  if (!author) return i18n._('Someone')
  if (author.type === 'visitor' || author.type === 'contact') {
    return i18n._('You')
  }
  return author.first_name || i18n._('Someone')
})

const continueConversation = async () => {
  widgetStore.navigateToChat()
  await chatStore.loadConversation(props.conversation.uuid)
}
</script>
