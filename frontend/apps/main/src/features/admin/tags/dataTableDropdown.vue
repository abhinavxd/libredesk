<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem @click="editTag">
        <Pencil class="mr-2 h-4 w-4" />
        {{ i18n._('Edit') }}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        @click="() => (alertOpen = true)"
        class="text-destructive focus:text-destructive"
      >
        <Trash class="mr-2 h-4 w-4" />
        {{ i18n._('Delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ i18n._('Are you absolutely sure?') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ i18n._('Are you sure you want to delete this tag? This will also remove it from all conversations') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="deleteTag">{{
          i18n._('Delete')
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
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from '@shared-ui/i18n'
import api from '@/api/index.js'

const i18n = useI18n()
const alertOpen = ref(false)
const emitter = useEmitter()

const props = defineProps({
  tag: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      name: ''
    })
  }
})

const editTag = () => {
  emitter.emit(EMITTER_EVENTS.EDIT_MODEL, {
    model: 'tags',
    data: props.tag
  })
}

const deleteTag = async () => {
  await api.deleteTag(props.tag.id)
  alertOpen.value = false
  emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'tags' })
}
</script>
