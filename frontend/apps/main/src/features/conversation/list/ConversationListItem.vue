<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <router-link
        :to="conversationRoute"
        class="group relative block px-3 py-2.5 transition-colors duration-150 ease-in-out cursor-pointer"
        :class="{
          'bg-accent': isCurrent,
          'hover:bg-accent/40': !isCurrent
        }"
      >
        <span
          v-if="isUnread"
          class="pointer-events-none absolute inset-y-0 left-0 w-1 bg-success"
          data-unread-indicator
          aria-hidden="true"
        />
        <div class="flex items-start gap-2">
          <!-- Content container -->
          <div class="flex-1 min-w-0 space-y-1.5">
            <!-- Name + Subject group -->
            <div class="space-y-0.5">
              <!-- Contact name + channel + time -->
              <div class="flex items-baseline justify-between gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <h3
                      class="text-sm truncate min-w-0 text-foreground"
                      :class="isUnread ? 'font-semibold' : 'font-medium'"
                    >
                      {{ contactFullName }}
                    </h3>
                  </TooltipTrigger>
                  <TooltipContent>{{ contactFullName }}</TooltipContent>
                </Tooltip>
                <div class="flex items-center gap-1 flex-shrink-0">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <component
                        :is="Mail"
                        class="w-3 h-3 text-muted-foreground"
                        role="img"
                        :aria-label="conversation.inbox_name"
                      />
                    </TooltipTrigger>
                    <TooltipContent>{{ conversation.inbox_name }}</TooltipContent>
                  </Tooltip>
                  <span
                    class="text-xs text-muted-foreground whitespace-nowrap tabular-nums"
                    v-if="conversation.last_message_at"
                  >
                    {{ relativeLastMessageTime }}
                  </span>
                </div>
              </div>

              <!-- Subject -->
              <p
                v-if="showSubject && conversation.subject"
                class="text-xs text-muted-foreground truncate"
              >
                {{ conversation.subject }}
              </p>
            </div>

            <!-- Message preview + unread count -->
            <div class="flex items-center justify-between gap-2">
              <p
                class="text-sm flex-1 min-w-0 truncate"
                :class="isUnread ? 'text-foreground font-medium' : 'text-muted-foreground'"
              >
                <template v-if="isTyping">
                  <span class="italic text-foreground">{{ conversationStore.typingNames(conversation.uuid) ? $t('conversation.agentsTyping', { names: conversationStore.typingNames(conversation.uuid) }) : $t('globals.terms.typing') }}</span>
                </template>
                <template v-else-if="hasDraftForConversation && !isCurrent">
                  <span class="font-medium text-foreground">{{ $t('globals.terms.draft') }}:</span>
                  {{ draftPreview }}
                </template>
                <template v-else>
                  <Reply
                    class="text-success inline-block align-text-bottom mr-0.5"
                    :size="14"
                    v-if="conversation.last_message_sender === 'agent'"
                  />{{ trimmedLastMessage }}
                </template>
              </p>
              <div
                v-if="isUnread"
                class="flex items-center justify-center w-5 h-5 bg-primary text-primary-foreground text-xs font-medium rounded-full flex-shrink-0"
              >
                {{
                  conversation.unread_message_count > 9 ? '9+' : conversation.unread_message_count
                }}
              </div>
            </div>

            <!-- SLA Badges -->
            <div v-if="hasSlaDeadlines" class="flex items-center gap-1"></div>
          </div>
        </div>
      </router-link>
    </ContextMenuTrigger>
    <ContextMenuContent>
      <ContextMenuItem @click="handleMarkAsUnread">
        <MailOpen class="w-4 h-4 mr-2" />
        {{ $t('globals.messages.markAsUnread') }}
      </ContextMenuItem>
    </ContextMenuContent>
  </ContextMenu>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { Mail, Reply, MailOpen } from 'lucide-vue-next'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger
} from '@shared-ui/components/ui/context-menu'

import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useConversationStore } from '@main/stores/conversation'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useI18n } from 'vue-i18n'

let timer = null
const now = ref(new Date())
const route = useRoute()
const conversationStore = useConversationStore()
const appSettingsStore = useAppSettingsStore()
const { t } = useI18n()

const props = defineProps({
  conversation: Object,
  currentConversation: Object,
  contactFullName: String
})

const handleMarkAsUnread = () => {
  conversationStore.markAsUnread(props.conversation.uuid)
}

const conversationRoute = computed(() => {
  return {
    name: 'address-inbox-conversation',
    params: {
      uuid: props.conversation.uuid,
      addressID: route.params.addressID || props.conversation.address_id
    },
    query: props.conversation.mentioned_message_uuid
      ? { scrollTo: props.conversation.mentioned_message_uuid }
      : {}
  }
})

onMounted(() => {
  timer = setInterval(() => {
    now.value = new Date()
  }, 60000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const trimmedLastMessage = computed(() => {
  const message = props.conversation.last_message || ''
  return message.length > 120 ? message.slice(0, 120) + '...' : message
})

const relativeLastMessageTime = computed(() => {
  return props.conversation.last_message_at
    ? getRelativeTime(props.conversation.last_message_at, now.value)
    : ''
})

const hasSlaDeadlines = computed(() => {
  const c = props.conversation
  return c.first_response_deadline_at || c.resolution_deadline_at || c.next_response_deadline_at
})

const hasDraftForConversation = computed(() => {
  return conversationStore.conversationHasDraft(props.conversation.uuid)
})

const isTyping = computed(() => conversationStore.typingByUUID[props.conversation.uuid] === true)

const draftPreview = computed(() => {
  const draft = conversationStore.conversationDraftPreview(props.conversation.uuid)
  if (!draft?.content && !draft?.meta?.attachments?.length) return ''
  const text = (draft.content || '').replace(/<[^>]*>/g, '').trim()
  if (text) return text.length > 120 ? text.slice(0, 120) + '...' : text
  if (draft.meta?.attachments?.length)
    return conversationStore.getMediaPreview(draft.meta.attachments)
  if (/<img\b/i.test(draft.content || '')) return t('globals.terms.image', 1)
  return ''
})

const showSubject = computed(
  () => appSettingsStore.settings['app.show_conversation_subject'] !== false
)

const isUnread = computed(() => props.conversation.unread_message_count > 0)

const isCurrent = computed(() => props.conversation.uuid === props.currentConversation?.uuid)
</script>
