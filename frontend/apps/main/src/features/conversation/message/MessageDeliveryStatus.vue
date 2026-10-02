<template>
  <span
    v-if="message.status === 'pending'"
    class="flex items-center gap-1 text-xs text-muted-foreground"
    role="status"
  >
    <Loader2 class="h-3.5 w-3.5 animate-spin" />{{ t('conversation.sendingMessage') }}
  </span>
  <span v-else-if="message.status === 'sent'" class="flex items-center gap-1 text-xs text-success">
    <Check class="h-3.5 w-3.5" />{{ t('globals.terms.sent') }}
  </span>
  <div
    v-else-if="message.status === 'failed'"
    class="flex flex-wrap items-center gap-2 text-xs text-destructive"
    role="status"
  >
    <span>{{
      t(uncertain ? 'conversation.deliveryUncertain' : 'conversation.deliveryFailed')
    }}</span>
    <Button
      v-if="canRetry"
      type="button"
      size="sm"
      variant="outline"
      :disabled="retrying"
      @click="requestRetry"
    >
      {{ t('conversation.retryDelivery') }}
    </Button>
  </div>
  <AlertDialog :open="confirmRetry" @update:open="confirmRetry = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('conversation.retryUncertainTitle') }}</AlertDialogTitle>
        <AlertDialogDescription>{{
          t('conversation.retryUncertainDescription')
        }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="retry">{{ t('conversation.retryDelivery') }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed, ref } from 'vue'
import { Check, Loader2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction
} from '@shared-ui/components/ui/alert-dialog'
import { useUserStore } from '@main/stores/user'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http'
import api from '@main/api'

const props = defineProps({ message: { type: Object, required: true } })
const { t } = useI18n()
const user = useUserStore()
const conversations = useConversationStore()
const emitter = useEmitter()
const uncertain = computed(() => props.message.meta?.delivery_uncertain === true)
const canRetry = computed(() => props.message.sender_id === user.userID)
const confirmRetry = ref(false)
const retrying = ref(false)
let target = null
function requestRetry() {
  target = { uuid: props.message.uuid, conversationUUID: props.message.conversation_uuid }
  if (uncertain.value) confirmRetry.value = true
  else retry()
}
async function retry() {
  if (!target || retrying.value) return
  const { uuid, conversationUUID } = target
  retrying.value = true
  try {
    await api.retryMessage(conversationUUID, uuid)
    await conversations.fetchMessage(conversationUUID, uuid)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    retrying.value = false
    confirmRetry.value = false
  }
}
</script>
