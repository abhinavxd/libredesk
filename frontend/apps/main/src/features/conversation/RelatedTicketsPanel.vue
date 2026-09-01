<template>
  <div v-if="canWrite || related.length" class="space-y-2">
    <p class="text-xs font-medium text-muted-foreground">{{ t('conversation.related.title') }}</p>

    <div v-if="related.length" class="space-y-1">
      <button
        v-for="item in related"
        :key="item.uuid"
        type="button"
        class="w-full text-left rounded-md border px-2 py-1.5 text-sm hover:bg-muted"
        @click="open(item.uuid)"
      >
        <span class="font-medium tabular-nums">#{{ item.reference_number }}</span>
        <span class="ml-1.5 text-xs text-muted-foreground">{{ relationLabel(item) }}</span>
        <p class="text-xs text-muted-foreground truncate">{{ item.subject || item.last_message || '' }}</p>
      </button>
    </div>
    <p v-else class="text-xs text-muted-foreground">{{ t('conversation.related.empty') }}</p>

    <div v-if="canWrite" class="flex flex-wrap gap-1.5">
      <Button size="sm" variant="outline" :disabled="busy" @click="create('child')">
        {{ t('conversation.related.createChild') }}
      </Button>
      <Button size="sm" variant="outline" :disabled="busy" @click="create('follow_up')">
        {{ t('conversation.related.createFollowUp') }}
      </Button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useTicketActions } from '@main/composables/useTicketActions'
import { permissions as perms } from '@main/constants/permissions'

const { t } = useI18n()
const conversationStore = useConversationStore()
const userStore = useUserStore()
const actions = useTicketActions()
const busy = ref(false)

const canWrite = computed(() => userStore.can(perms.CONVERSATIONS_WRITE))
const related = computed(() => conversationStore.current?.related_conversations || [])

const relationLabel = (item) => {
  if (item.relation === 'parent') return t('conversation.related.parent')
  if (item.relation === 'follow_up' || item.origin === 'follow_up') {
    return t('conversation.related.followUp')
  }
  return t('conversation.related.child')
}

const open = (uuid) => actions.openTicket(uuid)

const create = async (origin) => {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return
  busy.value = true
  try {
    await actions.createLinked(uuid, origin)
  } finally {
    busy.value = false
  }
}
</script>
