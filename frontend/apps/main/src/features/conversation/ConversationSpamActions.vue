<template>
  <div v-if="canAct">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button
          variant="outline"
          size="sm"
          :class="compact ? 'h-7 text-xs shrink-0 px-2' : 'w-full justify-start'"
          :disabled="busy"
        >
          <Ban class="size-3.5 mr-1.5" />
          {{ t('conversation.spam.markAsSpam') }}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent :align="compact ? 'end' : 'start'" class="w-64">
        <DropdownMenuItem
          v-if="canBlock"
          class="cursor-pointer"
          @click="confirm = 'block'"
        >
          <Ban class="mr-2 size-4" />
          {{
            isBlocked
              ? t('globals.messages.unblock')
              : t('conversation.spam.blockSender')
          }}
        </DropdownMenuItem>
        <DropdownMenuItem
          v-if="canDeleteTicket"
          class="cursor-pointer"
          @click="confirm = 'delete'"
        >
          <Trash2 class="mr-2 size-4" />
          {{ t('conversation.spam.deleteTicket') }}
        </DropdownMenuItem>
        <DropdownMenuSeparator v-if="canBlock && canDeleteTicket && !isBlocked" />
        <DropdownMenuItem
          v-if="canBlock && canDeleteTicket && !isBlocked"
          class="cursor-pointer text-destructive"
          @click="confirm = 'spam'"
        >
          <ShieldOff class="mr-2 size-4" />
          {{ t('conversation.spam.blockAndDelete') }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <AlertDialog :open="Boolean(confirm)" @update:open="onDialogOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ dialogTitle }}</AlertDialogTitle>
          <AlertDialogDescription>{{ dialogDescription }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" :disabled="busy" @click="runAction">
            {{ dialogAction }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Ban, ShieldOff, Trash2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
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
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useInboxViewContext } from '@main/composables/useInboxViewContext'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

defineProps({
  compact: { type: Boolean, default: false }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const userStore = useUserStore()
const { listRoute } = useInboxViewContext()

const confirm = ref('')
const busy = ref(false)

const contact = computed(() => conversationStore.current?.contact)
const contactId = computed(() => conversationStore.current?.contact_id || contact.value?.id)
const isBlocked = computed(() => contact.value && contact.value.enabled === false)
const canBlock = computed(() => userStore.can('contacts:block') && Boolean(contactId.value))
const canDeleteTicket = computed(() => userStore.can('conversations:write'))
const canAct = computed(() => canBlock.value || canDeleteTicket.value)

const dialogTitle = computed(() => {
  if (confirm.value === 'block') {
    return isBlocked.value ? t('contact.unblockContact') : t('conversation.spam.blockSender')
  }
  if (confirm.value === 'delete') return t('conversation.spam.deleteTicket')
  return t('conversation.spam.markAsSpam')
})

const dialogDescription = computed(() => {
  if (confirm.value === 'block') {
    return isBlocked.value ? t('contact.unblockConfirm') : t('contact.blockConfirm')
  }
  if (confirm.value === 'delete') return t('conversation.spam.deleteTicketConfirm')
  return t('conversation.spam.blockAndDeleteConfirm')
})

const dialogAction = computed(() => {
  if (confirm.value === 'block') {
    return isBlocked.value ? t('globals.messages.unblock') : t('conversation.spam.blockSender')
  }
  if (confirm.value === 'delete') return t('conversation.spam.deleteTicket')
  return t('conversation.spam.blockAndDelete')
})

const onDialogOpen = (open) => {
  if (!open) confirm.value = ''
}

const toast = (description) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description })
}

const toastError = (err) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(err).message
  })
}

const leaveTicket = () => {
  router.push(listRoute.value)
}

const runAction = async () => {
  const action = confirm.value
  confirm.value = ''
  busy.value = true
  try {
    if (action === 'block') {
      const enabled = isBlocked.value
      const { data } = await api.blockContact(contactId.value, { enabled })
      conversationStore.mergeContactUpdate({
        contact_id: contactId.value,
        enabled: data.data.enabled
      })
      toast(enabled ? t('contact.unblockedSuccessfully') : t('contact.blockedSuccessfully'))
      return
    }
    if (action === 'delete') {
      await conversationStore.deleteCurrentConversation()
      toast(t('conversation.spam.ticketDeleted'))
      leaveTicket()
      return
    }
    if (action === 'spam') {
      await api.blockContact(contactId.value, { enabled: false })
      await conversationStore.deleteCurrentConversation()
      toast(t('conversation.spam.blockedAndDeleted'))
      leaveTicket()
    }
  } catch (err) {
    toastError(err)
  } finally {
    busy.value = false
  }
}
</script>
