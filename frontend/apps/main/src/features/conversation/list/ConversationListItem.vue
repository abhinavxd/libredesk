<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <router-link
        :to="conversationRoute"
        :aria-current="isCurrent ? 'page' : undefined"
        class="group relative block cursor-pointer px-3 py-3 transition-colors duration-150 focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
        :class="{
          'bg-accent before:absolute before:inset-y-3 before:left-0 before:w-0.5 before:rounded-r before:bg-primary': isCurrent,
          'bg-primary/5 can-hover:hover:bg-primary/10': isItemSelected && !isCurrent,
          'can-hover:hover:bg-accent/40': !isCurrent && !isItemSelected
        }"
      >
        <div class="flex items-start gap-2">
          <div class="relative flex-shrink-0 w-10 h-10">
            <div
              class="transition-opacity"
              :class="avatarOpacityClass"
              :aria-hidden="showCheckbox"
            >
              <Avatar class="w-10 h-10 rounded-full">
                <AvatarImage
                  :src="conversation.contact.avatar_url || ''"
                  class="object-cover"
                />
                <AvatarFallback>
                  {{ conversation.contact.first_name.substring(0, 2).toUpperCase() }}
                </AvatarFallback>
              </Avatar>
            </div>
            <div
              v-if="canBulkAct"
              class="absolute inset-0 items-center justify-center"
              :class="showCheckbox ? 'flex' : 'hidden can-hover:group-hover:flex'"
              @click.prevent.stop="handleCheckboxClick"
            >
              <Checkbox
                :checked="isItemSelected"
                :aria-label="t('conversation.bulkActions.selectConversation')"
                class="w-5 h-5"
              />
            </div>
          </div>

          <!-- Content container -->
          <div class="flex-1 min-w-0 space-y-1.5">
            <!-- Name + Subject group -->
            <div class="space-y-0.5">
              <!-- Contact name + channel + time -->
              <div class="flex items-baseline justify-between gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <h3
                      class="ui-subheading truncate min-w-0"
                      :class="isUnread ? 'font-semibold' : 'font-medium'"
                    >
                      {{ contactFullName }}
                    </h3>
                  </TooltipTrigger>
                  <TooltipContent>{{ contactFullName }}</TooltipContent>
                </Tooltip>
                <div class="flex items-center gap-1 flex-shrink-0">
                  <Tooltip v-if="nearestSlaDeadline">
                    <TooltipTrigger as-child>
                      <div role="img" :aria-label="`${t(nearestSlaDeadline.title)}: ${formatFullTimestamp(nearestSlaDeadline.dueAt)}`">
                        <SlaBadge :dueAt="nearestSlaDeadline.dueAt" compact :showExtra="false" />
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>
                      <div v-for="deadline in pendingSlaDeadlines" :key="deadline.label" class="py-1">
                        <p>{{ t(deadline.title) }}</p>
                        <p>{{ formatFullTimestamp(deadline.dueAt) }}</p>
                      </div>
                    </TooltipContent>
                  </Tooltip>
                  <PriorityMarker :priority="conversation.priority" />
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <component
                        :is="channelIcon"
                        class="size-3 shrink-0 text-muted-foreground/60"
                        role="img"
                        :aria-label="conversation.inbox_name"
                      />
                    </TooltipTrigger>
                    <TooltipContent>{{ conversation.inbox_name }}</TooltipContent>
                  </Tooltip>
                  <Tooltip v-if="isAwaitingResponse">
                    <TooltipTrigger as-child>
                      <span class="inline-flex shrink-0 items-center gap-1 whitespace-nowrap text-xs text-muted-foreground tabular-nums">
                        <span class="sr-only">{{ t('globals.terms.waiting') }}</span>
                        <span>{{ relativeWaitingTime }}</span>
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{{ t('globals.terms.waitingSince') }}</p>
                      <p>{{ formatFullTimestamp(conversation.waiting_since) }}</p>
                    </TooltipContent>
                  </Tooltip>
                  <span v-if="isAwaitingResponse && conversation.last_message_at" class="text-xs text-muted-foreground" aria-hidden="true">·</span>
                  <Tooltip v-if="conversation.last_message_at">
                    <TooltipTrigger as-child>
                      <span class="inline-flex items-center gap-1 text-xs text-muted-foreground whitespace-nowrap tabular-nums">
                        {{ relativeLastMessageTime }}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{{ formatFullTimestamp(conversation.last_message_at) }}</p>
                    </TooltipContent>
                  </Tooltip>
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
                  <span class="italic text-foreground">{{ $t('globals.terms.typing') }}</span>
                </template>
                <template v-else-if="hasDraftForConversation && !isCurrent">
                  <span class="font-medium text-foreground">{{ $t('globals.terms.draft') }}:</span>
                  {{ draftPreview }}
                </template>
                <template v-else>
                  <Reply
                    class="text-success inline-block align-[-2px] mr-0.5"
                    :size="14"
                    v-if="conversation.last_message_sender === 'agent'"
                  />{{ trimmedLastMessage }}
                </template>
              </p>
              <Tooltip v-if="conversation.assigned_user_id || conversation.assigned_team_id">
                <TooltipTrigger as-child>
                  <span class="flex size-5 shrink-0 items-center justify-center text-muted-foreground" role="img" :aria-label="ownerLabel">
                    <Avatar v-if="assignedAgent || (!conversation.assigned_user_id && assignedTeam)" :shape="assignedAgent ? 'circle' : 'square'" class="size-5 text-[9px]">
                      <AvatarImage v-if="assignedAgent" :src="assignedAgent.avatar_url || ''" class="object-cover" />
                      <AvatarFallback>{{ ownerInitials }}</AvatarFallback>
                    </Avatar>
                    <component :is="ownerIcon" v-else class="size-4" aria-hidden="true" />
                  </span>
                </TooltipTrigger>
                <TooltipContent>{{ ownerLabel }}</TooltipContent>
              </Tooltip>
              <div
                v-if="isUnread"
                class="flex items-center justify-center w-5 h-5 bg-primary text-primary-foreground text-xs font-medium rounded-full flex-shrink-0"
              >
                {{ conversation.unread_message_count > 9 ? '9+' : conversation.unread_message_count }}
              </div>
            </div>

          </div>
        </div>
      </router-link>
    </ContextMenuTrigger>
    <ContextMenuContent>
      <!-- Long press is the only way to reach the first checkbox on touch. -->
      <ContextMenuItem v-if="canBulkAct && !showCheckbox" @click="handleSelect">
        <SquareCheck class="w-4 h-4 mr-2" />
        {{ $t('conversation.bulkActions.selectConversation') }}
      </ContextMenuItem>
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
import { getRelativeTime, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import { Mail, Reply, MailOpen, SquareCheck, UserRoundCheck, Users } from 'lucide-vue-next'
import { CHANNEL_ICONS } from '@main/constants/channelIcons.js'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger
} from '@shared-ui/components/ui/context-menu'
import SlaBadge from '@main/features/sla/SlaBadge.vue'
import PriorityMarker from '@main/features/conversation/PriorityMarker.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { useConversationStore } from '@main/stores/conversation'
import { useUsersStore } from '@main/stores/users'
import { useTeamStore } from '@main/stores/team'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useI18n } from 'vue-i18n'

let timer = null
const now = ref(new Date())
const route = useRoute()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamStore = useTeamStore()
const appSettingsStore = useAppSettingsStore()
const { canBulkAct } = useBulkActionPermissions()
const { t } = useI18n()

const props = defineProps({
  conversation: Object,
  currentConversation: Object,
  contactFullName: String
})

const handleMarkAsUnread = () => {
  conversationStore.markAsUnread(props.conversation.uuid)
}

const channelIcon = computed(() => CHANNEL_ICONS[props.conversation.inbox_channel] || Mail)

const conversationRoute = computed(() => {
  const baseRoute = route.params.teamID
    ? 'team-inbox-conversation'
    : route.params.viewID
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

const conversationStatus = computed(() => conversationStore.statuses.find((status) => status.name === props.conversation.status))
const isAwaitingResponse = computed(() => !!props.conversation.waiting_since && conversationStatus.value?.category === 'open')
const relativeLastMessageTime = computed(() => props.conversation.last_message_at
  ? getRelativeTime(props.conversation.last_message_at, now.value) : '')
const relativeWaitingTime = computed(() => props.conversation.waiting_since
  ? getRelativeTime(props.conversation.waiting_since, now.value) : '')
const assignedAgent = computed(() => usersStore.users.find((agent) =>
  String(agent.id) === String(props.conversation.assigned_user_id)))
const assignedTeam = computed(() => teamStore.teams.find((team) =>
  String(team.id) === String(props.conversation.assigned_team_id)))
const ownerLabel = computed(() => {
  if (assignedAgent.value) {
    return [assignedAgent.value.first_name, assignedAgent.value.last_name].filter(Boolean).join(' ')
  }
  if (props.conversation.assigned_user_id) return t('globals.terms.assignee')
  if (props.conversation.assigned_team_id) return assignedTeam.value?.name || t('globals.terms.team', 1)
  return t('globals.terms.unassigned')
})
const ownerInitials = computed(() => {
  if (assignedAgent.value) {
    return [assignedAgent.value.first_name, assignedAgent.value.last_name]
      .map((name) => name?.charAt(0) || '').join('').toUpperCase()
  }
  const words = (assignedTeam.value?.name || '').trim().split(/\s+/)
  return (words.length > 1
    ? words.slice(0, 2).map((word) => word.charAt(0)).join('')
    : words[0].slice(0, 2)).toUpperCase()
})
const ownerIcon = computed(() => props.conversation.assigned_user_id
  ? UserRoundCheck
  : Users)

const pendingSlaDeadlines = computed(() => {
  const c = props.conversation
  if (conversationStatus.value?.category === 'resolved') return []
  return [
    { label: 'FRD', title: 'admin.sla.firstResponseTime', dueAt: c.first_response_deadline_at, actualAt: c.first_reply_at },
    { label: 'RD', title: 'admin.sla.resolutionTime', dueAt: c.resolution_deadline_at, actualAt: c.resolved_at },
    { label: 'NRD', title: 'admin.sla.nextResponseTime', dueAt: c.next_response_deadline_at, actualAt: c.next_response_met_at }
  ].filter((deadline) => deadline.dueAt && !deadline.actualAt)
    .sort((a, b) => Date.parse(a.dueAt) - Date.parse(b.dueAt))
})
const nearestSlaDeadline = computed(() => pendingSlaDeadlines.value[0])

const hasDraftForConversation = computed(() => {
  return conversationStore.conversationHasDraft(props.conversation.uuid)
})

const isTyping = computed(() => conversationStore.typingByUUID[props.conversation.uuid] === true)

const draftPreview = computed(() => {
  const draft = conversationStore.conversationDraftPreview(props.conversation.uuid)
  if (!draft?.content && !draft?.meta?.attachments?.length) return ''
  const text = (draft.content || '').replace(/<[^>]*>/g, '').trim()
  if (text) return text.length > 120 ? text.slice(0, 120) + '...' : text
  if (draft.meta?.attachments?.length) return conversationStore.getMediaPreview(draft.meta.attachments)
  if (/<img\b/i.test(draft.content || '')) return t('globals.terms.image', 1)
  return ''
})

const showSubject = computed(
  () => appSettingsStore.settings['app.show_conversation_subject'] !== false
)

const isUnread = computed(() => props.conversation.unread_message_count > 0)

const isCurrent = computed(() => props.conversation.uuid === props.currentConversation?.uuid)

const isItemSelected = computed(() => {
  return conversationStore.isSelected(props.conversation.uuid)
})

const showCheckbox = computed(() => {
  if (!canBulkAct.value) return false
  return isItemSelected.value || conversationStore.selectedCount > 0
})

const avatarOpacityClass = computed(() => {
  if (showCheckbox.value) return 'opacity-0'
  if (canBulkAct.value) return 'opacity-100 can-hover:group-hover:opacity-0'
  return 'opacity-100'
})

const handleCheckboxClick = (event) => {
  conversationStore.toggleSelect(props.conversation.uuid, event.shiftKey)
}

const handleSelect = () => {
  conversationStore.toggleSelect(props.conversation.uuid, false)
}
</script>
