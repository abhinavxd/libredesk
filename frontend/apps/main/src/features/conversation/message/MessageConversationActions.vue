<template>
  <div class="flex shrink-0 items-center gap-2">
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
import { useI18n } from 'vue-i18n'
import { MoreHorizontal } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS, CONVERSATION_ACTIONS } from '@main/constants/emitterEvents'

const { t } = useI18n()
const emitter = useEmitter()
const downloadTranscript = () =>
  emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.DOWNLOAD_TRANSCRIPT)
</script>
