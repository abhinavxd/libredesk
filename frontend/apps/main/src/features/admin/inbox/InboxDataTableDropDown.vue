<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem @click="editInbox(props.inbox.id)">
        <Pencil class="mr-2 h-4 w-4" />
        {{ $t('globals.messages.edit') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-if="props.inbox.enabled">
        <PowerOff class="mr-2 h-4 w-4" />
        {{ $t('globals.messages.disable') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-else>
        <Power class="mr-2 h-4 w-4" />
        {{ $t('globals.messages.enable') }}
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
          {{ $t('confirm.deleteInbox') }}
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
import { Pencil, Power, PowerOff, Trash } from 'lucide-vue-next'
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

const alertOpen = ref(false)
const props = defineProps({
  inbox: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['editInbox', 'deleteInbox', 'toggleInbox'])

function editInbox(id) {
  emit('editInbox', id)
}

function handleDelete() {
  emit('deleteInbox', props.inbox.id)
  alertOpen.value = false
}

function toggleInbox(id) {
  emit('toggleInbox', id)
}
</script>
