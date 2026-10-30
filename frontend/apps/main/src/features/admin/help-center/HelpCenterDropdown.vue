<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem @click="emit('open', props.helpCenter)">
        <FileText class="mr-2 h-4 w-4" />
        {{ i18n.ngettext('Article', 'Articles', 2) }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="emit('edit', props.helpCenter)">
        <Pencil class="mr-2 h-4 w-4" />
        {{ i18n._('Edit') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="visitSite">
        <ExternalLink class="mr-2 h-4 w-4" />
        {{ i18n._('Visit site') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="emit('toggle', props.helpCenter)">
        <component :is="props.helpCenter.is_active ? PowerOff : Power" class="mr-2 h-4 w-4" />
        {{ props.helpCenter.is_active ? i18n._('Pause site') : i18n._('Resume site') }}
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
          {{ i18n._('This will permanently delete the help center, including all its collections and articles.') }}
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
