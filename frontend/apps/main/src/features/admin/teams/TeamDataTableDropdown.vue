<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Open menu</span>
        <MoreVertical class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editTeam(props.team.id)">{{ t('globals.messages.edit') }}</DropdownMenuItem>
      <DropdownMenuItem @click="() => (alertOpen = true)">{{ t('globals.messages.delete') }}</DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.delete') }} {{ t('globals.terms.team', 1) }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ t('confirm.deleteTeam') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">{{ t('globals.messages.delete') }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref } from 'vue'
import { MoreVertical } from 'lucide-vue-next'
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
import { useRouter } from 'vue-router'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '../../../api'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const alertOpen = ref(false)
const router = useRouter()
const emit = useEmitter()

const props = defineProps({
  team: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

function editTeam(id) {
  router.push({ path: `/admin/teams/teams/${id}/edit` })
}

async function handleDelete() {
  try {
    await api.deleteTeam(props.team.id)
    alertOpen.value = false
    emitRefreshTeamList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const emitRefreshTeamList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'team'
  })
}
</script>
