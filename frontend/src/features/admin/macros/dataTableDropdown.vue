<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only"></span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editMacro">{{ $t('globals.messages.edit') }}</DropdownMenuItem>
      <DropdownMenuItem @click="() => (isDeleteOpen = true)">
        {{ $t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="isDeleteOpen" @update:open="isDeleteOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('globals.messages.deletionConfirmation', { name: $t('globals.terms.macro') }) }}
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
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
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
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useRouter } from 'vue-router'
import { useMacroStore } from '@/stores/macro'
import api from '@/api/index.js'

const router = useRouter()
const emit = useEmitter()
const macroStore = useMacroStore()
const isDeleteOpen = ref(false)

const props = defineProps({
  macro: {
    type: Object,
    required: true
  }
})

const handleDelete = async () => {
  await api.deleteMacro(props.macro.id)
  
  await macroStore.loadMacros(true)
  
  isDeleteOpen.value = false
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'macros' })
}

const editMacro = () => {
  router.push({ path: `/admin/conversations/macros/${props.macro.id}/edit` })
}
</script>