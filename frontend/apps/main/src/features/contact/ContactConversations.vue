<template>
  <div class="w-full">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-base font-semibold">
        {{ t('contact.tickets') }}
        <span v-if="!loading" class="ml-1 text-xs font-normal text-muted-foreground tabular-nums">
          {{ conversations.length }}
        </span>
      </h3>
    </div>

    <div v-if="loading" class="space-y-2">
      <Skeleton v-for="i in 4" :key="i" class="h-10 w-full" />
    </div>

    <p v-else-if="!conversations.length" class="text-sm text-muted-foreground py-6">
      {{ t('contact.noTickets') }}
    </p>

    <div v-else class="rounded-md border overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b bg-muted/40 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            <th class="px-3 py-2 w-24">{{ t('globals.terms.status') }}</th>
            <th class="px-3 py-2 w-20">{{ t('zendesk.id') }}</th>
            <th class="px-3 py-2">{{ t('globals.terms.subject') }}</th>
            <th class="px-3 py-2 w-28">{{ t('globals.terms.lastMessageAt') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="conversation in conversations"
            :key="conversation.uuid"
            class="border-b last:border-0 cursor-pointer hover:bg-muted/40"
            @click="openConversation(conversation.uuid)"
          >
            <td class="px-3 py-2">
              <span class="zendesk-status-badge" :class="categoryClass(conversation.status)">
                {{ conversation.status || '—' }}
              </span>
            </td>
            <td class="px-3 py-2 text-muted-foreground tabular-nums whitespace-nowrap">
              <span v-if="conversation.reference_number">#{{ conversation.reference_number }}</span>
            </td>
            <td class="px-3 py-2 min-w-0">
              <p class="truncate text-[#1f73b7]">
                {{ conversation.subject || t('zendesk.noSubject') }}
              </p>
              <p v-if="conversation.last_message" class="truncate text-xs text-muted-foreground">
                {{ conversation.last_message }}
              </p>
            </td>
            <td class="px-3 py-2 text-muted-foreground whitespace-nowrap tabular-nums">
              {{
                conversation.last_message_at
                  ? formatShortDate(conversation.last_message_at)
                  : conversation.created_at
                    ? formatShortDate(conversation.created_at)
                    : ''
              }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { formatShortDate } from '@shared-ui/utils/datetime.js'
import { useStatusCategory } from '@main/composables/useStatusCategory'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

const props = defineProps({
  contactId: { type: [String, Number], required: true }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const { categoryClass } = useStatusCategory()
const conversations = ref([])
const loading = ref(false)

const fetchConversations = async () => {
  loading.value = true
  try {
    const { data } = await api.getContactConversations(props.contactId)
    conversations.value = data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}

const openConversation = (uuid) => {
  router.push({
    name: 'inbox-conversation',
    params: { type: 'all', uuid }
  })
}

watch(() => props.contactId, fetchConversations)
onMounted(fetchConversations)
</script>
