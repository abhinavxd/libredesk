<template>
  <div>
    <div v-if="pageVisits.length === 0" class="text-center text-sm text-muted-foreground py-4">
      {{ t('globals.messages.noResultsFound') }}
    </div>
    <div v-else class="space-y-1">
      <a
        v-for="(page, idx) in pageVisits"
        :key="idx"
        :href="page.url"
        target="_blank"
        rel="noopener noreferrer"
        class="block p-2 rounded hover:bg-muted"
      >
        <div class="flex items-start justify-between gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <span class="sidebar-value font-medium truncate">
                {{ page.title || page.url }}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              {{ page.url }}
            </TooltipContent>
          </Tooltip>
          <Tooltip v-if="page.time">
            <TooltipTrigger asChild>
              <span class="sidebar-label flex-shrink-0">
                {{ getRelativeTime(new Date(page.time)) }}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              {{ formatFullTimestamp(new Date(page.time)) }}
            </TooltipContent>
          </Tooltip>
        </div>
        <span v-if="page.title && page.title !== page.url" class="sidebar-label truncate block">
          {{ page.url }}
        </span>
      </a>
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useI18n } from 'vue-i18n'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { getRelativeTime, formatFullTimestamp } from '@shared-ui/utils/datetime.js'
import api from '../../../api'

const conversationStore = useConversationStore()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()

const pageVisits = computed(() => {
  const visits = conversation.value?.contact?.page_visits || []
  const seen = new Set()
  return visits.filter((v) => {
    const key = `${v.url}|${v.time}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
})

watch(
  () => conversation.value?.uuid,
  async (uuid) => {
    if (uuid && conversation.value?.inbox_channel === 'livechat') {
      try {
        const resp = await api.getContactPageVisits(uuid)
        if (resp.data?.data) {
          conversationStore.mergeContactUpdate({
            contact_id: conversation.value?.contact_id,
            page_visits: resp.data.data
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
