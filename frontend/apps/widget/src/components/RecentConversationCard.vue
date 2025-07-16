<template>
  <div class="space-y-3">
    <Card
      @click="continueConversation"
      class="hover:bg-accent transition-colors cursor-pointer rounded-md"
    >
      <CardContent class="p-4">
        <div class="flex items-start justify-between">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-2">
              <div class="text-sm font-medium">Continue your conversation</div>
            </div>
            <div class="flex gap-2 items-start">
              <div class="text-sm text-foreground line-clamp-2 flex-1 min-w-0">
                {{ conversation.last_message.content }}
              </div>
              <UnreadCountBadge :count="conversation.unread_message_count" class="flex-shrink-0" />
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
import { useUserStore } from '@widget/store/user.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  conversation: {
    type: Object,
    required: true
  }
})

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const userStore = useUserStore()
const { t } = useI18n()

const authorDisplayName = computed(() => {
  const author = props.conversation.last_message.author
  if (!author) return t('globals.terms.someone')
  // Check if the author is the current user
  if (author.id === userStore.userID) {
    return t('globals.terms.you')
  }
  return author.first_name || t('globals.terms.someone')
})

const continueConversation = async () => {
  // Load the conversation details and messages
  await chatStore.loadConversation(props.conversation.uuid)
  // Navigate to chat view
  widgetStore.navigateToChat()
}
</script>
