<template>
  <div>
    <div v-if="chatStore.isLoadingConversations" class="py-8">
      <Spinner size="md" :text="$t('globals.terms.loading')" />
    </div>

    <div v-else-if="!chatStore.hasConversations" class="flex flex-col items-center justify-center py-12 px-4">
      <MessageCircleDashed class="w-10 h-10 text-muted-foreground mb-4" />
      <h4 class="text-sm text-muted-foreground mb-2">{{ $t('globals.messages.noMessagesYet') }}</h4>
    </div>

    <div v-else>
      <div
        v-for="conversation in chatStore.getConversations"
        :key="conversation.uuid"
        class="p-4 border-b border-border hover:bg-accent/50 cursor-pointer transition-colors"
        @click="openConversation(conversation.uuid)"
      >
        <div class="flex items-center gap-3">
          <Avatar class="size-10 flex-shrink-0">
            <AvatarImage :src="getAvatarUrl(conversation)" />
            <AvatarFallback>{{ getAvatarFallback(conversation) }}</AvatarFallback>
          </Avatar>
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between mb-0.5">
              <span class="text-sm font-medium text-foreground">{{ getSenderLabel(conversation.last_message.author) }}</span>
              <span class="text-xs text-muted-foreground flex-shrink-0 ml-2 rtl:ml-0 rtl:mr-2">{{ getRelativeTime(new Date(conversation.last_message.created_at)) }}</span>
            </div>
            <div class="flex items-center gap-2">
              <p class="text-sm text-muted-foreground truncate flex-1 min-w-0">
                {{ conversation.last_message.content }}
              </p>
              <UnreadCountBadge :count="conversation.unread_message_count" class="flex-shrink-0" />
            </div>
          </div>
          <ChevronRight v-if="!widgetStore.isRtl" class="w-4 h-4 text-muted-foreground flex-shrink-0" />
          <ChevronLeft v-else class="w-4 h-4 text-muted-foreground flex-shrink-0" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { MessageCircleDashed, ChevronRight, ChevronLeft } from 'lucide-vue-next'
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
