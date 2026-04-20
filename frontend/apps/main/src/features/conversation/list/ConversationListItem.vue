<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <router-link
        :to="conversationRoute"
        class="group relative block px-3 py-3 transition-all duration-200 ease-in-out cursor-pointer hover:bg-accent/20 dark:hover:bg-accent/60"
        :class="{
          'bg-accent/60': conversation.uuid === currentConversation?.uuid,
          'bg-primary/5': isItemSelected && conversation.uuid !== currentConversation?.uuid
        }"
      >
        <div class="flex items-start gap-2">
          <!-- Selection checkbox -->
          <div class="flex items-center pt-2" @click.prevent.stop="handleCheckboxClick">
            <Checkbox
              :checked="isItemSelected"
              :aria-label="t('conversation.bulkActions.selectConversation')"
            />
          </div>

          <!-- Avatar with channel indicator -->
          <div class="relative flex-shrink-0">
            <Avatar class="w-10 h-10 rounded-full">
              <AvatarImage
                :src="conversation.contact.avatar_url || ''"
                class="object-cover"
              />
              <AvatarFallback>
                {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
              </AvatarFallback>
            </Avatar>
            <span class="absolute -bottom-0.5 -right-0.5 flex items-center justify-center w-4 h-4 rounded-full bg-background border border-border">
              <component :is="conversation.inbox_channel === 'livechat' ? MessageSquare : Mail" class="w-2.5 h-2.5 text-muted-foreground" />
            </span>
          </div>

          <!-- Content container -->
          <div class="flex-1 min-w-0 space-y-2">
            <!-- Name + Subject group -->
            <div>
              <!-- Contact name + inbox + time -->
              <div class="flex items-baseline justify-between gap-2">
                <div class="flex items-baseline gap-1.5 min-w-0">
                  <h3 class="text-sm font-semibold truncate text-foreground">
                    {{ contactFullName }}
                  </h3>
                  <span class="text-xs text-muted-foreground truncate">
                    {{ conversation.inbox_name }}
                  </span>
                </div>
                <span
                  class="text-xs text-muted-foreground whitespace-nowrap flex-shrink-0 tabular-nums"
                  v-if="conversation.last_message_at"
                >
                  {{ relativeLastMessageTime }}
                </span>
              </div>

              <!-- Subject -->
              <p
                v-if="conversation.subject"
                class="text-xs text-muted-foreground truncate"
              >
                {{ conversation.subject }}
              </p>
            </div>

            <!-- Message preview + unread count -->
            <div class="flex items-center justify-between gap-2">
              <p class="text-sm flex-1 min-w-0 truncate text-muted-foreground">
                <template v-if="hasDraftForConversation">
                  <span class="font-medium text-primary">{{ $t('globals.terms.draft') }}:</span>
                  {{ draftPreview }}
                </template>
                <template v-else>
                  <Reply
                    class="text-green-600 inline-block align-text-bottom mr-0.5"
                    :size="14"
                    v-if="conversation.last_message_sender === 'agent'"
                  />{{ trimmedLastMessage }}
                </template>
              </p>
              <div
                v-if="conversation.unread_message_count > 0"
                class="flex items-center justify-center w-5 h-5 bg-green-600 text-white text-xs font-medium rounded-full flex-shrink-0"
              >
                {{ conversation.unread_message_count }}
              </div>
            </div>

            <!-- SLA Badges -->
            <div v-if="hasSlaDeadlines" class="flex items-center gap-1">
              <SlaBadge
                v-show="frdStatus === 'overdue' || frdStatus === 'remaining'"
                :dueAt="conversation.first_response_deadline_at"
                :actualAt="conversation.first_reply_at"
                :label="'FRD'"
                :showExtra="false"
                @status="frdStatus = $event"
                :key="`${conversation.uuid}-${conversation.first_response_deadline_at}-${conversation.first_reply_at}`"
              />
              <SlaBadge
                v-show="rdStatus === 'overdue' || rdStatus === 'remaining'"
                :dueAt="conversation.resolution_deadline_at"
                :actualAt="conversation.resolved_at"
                :label="'RD'"
                :showExtra="false"
                @status="rdStatus = $event"
                :key="`${conversation.uuid}-${conversation.resolution_deadline_at}-${conversation.resolved_at}`"
              />
              <SlaBadge
                v-show="nrdStatus === 'overdue' || nrdStatus === 'remaining'"
                :dueAt="conversation.next_response_deadline_at"
                :actualAt="conversation.next_response_met_at"
                :label="'NRD'"
                :showExtra="false"
                @status="nrdStatus = $event"
                :key="`${conversation.uuid}-${conversation.next_response_deadline_at}-${conversation.next_response_met_at}`"
              />
            </div>
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
import { Mail, MessageSquare, Reply, MailOpen } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger
} from '@shared-ui/components/ui/context-menu'
import SlaBadge from '@main/features/sla/SlaBadge.vue'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { useConversationStore } from '@main/stores/conversation'
import { useI18n } from 'vue-i18n'

let timer = null
const now = ref(new Date())
const route = useRoute()
const conversationStore = useConversationStore()
const { t } = useI18n()
const frdStatus = ref('')
const rdStatus = ref('')
const nrdStatus = ref('')

const props = defineProps({
  conversation: Object,
  currentConversation: Object,
  contactFullName: String
})

const handleMarkAsUnread = () => {
  conversationStore.markAsUnread(props.conversation.uuid)
}

const conversationRoute = computed(() => {
  const baseRoute = route.name.includes('team')
    ? 'team-inbox-conversation'
    : route.name.includes('view')
      ? 'view-inbox-conversation'
      : 'inbox-conversation'
  return {
    name: baseRoute,
    params: {
      uuid: props.conversation.uuid,
      ...(baseRoute === 'team-inbox-conversation' && { teamID: route.params.teamID }),
      ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID })
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
  return conversationStore.hasDraft(props.conversation.uuid)
})

const draftPreview = computed(() => {
  const draft = conversationStore.getDraft(props.conversation.uuid)
  if (!draft?.content) return ''
  const text = draft.content.replace(/<[^>]*>/g, '').trim()
  return text.length > 120 ? text.slice(0, 120) + '...' : text
})

const isItemSelected = computed(() => {
  return conversationStore.isSelected(props.conversation.uuid)
})

const handleCheckboxClick = (event) => {
  conversationStore.toggleSelect(props.conversation.uuid, event.shiftKey)
}
</script>
