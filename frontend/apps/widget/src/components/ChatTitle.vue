<template>
  <div class="flex items-center justify-center gap-3">
    <Avatar class="size-10">
      <AvatarImage :src="chatTitle.avatarUrl" />
      <AvatarFallback>{{ chatTitle.avatarFallback }}</AvatarFallback>
    </Avatar>
    <div class="flex flex-col">
      <h3 class="text-base font-bold text-foreground">
        {{ chatTitle.name }}
      </h3>
      <p class="text-xs text-muted-foreground">
        <!-- Show business hours status meaning we are out of business hours -->
        <span v-if="businessHoursStatus">
          {{ businessHoursStatus }}
        </span>
        <span
          v-else-if="
            chatTitle.availability_status === 'online' || chatTitle.availability_status === 'away'
          "
        >
          <span
            class="inline-block w-2 h-2 rounded-full mr-1"
            :class="{
              'bg-green-500': chatTitle.availability_status === 'online',
              'bg-amber-500': chatTitle.availability_status === 'away'
            }"
          ></span>
          {{
            chatTitle.availability_status.charAt(0).toUpperCase() +
            chatTitle.availability_status.slice(1)
          }}
        </span>
        <span v-else-if="chatStore.currentConversation?.assignee?.active_at">
          {{ $t('globals.terms.active') }}
          {{ getRelativeTime(chatStore.currentConversation?.assignee?.active_at).toLowerCase() }}
        </span>
      </p>
    </div>
  </div>
</template>

<script setup>
import { useChatStore } from '@widget/store/chat.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { computed } from 'vue'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { useBusinessHours } from '@widget/composables/useBusinessHours.js'

const chatStore = useChatStore()
const widgetStore = useWidgetStore()
const { resolveBusinessHours, getBusinessHoursStatus } = useBusinessHours()

const businessHoursStatus = computed(() => {
  const config = widgetStore.config

  // Show business hrs?
  if (!config.show_office_hours_in_chat) {
    return null
  }

  const conversation = chatStore.currentConversation
  if (!conversation) {
    return null
  }

  const businessHours = resolveBusinessHours({
    showOfficeHours: config.show_office_hours_in_chat,
    showAfterAssignment: config.show_office_hours_after_assignment,
    assignedBusinessHoursId: conversation.business_hours_id,
    defaultBusinessHoursId: config.default_business_hours_id,
    businessHoursList: config.business_hours
  })

  if (!businessHours) {
    return null
  }

  const utcOffset = conversation.working_hours_utc_offset || 0
  const withinHoursMessage = config.chat_reply_expectation_message || ''

  const { status, isWithin } = getBusinessHoursStatus(businessHours, utcOffset, withinHoursMessage)
  return isWithin ? null : status
})

const chatTitle = computed(() => {
  const config = widgetStore.config
  const assignee = chatStore.currentConversation?.assignee
  if (assignee?.id && assignee?.id > 0) {
    return {
      name: assignee.first_name,
      avatarUrl: assignee.avatar_url || '',
      avatarFallback: assignee.first_name.charAt(0).toUpperCase(),
      availability_status: assignee.availability_status,
      hasAssignee: true
    }
  }
  // Default brand values
  return {
    name: config.brand_name,
    avatarUrl: config.launcher?.logo_url || '',
    avatarFallback: config.brand_name.charAt(0).toUpperCase(),
    availability_status: null,
    hasAssignee: false
  }
})
</script>
