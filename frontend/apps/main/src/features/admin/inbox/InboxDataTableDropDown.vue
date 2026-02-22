<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only"></span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editInbox(props.inbox.id)">{{
        $t('globals.messages.edit')
      }}</DropdownMenuItem>
      <DropdownMenuItem @click="() => (alertOpen = true)">{{
        $t('globals.messages.delete')
      }}</DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-if="props.inbox.enabled">
        {{ $t('globals.messages.disable') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-else>{{
        $t('globals.messages.enable')
      }}</DropdownMenuItem>
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
