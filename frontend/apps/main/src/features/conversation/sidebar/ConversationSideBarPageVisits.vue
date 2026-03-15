<template>
  <div class="space-y-2">
    <div
      v-if="pageVisits.length === 0"
      class="text-sm text-muted-foreground"
    >
      {{ t('globals.messages.noResults') }}
    </div>
    <div
      v-for="(page, idx) in pageVisits"
      :key="idx"
      class="flex items-center justify-between gap-2 py-1.5"
    >
      <a
        :href="page.url"
        target="_blank"
        rel="noopener"
        class="text-xs truncate hover:underline"
        :title="page.url"
      >
        {{ page.title || page.url }}
      </a>
      <span v-if="page.time" class="text-[11px] text-muted-foreground flex-shrink-0">
        {{ formatDate(page.time) }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
import api from '../../../api'

const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()

const pageVisits = computed(() => conversation.value?.contact?.page_visits || [])

function formatDate (dateStr) {
  try {
    return format(new Date(dateStr), 'd MMM yyyy')
  } catch {
    return ''
  }
}

watch(
  () => conversation.value?.uuid,
  async (uuid) => {
    if (uuid && conversation.value?.inbox_channel === 'livechat') {
      try {
        const resp = await api.getContactPageVisits(uuid)
        if (resp.data?.data) {
          conversationStore.updateConversationProp({
            contact_id: conversation.value?.contact_id,
            prop: 'contact.page_visits',
            value: resp.data.data
          })
        }
      } catch {
        // Page visits are optional.
      }
    }
  },
  { immediate: true }
)
</script>
