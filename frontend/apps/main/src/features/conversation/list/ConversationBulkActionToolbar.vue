<template>
  <div
    role="toolbar"
    :aria-label="i18n._('Bulk actions toolbar')"
    class="p-2 flex flex-wrap items-center gap-1 bg-muted/30"
  >
    <Checkbox
      :checked="conversationStore.allSelected"
      @update:checked="toggleSelectAll"
      :aria-label="i18n._('Select all conversations')"
      class="ml-1 mr-1"
    />
    <span
      class="text-xs font-medium whitespace-nowrap tabular-nums inline-block min-w-20 mr-1"
      aria-live="polite"
    >
      {{ conversationStore.selectedCount === 0
        ? i18n._('No conversations selected')
        : i18n.ngettext('1 selected', '{count} selected', conversationStore.selectedCount) }}
    </span>

    <!-- Assign Agent -->
    <SelectAgentCombobox
      v-if="canAssignAgent"
      include-none
      align="start"
      @select="(item) => onAssigneeSelect('user', item)"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="i18n._('Assign agent')"
          :aria-label="i18n._('Assign agent')"
        >
          <UserPlus class="w-4 h-4" />
        </Button>
      </template>
    </SelectAgentCombobox>

    <!-- Assign Team -->
    <SelectTeamCombobox
      v-if="canAssignTeam"
      include-none
      align="start"
      @select="(item) => onAssigneeSelect('team', item)"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="i18n._('Assign team')"
          :aria-label="i18n._('Assign team')"
        >
          <Users class="w-4 h-4" />
        </Button>
      </template>
    </SelectTeamCombobox>

    <!-- Add Tag -->
    <SelectTagCombobox
      v-if="canUpdateTags"
      align="start"
      @select="onTagSelect"
    >
      <template #trigger>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="i18n._('Add tags')"
          :aria-label="i18n._('Add tags')"
        >
          <Tag class="w-4 h-4" />
        </Button>
      </template>
    </SelectTagCombobox>

    <!-- Set Status -->
    <DropdownMenu v-if="canUpdateStatus">
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          :disabled="bulkLoading"
          :title="i18n._('Set status')"
          :aria-label="i18n._('Set status')"
        >
          <CircleDot class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        <DropdownMenuItem
          v-for="status in conversationStore.statusOptionsNoSnooze"
          :key="status.value"
          @click="bulkUpdateStatus(status.label)"
        >
          {{ status.label }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <Loader2 v-if="bulkLoading" class="w-4 h-4 animate-spin text-muted-foreground ml-2" />

    <Button
      variant="ghost"
      size="icon"
      class="ml-auto"
      :aria-label="i18n._('Clear selection')"
      @click="conversationStore.clearSelection()"
    >
      <X class="w-4 h-4" />
    </Button>
  </div>
</template>

<script setup>
import { useI18n } from '@shared-ui/i18n'
import { UserPlus, Users, Tag, CircleDot, Loader2, X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import SelectTagCombobox from '@main/components/combobox/SelectTagCombobox.vue'
import { useConversationStore } from '@/stores/conversation'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { useBulkActions } from '@/composables/useBulkActions'

const conversationStore = useConversationStore()
const i18n = useI18n()
const { bulkLoading, bulkAssign, bulkAddTag, bulkUpdateStatus } = useBulkActions()

const { canAssignAgent, canAssignTeam, canUpdateStatus, canUpdateTags } = useBulkActionPermissions()

const toggleSelectAll = () => {
  if (conversationStore.allSelected) {
    conversationStore.clearSelection()
  } else {
    conversationStore.selectAll()
  }
}

const onAssigneeSelect = (assigneeType, item) => bulkAssign(assigneeType, item.value)

const onTagSelect = (item) => bulkAddTag(item.value)
</script>
