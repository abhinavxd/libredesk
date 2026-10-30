<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem @click="emit('open', props.helpCenter)">
        <FileText class="mr-2 h-4 w-4" />
        {{ $t('globals.terms.article', 2) }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="emit('edit', props.helpCenter)">
        <Pencil class="mr-2 h-4 w-4" />
        {{ $t('globals.messages.edit') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="visitSite">
        <ExternalLink class="mr-2 h-4 w-4" />
        {{ $t('helpCenter.visitSite') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="emit('toggle', props.helpCenter)">
        <component :is="props.helpCenter.is_active ? PowerOff : Power" class="mr-2 h-4 w-4" />
        {{ props.helpCenter.is_active ? $t('helpCenter.pause') : $t('helpCenter.resume') }}
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
          {{ $t('helpCenter.deleteConfirmation') }}
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
import { ExternalLink, FileText, Pencil, Power, PowerOff, Trash } from 'lucide-vue-next'
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
import { useAppSettingsStore } from '@/stores/appSettings'

const alertOpen = ref(false)
const appSettingsStore = useAppSettingsStore()

const props = defineProps({
  helpCenter: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['open', 'edit', 'delete', 'toggle'])

function visitSite() {
  const rootUrl = appSettingsStore.settings?.['app.root_url'] || window.location.origin
  window.open(`${rootUrl.replace(/\/$/, '')}/hc/${props.helpCenter.slug}`, '_blank', 'noopener')
}

function handleDelete() {
  emit('delete', props.helpCenter)
  alertOpen.value = false
}
</script>
