<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only"></span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem :as-child="true">
        <RouterLink :to="{ name: 'edit-webhook', params: { id: props.webhook.id } }">
          {{ $t('globals.messages.edit') }}
        </RouterLink>
      </DropdownMenuItem>
      <DropdownMenuItem @click="handleToggle">
        {{
          props.webhook.is_active ? $t('globals.messages.disable') : $t('globals.messages.enable')
        }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="handleTest">
        {{
          $t('globals.messages.send', {
            name: $t('globals.terms.test').toLowerCase()
          })
        }}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem @click="() => (alertOpen = true)" class="text-destructive">
        {{ $t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('globals.messages.deletionConfirmation', { name: $t('globals.terms.webhook') }) }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">
          {{ $t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
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
      title: 'Success',
      description: t('globals.messages.deletedSuccessfully', {
        name: t('globals.terms.webhook')
      })
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
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.webhook')
      })
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
      description: t('globals.messages.sentSuccessfully', {
        name: t('globals.terms.webhook')
      })
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
