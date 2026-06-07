<template>
  <div class="space-y-3">
    <div v-if="conversation.inbox_name">
      <p class="sidebar-label">{{ $t('globals.terms.inbox', 1) }}</p>
      <div class="flex items-center gap-1.5">
        <component
          :is="conversation.inbox_channel === 'livechat' ? MessageSquare : Mail"
          class="size-3.5 text-muted-foreground flex-shrink-0"
        />
        <p class="sidebar-value break-all">{{ conversation.inbox_name }}</p>
      </div>
    </div>

    <div v-if="conversation.subject">
      <p class="sidebar-label">{{ $t('globals.terms.subject') }}</p>
      <p class="sidebar-value break-all">
        {{ conversation.subject }}
      </p>
    </div>

    <div>
      <p class="sidebar-label">{{ $t('globals.terms.referenceNumber') }}</p>
      <p class="sidebar-value">
        {{ conversation.reference_number }}
      </p>
    </div>

    <div>
      <p class="sidebar-label">{{ $t('globals.terms.initiatedAt') }}</p>
      <p v-if="conversation.created_at" class="sidebar-value">
        {{ format(conversation.created_at, 'PPpp') }}
      </p>
      <p v-else class="sidebar-value">-</p>
    </div>

    <div>
      <div class="flex items-center gap-2">
        <p class="sidebar-label">{{ $t('globals.terms.firstReplyAt') }}</p>
        <SlaBadge
          v-if="conversation.first_response_deadline_at"
          :dueAt="conversation.first_response_deadline_at"
          :actualAt="conversation.first_reply_at"
          :key="`${conversation.uuid}-${conversation.first_response_deadline_at}-${conversation.first_reply_at}`"
        />
      </div>
      <p v-if="conversation.first_reply_at" class="sidebar-value">
        {{ format(conversation.first_reply_at, 'PPpp') }}
      </p>
      <p v-else class="sidebar-value">-</p>
    </div>

    <div>
      <div class="flex items-center gap-2">
        <p class="sidebar-label">{{ $t('globals.terms.resolvedAt') }}</p>
        <SlaBadge
          v-if="conversation.resolution_deadline_at"
          :dueAt="conversation.resolution_deadline_at"
          :actualAt="conversation.resolved_at"
          :key="`${conversation.uuid}-${conversation.resolution_deadline_at}-${conversation.resolved_at}`"
        />
      </div>
      <p v-if="conversation.resolved_at" class="sidebar-value">
        {{ format(conversation.resolved_at, 'PPpp') }}
      </p>
      <p v-else class="sidebar-value">-</p>
    </div>

    <div>
      <div class="flex items-center gap-2">
        <p class="sidebar-label">{{ $t('globals.terms.lastReplyAt') }}</p>
        <SlaBadge
          v-if="conversation.next_response_deadline_at"
          :dueAt="conversation.next_response_deadline_at"
          :actualAt="conversation.next_response_met_at"
          :key="`${conversation.uuid}-${conversation.next_response_deadline_at}-${conversation.next_response_met_at}`"
        />
      </div>
      <p v-if="conversation.last_reply_at" class="sidebar-value">
        {{ format(conversation.last_reply_at, 'PPpp') }}
      </p>
      <p v-else class="sidebar-value">-</p>
    </div>

    <div v-if="conversation.closed_at">
      <p class="sidebar-label">{{ $t('globals.terms.closedAt') }}</p>
      <p class="sidebar-value">
        {{ format(conversation.closed_at, 'PPpp') }}
      </p>
    </div>

    <div v-if="conversation.sla_policy_name">
      <p class="sidebar-label">{{ $t('globals.terms.slaPolicy') }}</p>
      <p class="sidebar-value">
        {{ conversation.sla_policy_name }}
      </p>
    </div>

    <CustomAttributes
      v-if="customAttributeStore.conversationAttributeOptions.length > 0"
      :loading="conversationStore.conversation.loading"
      :attributes="customAttributeStore.conversationAttributeOptions"
      :custom-attributes="conversation.custom_attributes || {}"
      @update:setattributes="updateCustomAttributes"
    />

    <div v-if="conversation.csat_responded_at && conversation.csat_rating">
      <p class="sidebar-label">{{ $t('globals.terms.csatRating') }}</p>
      <div class="flex items-center gap-2">
        <span class="text-lg">{{ csatRatingEmoji(conversation.csat_rating) }}</span>
        <span class="sidebar-value">{{ $t(csatRatingTextKey(conversation.csat_rating)) }}</span>
        <span class="text-xs text-muted-foreground">{{ conversation.csat_rating }}/5</span>
      </div>
    </div>

    <div v-if="conversation.csat_responded_at && conversation.csat_feedback">
      <p class="sidebar-label">{{ $t('globals.terms.csatFeedback') }}</p>
      <p
        class="sidebar-value italic whitespace-pre-wrap"
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
        {{ feedbackExpanded ? $t('globals.messages.showLess') : $t('globals.messages.showMore') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { format } from 'date-fns'
import { Mail, MessageSquare } from 'lucide-vue-next'
import SlaBadge from '@/features/sla/SlaBadge.vue'
import { useConversationStore } from '../../../stores/conversation'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '../../../stores/customAttributes'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { csatRatingEmoji, csatRatingTextKey } from '@shared-ui/utils/csat.js'
import api from '../../../api'
import { useI18n } from 'vue-i18n'

const emitter = useEmitter()
const { t } = useI18n()
const customAttributeStore = useCustomAttributeStore()
const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
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
