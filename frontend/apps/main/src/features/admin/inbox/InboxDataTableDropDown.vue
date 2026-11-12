<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem @click="editInbox(props.inbox.id)">
        <Pencil class="mr-2 h-4 w-4" />
        {{ i18n._('Edit') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-if="props.inbox.enabled">
        <PowerOff class="mr-2 h-4 w-4" />
        {{ i18n._('Disable') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-else>
        <Power class="mr-2 h-4 w-4" />
        {{ i18n._('Enable') }}
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
          {{ i18n._('This action cannot be undone. This will permanently delete this inbox.') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDelete">{{
          i18n._('Delete')
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
