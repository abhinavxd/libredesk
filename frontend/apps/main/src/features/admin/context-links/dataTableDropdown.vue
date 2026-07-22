<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only"></span>
        <MoreVertical class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem :as-child="true">
        <RouterLink :to="{ name: 'edit-context-link', params: { id: props.contextLink.id } }">
          {{ $t('globals.messages.edit') }}
        </RouterLink>
      </DropdownMenuItem>
      <DropdownMenuItem @click="handleToggle">
        {{
          props.contextLink.is_active
            ? $t('globals.messages.disable')
            : $t('globals.messages.enable')
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
          {{ $t('confirm.deleteContextLink') }}
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
import { MoreVertical } from 'lucide-vue-next'
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
import { Button } from '@shared-ui/components/ui/button'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const emit = useEmitter()
const { t } = useI18n()
const alertOpen = ref(false)

const props = defineProps({
  contextLink: {
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
    await api.deleteContextLink(props.contextLink.id)
    alertOpen.value = false
    emit.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'context-link' })
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
    await api.toggleContextLink(props.contextLink.id)
    emit.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'context-link' })
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
</script>
