<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="h-12 flex-shrink-0 px-2 border-b flex items-center justify-between gap-2">
      <div class="flex items-center gap-1 min-w-0">
        <Button
          v-if="isMobile"
          variant="ghost"
          class="w-11 h-11 md:w-8 md:h-8 p-0 shrink-0 -ml-2 md:-ml-1"
          :aria-label="t('globals.messages.back')"
          @click="goBackToList"
        >
          <ChevronLeft class="w-4 h-4" />
        </Button>
        <span class="truncate">{{ conversationStore.currentContactName }}</span>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <Button
          v-if="isMobile"
          variant="ghost"
          class="w-11 h-11 md:w-8 md:h-8 p-0"
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
              class="flex items-center space-x-1 cursor-pointer bg-primary px-3 py-3 md:px-2 md:py-1 rounded-md text-sm"
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
            <Button variant="ghost" class="w-11 h-11 md:w-8 md:h-8 p-0">
              <MoreHorizontal class="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem @click="downloadTranscript">
              {{ t('conversation.downloadTranscript') }}
            </DropdownMenuItem>
            <DropdownMenuItem
              v-if="userStore.can(perms.MESSAGES_WRITE_PRIVATE)"
              :disabled="isSummarizing"
              @click="summarize"
            >
              {{ t('conversation.summarize') }}
            </DropdownMenuItem>
            <template v-if="userStore.can(perms.CONVERSATIONS_DELETE)">
              <DropdownMenuSeparator />
              <DropdownMenuItem
                class="text-destructive focus:text-destructive"
                @click="openDeleteDialog"
              >
                {{ t('conversation.delete') }}
              </DropdownMenuItem>
            </template>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>

    <AlertDialog :open="deleteDialogOpen" @update:open="deleteDialogOpen = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ t('conversation.delete') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ t('conversation.deleteConfirmation') }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div v-if="isEmailConversation" class="flex items-start space-x-3">
          <Checkbox
            id="delete-conversation-purge-mail"
            :checked="purgeMail"
            @update:checked="(value) => (purgeMail = value === true)"
          />
          <label for="delete-conversation-purge-mail" class="text-sm leading-tight">
            {{ t('conversation.deletePurgeMail') }}
            <span class="block text-muted-foreground">
              {{ t('conversation.deletePurgeMailHint') }}
            </span>
          </label>
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" @click="confirmDelete">
            {{ t('globals.messages.delete') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Messages & reply box -->
    <div class="flex flex-col flex-grow overflow-hidden">
      <MessageList class="flex-1 overflow-y-auto" />
      <ReplyBox v-if="canCompose" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Clock, MoreHorizontal, ChevronLeft, PanelRight } from 'lucide-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { useIsMobile } from '@shared-ui/composables'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { formatMessageTimestamp } from '@shared-ui/utils/datetime.js'
import { Button } from '@shared-ui/components/ui/button'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import { EMITTER_EVENTS, CONVERSATION_ACTIONS } from '@main/constants/emitterEvents.js'
import { useCommandPalette } from '@/features/command/useCommandPalette'
import { SNOOZE_COMMAND } from '@/features/command/providers/useConversationCommands'
import { CONVERSATION_DEFAULT_STATUSES } from '@main/constants/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { downloadBlobResponse, parseBlobError } from '@shared-ui/utils/file'
import api from '@main/api'
import { permissions as perms } from '@main/constants/permissions.js'
import { deletionToast } from '@main/utils/conversation-delete'
const conversationStore = useConversationStore()
const userStore = useUserStore()
const emitter = useEmitter()
const palette = useCommandPalette()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const isMobile = useIsMobile()
const canCompose = computed(
  () => userStore.can(perms.MESSAGES_WRITE) || userStore.can(perms.MESSAGES_WRITE_PRIVATE)
)

// Each detail route is `<list route name>-conversation`.
const goBackToList = () => {
  const listName = String(route.name).replace(/-conversation$/, '')
  const { uuid, ...params } = route.params
  const target = router.resolve({ name: listName, params })
  if (window.history.state?.back?.split('?')[0] === target.path) router.back()
  else router.push(target)
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
    downloadBlobResponse(response, `transcript-${conversation.reference_number}.txt`)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(await parseBlobError(error)).message
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

const deleteDialogOpen = ref(false)
const isDeleting = ref(false)
const purgeMail = ref(true)
const isEmailConversation = computed(() => conversationStore.current?.inbox_channel === 'email')

const openDeleteDialog = () => {
  purgeMail.value = true
  deleteDialogOpen.value = true
}

const confirmDelete = async () => {
  const conversation = conversationStore.current
  if (!conversation?.uuid || isDeleting.value) return
  try {
    isDeleting.value = true
    const result = await conversationStore.deleteConversation(conversation.uuid, {
      purgeMail: isEmailConversation.value && purgeMail.value
    })
    const toast = deletionToast(result?.unpurged_message_ids)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: toast.variant,
      description:
        toast.count > 0 ? t(toast.key, toast.count, { count: toast.count }) : t(toast.key)
    })
    deleteDialogOpen.value = false
    goBackToList()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isDeleting.value = false
  }
}

const handleUpdateStatus = (status) => {
  if (status === CONVERSATION_DEFAULT_STATUSES.SNOOZED) {
    palette.openPalette({ parent: SNOOZE_COMMAND })
    return
  }
  conversationStore.updateStatus(status)
}

const paletteActions = {
  [CONVERSATION_ACTIONS.DOWNLOAD_TRANSCRIPT]: downloadTranscript,
  [CONVERSATION_ACTIONS.SUMMARIZE]: summarize
}
const onPaletteAction = (action) => paletteActions[action]?.()

onMounted(() => emitter.on(EMITTER_EVENTS.CONVERSATION_ACTION, onPaletteAction))
onUnmounted(() => emitter.off(EMITTER_EVENTS.CONVERSATION_ACTION, onPaletteAction))
</script>
