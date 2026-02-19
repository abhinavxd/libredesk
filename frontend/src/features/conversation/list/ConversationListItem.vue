<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <router-link
        :to="conversationRoute"
        class="group relative block px-4 p-4 transition-all duration-200 ease-in-out cursor-pointer hover:bg-accent/20 dark:hover:bg-accent/60"
        :class="{
          'bg-accent/60': conversation.uuid === currentConversation?.uuid
        }"
      >
        <div class="flex items-start gap-4">
          <!-- Avatar -->
          <Avatar class="w-12 h-12 rounded-full shadow">
            <AvatarImage
              :src="conversation.contact.avatar_url || ''"
              class="object-cover"
              v-if="conversation.contact.avatar_url || ''"
            />
            <AvatarFallback>
              {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
            </AvatarFallback>
          </Avatar>

          <!-- Content container -->
          <div class="flex-1 min-w-0 space-y-2">
            <!-- Contact name and last message time -->
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-1.5 min-w-0">
                <h3 class="text-sm font-semibold truncate">
                  {{ contactFullName }}
                </h3>
                <Pencil
                  v-if="hasDraftForConversation"
                  class="w-3 h-3 text-muted-foreground flex-shrink-0"
                />
              </div>
              <span
                class="text-xs text-gray-400 whitespace-nowrap"
                v-if="conversation.last_message_at"
              >
                {{ relativeLastMessageTime }}
              </span>
            </div>

            <!-- Subject -->
            <p v-if="conversation.subject" class="text-xs font-medium text-muted-foreground truncate">
              {{ conversation.subject }}
            </p>

            <!-- Inbox name -->
            <p class="text-xs text-gray-400 flex items-center gap-1.5">
              <Mail class="w-3.5 h-3.5 text-gray-400/80" />
              <span>{{ conversation.inbox_name }}</span>
            </p>

            <!-- Message preview and unread count -->
            <div class="flex items-start justify-between gap-2">
              <div
                class="text-sm flex items-center gap-1.5 flex-1 break-all text-gray-600 dark:text-gray-300"
              >
                <Reply
                  class="text-green-600 flex-shrink-0"
                  size="15"
                  v-if="conversation.last_message_sender === 'agent'"
                />
                {{ trimmedLastMessage }}
              </div>
              <div
                v-if="conversation.unread_message_count > 0"
                class="flex items-center justify-center w-6 h-6 bg-green-600 text-white text-xs font-medium rounded-full"
              >
                {{ conversation.unread_message_count }}
              </div>
            </div>

            <!-- SLA Badges -->
            <div class="flex items-center">
              <div :class="getSlaClass(frdStatus)">
                <SlaBadge
                  :dueAt="conversation.first_response_deadline_at"
                  :actualAt="conversation.first_reply_at"
                  :label="'FRD'"
                  :showExtra="false"
                  @status="frdStatus = $event"
                  :key="`${conversation.uuid}-${conversation.first_response_deadline_at}-${conversation.first_reply_at}`"
                />
              </div>
              <div :class="getSlaClass(rdStatus)">
                <SlaBadge
                  :dueAt="conversation.resolution_deadline_at"
                  :actualAt="conversation.resolved_at"
                  :label="'RD'"
                  :showExtra="false"
                  @status="rdStatus = $event"
                  :key="`${conversation.uuid}-${conversation.resolution_deadline_at}-${conversation.resolved_at}`"
                />
              </div>
              <div :class="getSlaClass(nrdStatus)">
                <SlaBadge
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
import { getRelativeTime } from '@/utils/datetime'
import { Mail, Reply, Pencil, MailOpen } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger
} from '@/components/ui/context-menu'
import SlaBadge from '@/features/sla/SlaBadge.vue'
import { useConversationStore } from '@/stores/conversation'

let timer = null
const now = ref(new Date())
const route = useRoute()
const conversationStore = useConversationStore()
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
  return message.length > 100 ? message.slice(0, 100) + '...' : message
})

const getSlaClass = (status) => (['overdue', 'remaining'].includes(status) ? 'mr-2' : '')

const relativeLastMessageTime = computed(() => {
  return props.conversation.last_message_at
    ? getRelativeTime(props.conversation.last_message_at, now.value)
    : ''
})

const hasDraftForConversation = computed(() => {
  return conversationStore.hasDraft(props.conversation.uuid)
})
</script>
