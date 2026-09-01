<template>
  <div class="zendesk-ticket-breadcrumb flex items-stretch justify-between shrink-0">
    <nav class="zendesk-breadcrumb-nav min-w-0" aria-label="breadcrumb">
      <router-link :to="listRoute" class="zendesk-breadcrumb-segment zendesk-breadcrumb-link">
        {{ viewLabel }}
      </router-link>
      <span class="zendesk-breadcrumb-segment truncate max-w-[14rem]">
        {{ requesterName }}
      </span>
      <span class="zendesk-breadcrumb-segment zendesk-breadcrumb-current flex items-center gap-2 min-w-0">
        <span
          v-if="status"
          class="zendesk-status-badge shrink-0"
          :class="statusClass"
        >
          {{ status }}
        </span>
        <span v-if="ticketNumber" class="shrink-0 whitespace-nowrap">
          {{ t('zendesk.ticketNumber', { number: ticketNumber }) }}
        </span>
      </span>
      <span
        v-if="hasSla"
        class="zendesk-breadcrumb-segment shrink-0"
      >
        <ZendeskSlaIndicator :conversation="conversationStore.current" />
      </span>
    </nav>

    <div class="zendesk-breadcrumb-meta flex items-stretch shrink-0">
      <div class="flex items-center px-3 gap-2 h-full">
        <Button variant="ghost" size="sm" class="h-7 text-xs hidden lg:inline-flex" @click="emit('next')" :disabled="!hasNext">
          {{ t('zendesk.next') }}
          <ArrowRight class="size-3.5 ml-1" />
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowRight } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { useInboxViewContext } from '@main/composables/useInboxViewContext'
import { useStatusCategory } from '@main/composables/useStatusCategory'
import { useConversationStore } from '@main/stores/conversation'
import ZendeskSlaIndicator from './ZendeskSlaIndicator.vue'

const props = defineProps({
  requesterName: { type: String, default: '' },
  status: { type: String, default: '' },
  ticketNumber: { type: [String, Number], default: '' },
  hasNext: { type: Boolean, default: false }
})

const emit = defineEmits(['next'])

const { t } = useI18n()
const { viewLabel, listRoute } = useInboxViewContext()
const { categoryClass } = useStatusCategory()
const conversationStore = useConversationStore()

const statusClass = computed(() => categoryClass(props.status))

const hasSla = computed(() => {
  const c = conversationStore.current
  return !!(c?.first_response_deadline_at || c?.resolution_deadline_at || c?.next_response_deadline_at)
})
</script>
