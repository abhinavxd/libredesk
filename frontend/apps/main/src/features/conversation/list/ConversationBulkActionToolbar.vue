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
    <SelectComboBox
      v-if="canAssignAgent"
      :items="agentItems"
      :placeholder="t('placeholders.selectAgent')"
      type="user"
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
    </SelectComboBox>

    <!-- Assign Team -->
    <SelectComboBox
      v-if="canAssignTeam"
      :items="teamItems"
      :placeholder="t('placeholders.selectTeam')"
      type="team"
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
    </SelectComboBox>

    <!-- Add Tag -->
    <SelectComboBox
      v-if="canUpdateTags"
      :items="tagItems"
      :placeholder="t('placeholders.selectTags')"
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
    </SelectComboBox>

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
import { computed, ref, onMounted } from 'vue'
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
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import { TAG_ACTION } from '@/constants/conversation'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useTagStore } from '@/stores/tag'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import api from '@/api'

const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()
const { t } = useI18n()
const emitter = useEmitter()
const bulkLoading = ref(false)

const { canAssignAgent, canAssignTeam, canUpdateStatus, canUpdateTags } = useBulkActionPermissions()

onMounted(() => {
  if (canAssignAgent.value) usersStore.fetchUsers()
  if (canAssignTeam.value) teamsStore.fetchTeams()
  if (canUpdateTags.value) tagStore.fetchTags()
})

const toggleSelectAll = () => {
  if (conversationStore.allSelected) {
    conversationStore.clearSelection()
  } else {
    conversationStore.selectAll()
  }
}

const withNoneOption = (options) => [
  { value: 'none', label: t('globals.terms.none') },
  ...options
]

const agentItems = computed(() => withNoneOption(usersStore.options))
const teamItems = computed(() => withNoneOption(teamsStore.options))
const tagItems = computed(() =>
  tagStore.tagNames.map((name) => ({ label: name, value: name }))
)

const runBulkAction = async (actionFn) => {
  const uuids = [...conversationStore.selectedUUIDs]
  bulkLoading.value = true
  const results = await Promise.allSettled(uuids.map((uuid) => actionFn(uuid)))
  bulkLoading.value = false

  const hasFailures = results.some((r) => r.status === 'rejected')

  conversationStore.clearSelection()
  conversationStore.fetchFirstPageConversations()

  if (hasFailures) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      title: t('globals.terms.error', 1),
      description: t('conversation.bulkActions.failedToast')
    })
  } else {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('conversation.bulkActions.successToast')
    })
  }
}

const onAssigneeSelect = (assigneeType, item) => {
  if (item.value === 'none') {
    runBulkAction((uuid) => api.removeAssignee(uuid, assigneeType))
    return
  }
  const assigneeId = parseInt(item.value, 10)
  runBulkAction((uuid) => api.updateAssignee(uuid, assigneeType, { assignee_id: assigneeId }))
}

const onTagSelect = (item) => {
  runBulkAction((uuid) => conversationStore.updateConversationTags(uuid, TAG_ACTION.ADD, [item.value]))
}

const bulkUpdateStatus = (status) => {
  runBulkAction((uuid) => api.updateConversationStatus(uuid, { status }))
}
</script>
