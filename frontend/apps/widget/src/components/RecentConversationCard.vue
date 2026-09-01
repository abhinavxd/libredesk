<template>
  <button type="button" class="widget-starter !items-start text-left" @click="continueConversation">
    <div class="flex-1 min-w-0">
      <div class="text-sm font-semibold mb-1">{{ $t('globals.messages.continueConversation') }}</div>
      <div class="flex gap-2 items-start">
        <div class="text-sm text-muted-foreground line-clamp-2 flex-1 min-w-0 font-normal">
          {{ conversation.last_message.content }}
        </div>
        <UnreadCountBadge :count="conversation.unread_message_count" class="flex-shrink-0" />
      </div>
      <div class="text-xs text-muted-foreground mt-1.5 font-normal">
        <span>{{ authorDisplayName }}</span>
        <span class="mx-1">•</span>
        <span>{{ getRelativeTime(new Date(conversation.last_message.created_at)) }}</span>
      </div>
    </div>
    <ArrowRight class="widget-starter__icon self-center" />
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowRight } from 'lucide-vue-next'
import UnreadCountBadge from '@widget/components/UnreadCountBadge.vue'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { useChatStore } from '@widget/store/chat.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  conversation: {
    type: Object,
    required: true
  }
})

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const { t } = useI18n()

const authorDisplayName = computed(() => {
  const author = props.conversation.last_message.author
  if (!author) return t('globals.terms.someone')
  if (author.type === 'visitor' || author.type === 'contact') {
    return t('globals.terms.you')
  }
  return author.first_name || t('globals.terms.someone')
})

const continueConversation = async () => {
  widgetStore.navigateToChat()
  await chatStore.loadConversation(props.conversation.uuid)
}
</script>
