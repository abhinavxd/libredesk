<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only"></span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editCustomAttribute">
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
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>{{
          $t('globals.messages.deletionConfirmation', {
            name: $t('globals.terms.customAttribute').toLowerCase()
          })
        }}</AlertDialogDescription>
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
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '../../../utils/http'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import api from '../../../api'

const alertOpen = ref(false)
const emit = useEmitter()

const props = defineProps({
  customAttribute: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

async function handleDelete() {
  try {
    await api.deleteCustomAttribute(props.customAttribute.id)
    alertOpen.value = false
    emitRefreshCustomAttributeList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const emitRefreshCustomAttributeList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'custom-attributes'
  })
}

const editCustomAttribute = () => {
  emit.emit(EMITTER_EVENTS.EDIT_MODEL, {
    model: 'custom-attributes',
    data: props.customAttribute
  })
}
</script>
