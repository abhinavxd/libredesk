<template>
  <ContextMenu>
    <ContextMenuTrigger asChild>
  <tr
    class="cursor-pointer"
    :class="{ selected: isCurrent || isSelected }"
    @click="openConversation"
    @mouseenter="onRowEnter"
    @mouseleave="onRowLeave"
    @contextmenu="onRowLeave"
  >
    <td @click.stop>
      <Checkbox
        v-if="canBulkAct"
        :checked="isSelected"
        class="ml-1"
        @update:checked="onCheckboxChange"
      />
    </td>
    <td>
      <span class="zendesk-status-badge" :class="statusClass">
        {{ conversation.status }}
      </span>
    </td>
    <td class="text-muted-foreground tabular-nums whitespace-nowrap">
      <span v-if="conversation.reference_number">#{{ conversation.reference_number }}</span>
    </td>
    <td>
      <div class="flex items-center gap-2 min-w-0">
        <span
          v-if="conversation.priority"
          class="zendesk-priority-dot shrink-0"
          :class="priorityClass"
          :title="conversation.priority"
        />
        <span class="text-[#1f73b7] hover:underline truncate">
          {{ conversation.subject || t('zendesk.noSubject') }}
        </span>
      </div>
    </td>
    <td class="text-muted-foreground whitespace-nowrap">
      <span class="inline-flex items-center gap-1.5">
        <component :is="channel.icon" class="size-3.5 shrink-0" aria-hidden="true" />
        {{ channel.label }}
      </span>
    </td>
    <td class="text-muted-foreground truncate max-w-[12rem]">
      {{ contactFullName }}
    </td>
    <td class="text-muted-foreground truncate max-w-[10rem]">
      <span v-if="assigneeName">{{ assigneeName }}</span>
      <span v-else class="text-muted-foreground/60">{{ t('zendesk.unassigned') }}</span>
    </td>
    <td class="text-muted-foreground whitespace-nowrap tabular-nums">
      <span v-if="conversation.created_at">{{ requestedDate }}</span>
    </td>
    <td>
      <ZendeskSlaIndicator :conversation="conversation" />
    </td>
  </tr>
    </ContextMenuTrigger>
    <TicketContextMenuItems :conversation="conversation" />
  </ContextMenu>

  <ZendeskRowPreview
    v-if="previewRect && !isCurrent"
    :conversation="conversation"
    :requester="contactFullName"
    :anchor="previewRect"
  />
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { formatShortDate } from '@shared-ui/utils/datetime.js'
import { useConversationStore } from '@main/stores/conversation'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useStatusCategory } from '@main/composables/useStatusCategory'
import { priorityDotClass } from '@main/composables/useConversationPriority'
import { channelMeta } from '@main/composables/useConversationChannel'
import { useAgentLookup } from '@main/composables/useAgentLookup'
import { ContextMenu, ContextMenuTrigger } from '@shared-ui/components/ui/context-menu'
import TicketContextMenuItems from '@/features/conversation/TicketContextMenuItems.vue'
import ZendeskSlaIndicator from './ZendeskSlaIndicator.vue'
import ZendeskRowPreview from './ZendeskRowPreview.vue'

const props = defineProps({
  conversation: { type: Object, required: true },
  contactFullName: { type: String, default: '' },
  isCurrent: { type: Boolean, default: false }
})

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const conversationStore = useConversationStore()
const { canBulkAct } = useBulkActionPermissions()
const { categoryClass } = useStatusCategory()
const { agentName } = useAgentLookup()

const statusClass = computed(() => categoryClass(props.conversation.status))
const priorityClass = computed(() => priorityDotClass(props.conversation.priority))
const channel = computed(() => channelMeta(props.conversation.inbox_channel))
const assigneeName = computed(() => agentName(props.conversation.assigned_user_id))
const requestedDate = computed(() =>
  props.conversation.created_at ? formatShortDate(props.conversation.created_at) : ''
)

// Hover preview: capture the row's rect so the floating card can anchor to it.
const previewRect = ref(null)
let previewTimer = null
const onRowEnter = (e) => {
  const rect = e.currentTarget.getBoundingClientRect()
  previewTimer = setTimeout(() => { previewRect.value = rect }, 350)
}
const onRowLeave = () => {
  clearTimeout(previewTimer)
  previewRect.value = null
}

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
      ...(baseRoute === 'view-inbox-conversation' && { viewID: route.params.viewID }),
      ...(baseRoute === 'inbox-conversation' && { type: route.params.type || 'assigned' })
    },
    query: props.conversation.mentioned_message_uuid
      ? { scrollTo: props.conversation.mentioned_message_uuid }
      : {}
  }
})

const isSelected = computed(() => conversationStore.isSelected(props.conversation.uuid))

const onCheckboxChange = () => {
  conversationStore.toggleSelect(props.conversation.uuid, false)
}

const openConversation = () => {
  router.push(conversationRoute.value)
}
</script>
