<template>
  <div class="space-y-3">
    <dl :class="INFO_GRID_CLASS">
      <template v-if="conversation.inbox_name">
        <dt class="sidebar-label">{{ $t('globals.terms.inbox', 1) }}</dt>
        <dd :class="INFO_VALUE_CLASS">
          <div class="flex items-start justify-end gap-1.5">
            <component
              :is="channelIcon"
              class="mt-0.5 size-3.5 shrink-0 text-muted-foreground"
              aria-hidden="true"
            />
            <span class="min-w-0">{{ conversation.inbox_name }}</span>
          </div>
        </dd>
      </template>

      <template v-if="conversation.subject">
        <dt class="sidebar-label">{{ $t('globals.terms.subject') }}</dt>
        <dd :class="INFO_VALUE_CLASS">{{ conversation.subject }}</dd>
      </template>

      <dt class="sidebar-label">{{ $t('globals.terms.referenceNumber') }}</dt>
      <dd :class="INFO_VALUE_CLASS">{{ conversation.reference_number }}</dd>

      <dt class="sidebar-label">{{ $t('globals.terms.initiatedAt') }}</dt>
      <dd :class="INFO_VALUE_CLASS">
        {{ conversation.created_at ? format(conversation.created_at, 'PPpp') : '-' }}
      </dd>

      <dt class="sidebar-label flex flex-wrap items-center gap-1.5">
        <span>{{ $t('globals.terms.firstReplyAt') }}</span>
        <SlaBadge
          v-if="conversation.first_response_deadline_at"
          :dueAt="conversation.first_response_deadline_at"
          :actualAt="conversation.first_reply_at"
          :key="`${conversation.uuid}-${conversation.first_response_deadline_at}-${conversation.first_reply_at}`"
        />
      </dt>
      <dd :class="INFO_VALUE_CLASS">
        {{ conversation.first_reply_at ? format(conversation.first_reply_at, 'PPpp') : '-' }}
      </dd>

      <dt class="sidebar-label flex flex-wrap items-center gap-1.5">
        <span>{{ $t('globals.terms.resolvedAt') }}</span>
        <SlaBadge
          v-if="conversation.resolution_deadline_at"
          :dueAt="conversation.resolution_deadline_at"
          :actualAt="conversation.resolved_at"
          :key="`${conversation.uuid}-${conversation.resolution_deadline_at}-${conversation.resolved_at}`"
        />
      </dt>
      <dd :class="INFO_VALUE_CLASS">
        {{ conversation.resolved_at ? format(conversation.resolved_at, 'PPpp') : '-' }}
      </dd>

      <dt class="sidebar-label flex flex-wrap items-center gap-1.5">
        <span>{{ $t('globals.terms.lastReplyAt') }}</span>
        <SlaBadge
          v-if="conversation.next_response_deadline_at"
          :dueAt="conversation.next_response_deadline_at"
          :actualAt="conversation.next_response_met_at"
          :key="`${conversation.uuid}-${conversation.next_response_deadline_at}-${conversation.next_response_met_at}`"
        />
      </dt>
      <dd :class="INFO_VALUE_CLASS">
        {{ conversation.last_reply_at ? format(conversation.last_reply_at, 'PPpp') : '-' }}
      </dd>

      <template v-if="conversation.closed_at">
        <dt class="sidebar-label">{{ $t('globals.terms.closedAt') }}</dt>
        <dd :class="INFO_VALUE_CLASS">{{ format(conversation.closed_at, 'PPpp') }}</dd>
      </template>

      <template v-if="conversation.sla_policy_name">
        <dt class="sidebar-label">{{ $t('globals.terms.slaPolicy') }}</dt>
        <dd :class="INFO_VALUE_CLASS">{{ conversation.sla_policy_name }}</dd>
      </template>
    </dl>

    <CustomAttributes
      v-if="customAttributeStore.conversationAttributeOptions.length > 0"
      :loading="conversationStore.conversation.loading"
      :attributes="customAttributeStore.conversationAttributeOptions"
      :custom-attributes="conversation.custom_attributes || {}"
      @update:setattributes="updateCustomAttributes"
    />

    <dl
      v-if="
        conversation.csat_responded_at && (conversation.csat_rating || conversation.csat_feedback)
      "
      :class="INFO_GRID_CLASS"
    >
      <template v-if="conversation.csat_rating">
        <dt class="sidebar-label">{{ $t('globals.terms.csatRating') }}</dt>
        <dd :class="INFO_VALUE_CLASS">
          <div class="flex flex-wrap items-center justify-end gap-2">
            <span class="text-lg">{{ csatRatingEmoji(conversation.csat_rating) }}</span>
            <span>{{ $t(csatRatingTextKey(conversation.csat_rating)) }}</span>
            <span class="text-xs text-muted-foreground">{{ conversation.csat_rating }}/5</span>
          </div>
        </dd>
      </template>

      <template v-if="conversation.csat_feedback">
        <dt class="sidebar-label col-span-2">{{ $t('globals.terms.csatFeedback') }}</dt>
        <dd class="col-span-2 min-w-0">
          <p
            class="sidebar-value italic whitespace-pre-wrap [overflow-wrap:anywhere]"
            :class="{ 'line-clamp-3': isFeedbackLong && !feedbackExpanded }"
          >
            {{ conversation.csat_feedback }}
          </p>
          <button
            v-if="isFeedbackLong"
            type="button"
            class="text-xs text-muted-foreground hover:text-foreground mt-1"
            @click="feedbackExpanded = !feedbackExpanded"
          >
            {{
              feedbackExpanded ? $t('globals.messages.showLess') : $t('globals.messages.showMore')
            }}
          </button>
        </dd>
      </template>
    </dl>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { format } from 'date-fns'
import { Mail } from 'lucide-vue-next'
import { CHANNEL_ICONS } from '@main/constants/channelIcons.js'
import SlaBadge from '@/features/sla/SlaBadge.vue'
import { useConversationStore } from '@main/stores/conversation'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '@main/stores/customAttributes'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { csatRatingEmoji, csatRatingTextKey } from '@shared-ui/utils/csat.js'
import api from '@main/api'
import { useI18n } from 'vue-i18n'

const INFO_GRID_CLASS = 'grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-start gap-x-4 gap-y-3'
const INFO_VALUE_CLASS = 'sidebar-value min-w-0 text-right [overflow-wrap:anywhere]'

const emitter = useEmitter()
const { t } = useI18n()
const customAttributeStore = useCustomAttributeStore()
const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)

const channelIcon = computed(() => CHANNEL_ICONS[conversation.value?.inbox_channel] || Mail)
customAttributeStore.fetchCustomAttributes()

const feedbackExpanded = ref(false)
const isFeedbackLong = computed(() => (conversation.value?.csat_feedback?.length || 0) > 160)

const updateCustomAttributes = async (attributes) => {
  let previousAttributes = conversationStore.current.custom_attributes
  try {
    conversationStore.current.custom_attributes = attributes
    await api.updateConversationCustomAttribute(conversation.value.uuid, attributes)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
    conversationStore.current.custom_attributes = previousAttributes
  }
}
</script>
