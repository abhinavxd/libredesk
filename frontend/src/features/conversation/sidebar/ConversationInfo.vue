<template>
  <div class="space-y-4">
    <div class="flex flex-col" v-if="conversation.subject">
      <p class="font-medium">{{ $t('form.field.subject') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-else>
        {{ conversation.subject }}
      </p>
    </div>

    <div class="flex flex-col">
      <p class="font-medium">{{ $t('form.field.referenceNumber') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-else>
        {{ conversation.reference_number }}
      </p>
    </div>
    <div class="flex flex-col">
      <p class="font-medium">{{ $t('form.field.initiatedAt') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-if="conversation.created_at">
        {{ format(conversation.created_at, 'PPpp') }}
      </p>
      <p v-else>-</p>
    </div>

    <div class="flex flex-col">
      <div class="flex justify-start items-center space-x-2">
        <p class="font-medium">{{ $t('form.field.firstReplyAt') }}</p>
        <SlaBadge
          v-if="conversation.first_response_deadline_at"
          :dueAt="conversation.first_response_deadline_at"
          :actualAt="conversation.first_reply_at"
          :key="conversation.uuid"
        />
      </div>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <div v-else>
        <p v-if="conversation.first_reply_at">
          {{ format(conversation.first_reply_at, 'PPpp') }}
        </p>
        <p v-else>-</p>
      </div>
    </div>

    <div class="flex flex-col">
      <div class="flex justify-start items-center space-x-2">
        <p class="font-medium">{{ $t('form.field.resolvedAt') }}</p>
        <SlaBadge
          v-if="conversation.resolution_deadline_at"
          :dueAt="conversation.resolution_deadline_at"
          :actualAt="conversation.resolved_at"
          :key="conversation.uuid"
        />
      </div>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <div v-else>
        <p v-if="conversation.resolved_at">
          {{ format(conversation.resolved_at, 'PPpp') }}
        </p>
        <p v-else>-</p>
      </div>
    </div>

    <div class="flex flex-col">
      <div class="flex justify-start items-center space-x-2">
        <p class="font-medium">{{ $t('form.field.lastReplyAt') }}</p>
        <SlaBadge
          v-if="conversation.next_response_deadline_at"
          :dueAt="conversation.next_response_deadline_at"
          :actualAt="
            ['met'].includes(conversation.next_response_sla_event_status)
              ? conversation.last_reply_at
              : null
          "
          :key="conversation.uuid"
        />
      </div>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-if="conversation.last_reply_at">
        {{ format(conversation.last_reply_at, 'PPpp') }}
      </p>
      <p v-else>-</p>
    </div>

    <div class="flex flex-col" v-if="conversation.closed_at">
      <p class="font-medium">{{ $t('form.field.closedAt') }}</p>
      <Skeleton v-if="conversationStore.conversation.loading" class="w-32 h-4" />
      <p v-else>
        {{ format(conversation.closed_at, 'PPpp') }}
      </p>
    </div>

    <div class="flex flex-col" v-if="conversation.sla_policy_name">
      <p class="font-medium">{{ $t('form.field.slaPolicy') }}</p>
      <div>
        <p>
          {{ conversation.sla_policy_name }}
        </p>
      </div>
    </div>

    <CustomAttributes
      v-if="customAttributeStore.conversationAttributeOptions.length > 0"
      :loading="conversationStore.conversation.loading"
      :attributes="customAttributeStore.conversationAttributeOptions"
      :custom-attributes="conversation.custom_attributes || {}"
      @update:setattributes="updateCustomAttributes"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { format } from 'date-fns'
import SlaBadge from '@/features/sla/SlaBadge.vue'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@/components/ui/skeleton'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import { useI18n } from 'vue-i18n'

const emitter = useEmitter()
const { t } = useI18n()
const customAttributeStore = useCustomAttributeStore()
const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
customAttributeStore.fetchCustomAttributes()

const updateCustomAttributes = async (attributes) => {
  let previousAttributes = conversationStore.current.custom_attributes
  try {
    conversationStore.current.custom_attributes = attributes
    await api.updateConversationCustomAttribute(conversation.value.uuid, attributes)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.attribute')
      })
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
