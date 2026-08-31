<template>
  <aside class="zendesk-ticket-props p-4 space-y-1 min-w-0 min-h-0 overflow-y-auto">
    <div class="prop-field">
      <p class="prop-label">{{ t('zendesk.requester') }}</p>
      <p class="text-sm font-medium truncate">
        {{ conversationStore.currentContactName }}
      </p>
    </div>

    <div class="prop-field">
      <p class="prop-label">{{ t('zendesk.assignee') }}</p>
      <SelectComboBox
        v-if="conversationStore.current"
        v-model="conversationStore.current.assigned_user_id"
        :items="[{ value: 'none', label: t('globals.terms.none') }, ...agentOptions]"
        :placeholder="t('placeholders.selectAgent')"
        @select="selectAgent"
        type="user"
      />
    </div>

    <div class="prop-field">
      <p class="prop-label">{{ t('globals.terms.team') }}</p>
      <SelectComboBox
        v-if="conversationStore.current"
        v-model="conversationStore.current.assigned_team_id"
        :items="[{ value: 'none', label: t('globals.terms.none') }, ...teamsStore.options]"
        :placeholder="t('placeholders.selectTeam')"
        @select="selectTeam"
        type="team"
      />
    </div>

    <div class="prop-field">
      <p class="prop-label">{{ t('globals.terms.priority') }}</p>
      <SelectComboBox
        v-if="conversationStore.current"
        v-model="conversationStore.current.priority_id"
        :items="priorityOptions"
        :placeholder="t('placeholders.selectPriority')"
        @select="selectPriority"
        type="priority"
      />
    </div>

    <div class="prop-field">
      <p class="prop-label">{{ t('globals.terms.tag', 2) }}</p>
      <SelectTag
        v-if="conversationStore.current"
        :model-value="conversationStore.current.tags || []"
        @update:modelValue="onTagsChange"
        :items="tagItems"
        :placeholder="t('placeholders.selectTags')"
      />
    </div>

    <div class="prop-field">
      <p class="prop-label">{{ t('globals.terms.status') }}</p>
      <SelectComboBox
        v-if="conversationStore.current"
        :model-value="conversationStore.current.status"
        :items="statusOptions"
        :placeholder="t('placeholders.selectStatus')"
        @select="selectStatus"
        type="status"
      />
    </div>

    <div class="prop-field pt-2">
      <p class="prop-label">{{ t('globals.terms.macro', 2) }}</p>
      <ZendeskMacroPicker />
    </div>

    <div class="pt-4">
      <ConversationSpamActions />
    </div>
  </aside>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { SelectTag } from '@shared-ui/components/ui/select'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useTeamStore } from '@main/stores/team'
import { useTagStore } from '@main/stores/tag'
import { TAG_ACTION } from '@/constants/conversation'
import { useTeamFilteredAgentOptions } from '@main/composables/useTeamFilteredAgentOptions'
import ZendeskMacroPicker from './ZendeskMacroPicker.vue'
import ConversationSpamActions from '@/features/conversation/ConversationSpamActions.vue'

const { t } = useI18n()
const conversationStore = useConversationStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()

const agentOptions = useTeamFilteredAgentOptions(
  computed(() => conversationStore.current?.assigned_team_id),
  computed(() => conversationStore.current?.assigned_user_id)
)

onMounted(async () => {
  await tagStore.fetchTags()
})

const tagItems = computed(() =>
  tagStore.tagNames.map((tag) => ({ label: tag, value: tag }))
)
const priorityOptions = computed(() => conversationStore.priorityOptions)
const statusOptions = computed(() =>
  conversationStore.statusOptions.map((s) => ({ value: s.label, label: s.label }))
)

const handleAssignedUserChange = (id) => {
  conversationStore.updateAssignee('user', { assignee_id: parseInt(id) })
}

const handleAssignedTeamChange = (id) => {
  conversationStore.updateAssignee('team', { assignee_id: parseInt(id) })
}

const selectAgent = (agent) => {
  if (agent.value === 'none') {
    conversationStore.removeAssignee('user')
    return
  }
  conversationStore.current.assigned_user_id = agent.value
  handleAssignedUserChange(agent.value)
}

const selectTeam = (team) => {
  if (team.value === 'none') {
    conversationStore.removeAssignee('team')
    return
  }
  handleAssignedTeamChange(team.value)
}

const selectPriority = (priority) => {
  conversationStore.current.priority = priority.label
  conversationStore.current.priority_id = priority.value
  conversationStore.updatePriority(priority.label)
}

const selectStatus = (status) => {
  conversationStore.updateStatus(status.value)
}

const onTagsChange = async (newTags) => {
  const conv = conversationStore.current
  if (!conv) return
  await conversationStore.updateConversationTags(conv.uuid, TAG_ACTION.SET, newTags)
  tagStore.invalidateTags()
  await tagStore.fetchTags({ force: true })
}
</script>
