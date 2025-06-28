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
        <RouterLink :to="{ name: 'edit-sso', params: { id: props.role.id } }">
          {{ $t('globals.messages.edit') }}
        </RouterLink>
      </DropdownMenuItem>
      <DropdownMenuItem @click="() => (alertOpen = true)">{{
        $t('globals.messages.delete')
      }}</DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('globals.messages.deletionConfirmation', { name: $t('globals.terms.sso') }) }}
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
import api from '../../../api'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'

const emit = useEmitter()
const alertOpen = ref(false)

const props = defineProps({
  role: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

async function handleDelete() {
  await api.deleteOIDC(props.role.id)
  alertOpen.value = false
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'oidc'
  })
}
</script>
