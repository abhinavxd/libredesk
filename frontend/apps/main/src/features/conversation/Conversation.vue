<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="h-12 flex-shrink-0 px-2 border-b flex items-center justify-between gap-2">
      <div class="flex items-center gap-1 min-w-0">
        <!-- Mobile is single-pane, so the list is only reachable by going
             back. On desktop the list is always visible alongside. -->
        <Button
          v-if="isMobile"
          variant="ghost"
          class="w-8 h-8 p-0 shrink-0 -ml-1"
          :aria-label="t('globals.messages.back')"
          @click="goBackToList"
        >
          <ChevronLeft class="w-5 h-5" />
        </Button>
        <span class="truncate">{{ conversationStore.currentContactName }}</span>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <!-- Opens the contact sidebar, which is a Sheet on mobile. Desktop has
             its own re-open tab on the panel edge. -->
        <Button
          v-if="isMobile"
          variant="ghost"
          class="w-8 h-8 p-0"
          :aria-label="t('globals.terms.contact')"
          @click="emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)"
        >
          <PanelRight class="w-4 h-4" />
        </Button>
        <Tooltip v-if="isSnoozed && snoozedUntilLabel">
          <TooltipTrigger as-child>
            <span class="flex items-center gap-1 text-xs text-muted-foreground whitespace-nowrap">
              <Clock :size="12" />
              {{ snoozedUntilLabel }}
            </span>
          </TooltipTrigger>
          <TooltipContent>
            {{ t('conversation.snoozedUntil', { time: snoozedUntilLabel }) }}
          </TooltipContent>
        </Tooltip>
        <DropdownMenu>
          <DropdownMenuTrigger>
            <div
              v-if="conversationStore.current?.status"
              class="flex items-center space-x-1 cursor-pointer bg-primary px-2 py-1 rounded-md text-sm"
            >
              <span class="text-primary-foreground font-medium inline-block">
                {{ conversationStore.current?.status }}
              </span>
            </div>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              v-for="status in conversationStore.statusOptions"
              :key="status.value"
              @click="handleUpdateStatus(status.label)"
            >
              {{ status.label }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" class="w-8 h-8 p-0">
              <MoreHorizontal class="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem @click="downloadTranscript">
              {{ t('conversation.downloadTranscript') }}
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="userStore.can('messages:write')"
              :disabled="isSummarizing"
              @click="summarize"
            >
              {{ t('conversation.summarize') }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <!-- Messages & reply box -->
    <div class="flex flex-col flex-grow overflow-hidden">
      <MessageList class="flex-1 overflow-y-auto" />
      <ReplyBox />
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useConversationStore } from '../../stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Clock, MoreHorizontal, ChevronLeft, PanelRight } from 'lucide-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { useIsMobile } from '@main/composables/useIsMobile'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { formatMessageTimestamp } from '@shared-ui/utils/datetime.js'
import { Button } from '@shared-ui/components/ui/button'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { CONVERSATION_DEFAULT_STATUSES } from '../../constants/conversation'
import { useEmitter } from '../../composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
const conversationStore = useConversationStore()
const userStore = useUserStore()
const emitter = useEmitter()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const isMobile = useIsMobile()

// Each detail route is `<list route name>-conversation`, so the parent list
// route is the same name without that suffix, with the same params.
const goBackToList = () => {
  const listName = String(route.name).replace(/-conversation$/, '')
  router.push({ name: listName, params: route.params })
}

const isSnoozed = computed(
  () => conversationStore.current?.status === CONVERSATION_DEFAULT_STATUSES.SNOOZED
)
const snoozedUntilLabel = computed(() =>
  conversationStore.current?.snoozed_until
    ? formatMessageTimestamp(conversationStore.current.snoozed_until)
    : ''
)

const downloadTranscript = async () => {
  const conversation = conversationStore.current
  if (!conversation) return
  try {
    const response = await api.getConversationTranscript(conversation.uuid)
    const url = URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = url
    link.download = `transcript-${conversation.reference_number}.txt`
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => URL.revokeObjectURL(url), 0)
  } catch (error) {
    if (error.response?.data instanceof Blob) {
      try {
        error.response.data = JSON.parse(await error.response.data.text())
      } catch {
        // keep the original blob, handleHTTPError falls back to a generic message
      }
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const isSummarizing = ref(false)

const summarize = async () => {
  const conversation = conversationStore.current
  if (!conversation || isSummarizing.value) return
  try {
    isSummarizing.value = true
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'info',
      description: t('conversation.summarizing')
    })
    await api.aiSummarizeConversation({ conversation_uuid: conversation.uuid })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('conversation.summarizeAdded')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSummarizing.value = false
  }
}

const handleUpdateStatus = (status) => {
  if (status === CONVERSATION_DEFAULT_STATUSES.SNOOZED) {
    emitter.emit(EMITTER_EVENTS.SET_NESTED_COMMAND, {
      command: 'snooze',
      open: true
    })
    return
  }
  conversationStore.updateStatus(status)
}
</script>
