<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem @click="editSharedView">
        <Pencil class="mr-2 h-4 w-4" />
        {{ $t('globals.messages.edit') }}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        @click="() => (isDeleteOpen = true)"
        class="text-destructive focus:text-destructive"
      >
        <Trash class="mr-2 h-4 w-4" />
        {{ $t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="isDeleteOpen" @update:open="isDeleteOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('confirm.deleteSharedView') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDelete">{{
          $t('globals.messages.delete')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import DataTableActionTrigger from '@main/components/datatable/DataTableActionTrigger.vue'
import { ref } from 'vue'
import { Pencil, Trash } from 'lucide-vue-next'
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
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useRouter } from 'vue-router'
import { useSharedViewStore } from '@/stores/sharedView'
import api from '@/api/index.js'

const router = useRouter()
const emit = useEmitter()
const sharedViewStore = useSharedViewStore()
const isDeleteOpen = ref(false)

const props = defineProps({
  sharedView: {
    type: Object,
    required: true
  }
})

const handleDelete = async () => {
  await api.deleteSharedView(props.sharedView.id)

  await sharedViewStore.refresh()

  isDeleteOpen.value = false
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'shared-views' })
}

const editSharedView = () => {
  router.push({ name: 'edit-shared-view', params: { id: props.sharedView.id } })
}
</script>
