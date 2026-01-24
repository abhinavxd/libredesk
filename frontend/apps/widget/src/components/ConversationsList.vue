<template>
  <div>
    <div v-if="chatStore.isLoadingConversations" class="py-8">
      <Spinner size="md" :text="$t('globals.terms.loading')" />
    </div>

    <div v-else-if="!chatStore.hasConversations" class="flex flex-col items-center justify-center py-12 px-4">
      <MessageCircleDashed class="w-10 h-10 text-muted-foreground mb-4" />
      <h4 class="text-sm text-muted-foreground mb-2">No messages yet</h4>
    </div>

    <div v-else class="divide-y divide-border">
      <div
        v-for="conversation in chatStore.getConversations"
        :key="conversation.uuid"
        class="p-4 hover:bg-accent/50 cursor-pointer transition-colors"
        @click="openConversation(conversation.uuid)"
      >
        <div class="flex items-center justify-between gap-3">
          <Avatar class="size-8 flex-shrink-0">
            <AvatarImage :src="getAvatarUrl(conversation)" />
            <AvatarFallback>{{ getAvatarFallback(conversation) }}</AvatarFallback>
          </Avatar>
          <div class="flex-1 min-w-0">
            <h4 class="font-medium text-foreground text-sm truncate mb-1">
              <span class="text-muted-foreground">{{ getSenderLabel(conversation.last_message.author) }}:</span>
              {{ conversation.last_message.content }}
            </h4>
            <div v-if="conversation.last_message" class="text-xs text-muted-foreground">
              {{ getRelativeTime(new Date(conversation.last_message.created_at)) }}
            </div>
          </div>

          <div class="flex items-center justify-center flex-shrink-0 gap-2">
            <UnreadCountBadge :count="conversation.unread_message_count" />
            <ArrowRight class="w-4 h-4" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { MessageCircleDashed, ArrowRight } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useChatStore } from '@widget/store/chat.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Spinner } from '@shared-ui/components/ui/spinner'
import UnreadCountBadge from './UnreadCountBadge.vue'

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const { t } = useI18n()

function isUserMessage(author) {
  return author?.type === 'contact' || author?.type === 'visitor'
}

function getInitial(name) {
  return name?.charAt(0)?.toUpperCase() || '?'
}

function getSenderLabel(author) {
  if (!author) return ''
  if (isUserMessage(author)) {
    return t('globals.terms.you')
  }
  return author.first_name || ''
}

function getAvatarUrl(conversation) {
  const author = conversation.last_message?.author
  if (isUserMessage(author)) {
    const assignee = conversation.assignee
    if (assignee?.id > 0) {
      return assignee.avatar_url || ''
    }
    return widgetStore.config.launcher?.logo_url || ''
  }
  return author?.avatar_url || ''
}

function getAvatarFallback(conversation) {
  const author = conversation.last_message?.author
  if (isUserMessage(author)) {
    const assignee = conversation.assignee
    if (assignee?.id > 0) {
      return getInitial(assignee.first_name)
    }
    return getInitial(widgetStore.config.brand_name)
  }
  return getInitial(author?.first_name)
}

async function openConversation(conversationUUID) {
  widgetStore.navigateToChat()
  const success = await chatStore.loadConversation(conversationUUID)
  if (!success) {
    widgetStore.navigateToMessages()
  }
}

onMounted(() => {
  chatStore.fetchConversations()
})
</script>
