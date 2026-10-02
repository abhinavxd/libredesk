<template>
  <div class="flex shrink-0 items-center gap-2">
    <Tooltip v-if="snoozedUntil">
      <TooltipTrigger as-child>
        <span class="flex items-center gap-1 text-xs text-muted-foreground whitespace-nowrap">
          <Clock :size="12" />{{ snoozedUntil }}
        </span>
      </TooltipTrigger>
      <TooltipContent>{{ t('conversation.snoozedUntil', { time: snoozedUntil }) }}</TooltipContent>
    </Tooltip>
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button variant="ghost" size="icon" :aria-label="t('globals.terms.action', 2)">
          <MoreHorizontal class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem @click="downloadTranscript">
          {{ t('conversation.downloadTranscript') }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Clock, MoreHorizontal } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS, CONVERSATION_ACTIONS } from '@main/constants/emitterEvents'
import { CONVERSATION_DEFAULT_STATUSES } from '@main/constants/conversation'
import { formatMessageTimestamp } from '@shared-ui/utils/datetime'

const { t } = useI18n()
const store = useConversationStore()
const emitter = useEmitter()
const snoozedUntil = computed(() =>
  store.current?.status === CONVERSATION_DEFAULT_STATUSES.SNOOZED && store.current.snoozed_until
    ? formatMessageTimestamp(store.current.snoozed_until)
    : ''
)
const downloadTranscript = () =>
  emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.DOWNLOAD_TRANSCRIPT)
</script>
