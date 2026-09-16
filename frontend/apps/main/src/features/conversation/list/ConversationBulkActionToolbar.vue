<template>
  <div
    role="toolbar"
    :aria-label="t('conversation.bulkActions.toolbar')"
    class="p-2 flex items-center gap-1 bg-muted/30"
  >
    <Checkbox
      :checked="conversationStore.allSelected"
      @update:checked="toggleSelectAll"
      :aria-label="t('conversation.bulkActions.selectAll')"
      class="ml-1 mr-1"
    />
    <span
      class="text-xs font-medium whitespace-nowrap tabular-nums inline-block min-w-20 mr-1"
      aria-live="polite"
    >
      {{ t('conversation.bulkActions.selected', conversationStore.selectedCount, { count: conversationStore.selectedCount }) }}
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
          :title="t('actions.assignAgent')"
          :aria-label="t('actions.assignAgent')"
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
          :title="t('actions.assignTeam')"
          :aria-label="t('actions.assignTeam')"
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
          :title="t('actions.addTags')"
          :aria-label="t('actions.addTags')"
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
          :title="t('actions.setStatus')"
          :aria-label="t('actions.setStatus')"
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
      :aria-label="t('conversation.bulkActions.clearSelection')"
      @click="conversationStore.clearSelection()"
    >
      <X class="w-4 h-4" />
    </Button>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
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
const { t } = useI18n()
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
