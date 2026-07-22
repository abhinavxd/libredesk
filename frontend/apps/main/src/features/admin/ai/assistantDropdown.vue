<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">{{ t('globals.terms.openMenu') }}</span>
        <MoreVertical class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editAssistant">
        {{ t('globals.messages.edit') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="() => (alertOpen = true)">
        {{ t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ t('admin.ai.assistant.deleteConfirmation') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="deleteAssistant">{{
          t('globals.messages.delete')
        }}</AlertDialogAction>
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
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu/index.js'
import { Button } from '@shared-ui/components/ui/button/index.js'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog/index.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const { t } = useI18n()
const alertOpen = ref(false)
const emitter = useEmitter()

const props = defineProps({
  assistant: { type: Object, required: true }
})

const editAssistant = () => {
  emitter.emit(EMITTER_EVENTS.EDIT_MODEL, { model: 'ai_assistants', data: props.assistant })
}

const deleteAssistant = async () => {
  try {
    await api.deleteAIAssistant(props.assistant.id)
    alertOpen.value = false
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'ai_assistants' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
