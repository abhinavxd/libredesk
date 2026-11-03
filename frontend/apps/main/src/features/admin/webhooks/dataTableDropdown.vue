<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem :as-child="true">
        <RouterLink :to="{ name: 'edit-webhook', params: { id: props.webhook.id } }">
          <Pencil class="mr-2 h-4 w-4" />
          {{ $t('globals.messages.edit') }}
        </RouterLink>
      </DropdownMenuItem>
      <DropdownMenuItem @click="handleToggle">
        <component :is="props.webhook.is_active ? PowerOff : Power" class="mr-2 h-4 w-4" />
        {{
          props.webhook.is_active ? $t('globals.messages.disable') : $t('globals.messages.enable')
        }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="handleTest">
        <Send class="mr-2 h-4 w-4" />
        {{ $t('webhook.sendTest') }}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        @click="() => (alertOpen = true)"
        class="text-destructive focus:text-destructive"
      >
        <Trash class="mr-2 h-4 w-4" />
        {{ $t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('confirm.deleteWebhook') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDelete">
          {{ $t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import DataTableActionTrigger from '@main/components/datatable/DataTableActionTrigger.vue'
import { ref } from 'vue'
import { Pencil, Power, PowerOff, Send, Trash } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator
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
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const emit = useEmitter()
const { t } = useI18n()
const alertOpen = ref(false)

const props = defineProps({
  webhook: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      name: '',
      is_active: false
    })
  }
})

async function handleDelete() {
  try {
    await api.deleteWebhook(props.webhook.id)
    alertOpen.value = false
    emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
      model: 'webhook'
    })
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

async function handleToggle() {
  try {
    await api.toggleWebhook(props.webhook.id)
    emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
      model: 'webhook'
    })
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

async function handleTest() {
  try {
    await api.testWebhook(props.webhook.id)
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: t('webhook.sentSuccessfully')
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
