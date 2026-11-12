<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem @click="edit(props.role.id)">
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
          {{ i18n._('This action cannot be undone. This will permanently delete this SLA policy.') }}
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
import { useRouter } from 'vue-router'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const i18n = useI18n()
const router = useRouter()
const emitter = useEmitter()
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

function edit(id) {
  router.push({ path: `/admin/sla/${id}/edit` })
}

async function handleDelete() {
  try {
    await api.deleteSLA(props.role.id)
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, {
      model: 'sla'
    })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    alertOpen.value = false
  }
}
</script>
