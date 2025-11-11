<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Open menu</span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuItem @click="viewOrganization(props.organization.id)">
        <Eye class="w-4 h-4 mr-2" />
        {{ $t('globals.messages.view') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="editOrganization(props.organization.id)">
        <Pencil class="w-4 h-4 mr-2" />
        {{ $t('globals.messages.edit') }}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem @click="() => (alertOpen = true)" class="text-red-600">
        <Trash2 class="w-4 h-4 mr-2" />
        {{ $t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('globals.messages.deleteOrganizationConfirmation') }}
          <span v-if="props.organization.contact_count > 0" class="block mt-2 text-amber-600">
            {{
              $t('globals.messages.organizationHasContacts', {
                count: props.organization.contact_count
              })
            }}
          </span>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="handleDelete" class="bg-red-600 hover:bg-red-700">
          {{ $t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref } from 'vue'
import { MoreHorizontal, Eye, Pencil, Trash2 } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
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
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const alertOpen = ref(false)
const emit = useEmitter()
const router = useRouter()
const { t } = useI18n()

const props = defineProps({
  organization: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      contact_count: 0
    })
  }
})

function viewOrganization(id) {
  router.push({ path: `/admin/organizations/${id}` })
}

function editOrganization(id) {
  router.push({ path: `/admin/organizations/${id}/edit` })
}

async function handleDelete() {
  try {
    await api.deleteOrganization(props.organization.id)
    alertOpen.value = false
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })
    emitRefreshOrganizationList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const emitRefreshOrganizationList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'organization'
  })
}
</script>
