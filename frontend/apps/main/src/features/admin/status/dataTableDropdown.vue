<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        class="w-8 h-8 p-0"
        v-if="!CONVERSATION_DEFAULT_STATUSES_LIST.includes(props.status.name)"
      >
        <span class="sr-only"></span>
        <MoreVertical class="w-4 h-4" />
      </Button>
      <div v-else class="w-8 h-8 p-0 invisible"></div>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editStatus">
        {{ $t('globals.messages.edit') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="() => (alertOpen = true)">
        {{ $t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle> {{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t('status.deletionConfirmation')
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">{{
          $t('globals.messages.delete')
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
import { Button } from '@shared-ui/components/ui/button/index.js'
import { CONVERSATION_DEFAULT_STATUSES_LIST } from '@/constants/conversation.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import api from '../../../api/index.js'

const alertOpen = ref(false)
const emit = useEmitter()

const props = defineProps({
  status: {
    type: Object,
    required: true
  }
})

const editStatus = () => {
  emit.emit(EMITTER_EVENTS.EDIT_MODEL, {
    model: 'status',
    data: props.status
  })
}

const handleDelete = async () => {
  try {
    await api.deleteStatus(props.status.id)
    alertOpen.value = false
    emit.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'status' })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
