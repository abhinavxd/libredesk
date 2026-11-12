<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem :as-child="true">
        <RouterLink :to="{ name: 'edit-sso', params: { id: props.role.id } }">
          <Pencil class="mr-2 h-4 w-4" />
          {{ i18n._('Edit') }}
        </RouterLink>
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
          {{ i18n._('This action cannot be undone. This will permanently delete this SSO.') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDelete">
          {{ i18n._('Delete') }}
        </AlertDialogAction>
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
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

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
