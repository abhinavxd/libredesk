<template>
  <div class="box p-5 space-y-6">
    <div class="space-y-6">
      <div v-for="(action, index) in actions" :key="index" class="space-y-6">
        <div v-if="index > 0">
          <hr class="border-t-2 border-dotted border-border" />
        </div>

        <div class="space-y-3">
          <div class="flex items-start justify-between gap-5">
            <div class="flex gap-5 flex-1 min-w-0">
              <div :class="fieldClass">
                <!-- Type -->
                <Select
                  v-model="action.type"
                  @update:modelValue="(value) => handleFieldChange(value, index)"
                >
                  <SelectTrigger class="m-auto">
                    <SelectValue :placeholder="i18n._('Select action')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem
                        v-for="(actionConfig, key) in conversationActions"
                        :key="key"
                        :value="key"
                      >
                        {{ actionConfig.label }}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <div
                v-if="action.type && conversationActions[action.type]?.type === 'richtext'"
                class="flex self-center"
              >
                <HelpPopover :text="messageVariablesHelp" />
              </div>

              <!-- Value -->
              <div
                v-if="action.type && conversationActions[action.type]?.type === 'tag'"
                class="flex-1 min-w-0 max-w-md"
              >
                <SelectTagCombobox multiple v-model="action.value" />
              </div>

              <div
                v-if="action.type && conversationActions[action.type]?.type === 'recipients'"
                class="flex-1 min-w-0 max-w-md space-y-3"
              >
                <SelectTag
                  :modelValue="action.recipients || []"
                  @update:modelValue="(value) => handleNotifyRecipientsChange(value, index)"
                  :items="notifyRecipientOptions"
                  :placeholder="i18n._('Select recipients')"
                  :search="searchRecipients"
                />
                <Input
                  type="text"
                  :placeholder="i18n.ngettext('Subject', 'Subjects', 1)"
                  :modelValue="action.subject || ''"
                  @update:modelValue="(value) => handleNotifySubjectChange(value, index)"
                />
                <Textarea
                  :placeholder="i18n.ngettext('Message', 'Messages', 1)"
                  :modelValue="action.message || ''"
                  @update:modelValue="(value) => handleNotifyMessageChange(value, index)"
                  class="h-36"
                />
              </div>

              <div
                :class="fieldClass"
                v-if="action.type && conversationActions[action.type]?.type === 'select'"
              >
                <SelectAgentCombobox
                  v-if="action.type === 'assign_user'"
                  v-model="action.value[0]"
                  @select="handleValueChange($event, index)"
                />
                <SelectTeamCombobox
                  v-else-if="action.type === 'assign_team'"
                  v-model="action.value[0]"
                  @select="handleValueChange($event, index)"
                />
                <SelectComboBox
                  v-else
                  v-model="action.value[0]"
                  :items="conversationActions[action.type]?.options"
                  :placeholder="i18n._('Select value')"
                  @select="handleValueChange($event, index)"
                />
              </div>

              <div
                class="flex gap-3 flex-1 min-w-0 max-w-2xl"
                v-if="action.type && conversationActions[action.type]?.type === 'webhook'"
              >
                <div class="flex-1 min-w-0">
                  <SelectComboBox
                    v-model="action.value[0]"
                    :items="webhookStore.options"
                    :placeholder="i18n._('Select webhook')"
                    @select="handleWebhookChange($event, index)"
                  />
                </div>
                <div class="flex-1 min-w-0">
                  <Input
                    type="text"
                    :placeholder="i18n._('e.g. priority.escalated')"
                    :modelValue="action.value[1] || ''"
                    @update:modelValue="(value) => handleWebhookEventChange(value, index)"
                  />
                  <p class="text-xs text-muted-foreground mt-1">
                    {{ i18n._('Event name sent in the webhook payload.') }}
                  </p>
                </div>
              </div>

              <div
                :class="fieldClass"
                v-if="action.type && conversationActions[action.type]?.type === 'text'"
              >
                <Input
                  type="text"
                  :placeholder="placeholderForText(action.type)"
                  :modelValue="action.value[0] || ''"
                  @update:modelValue="(value) => handleTextChange(value, index)"
                />
              </div>
            </div>

            <CloseButton :onClose="() => removeAction(index)" />
          </div>

          <div
            class="box p-2 h-96 min-h-96"
            v-if="action.type && conversationActions[action.type]?.type === 'richtext'"
          >
            <Editor
              :autoFocus="false"
              v-model:htmlContent="action.value[0]"
              @update:htmlContent="(value) => handleEditorChange(value, index)"
              :placeholder="i18n._('Shift + Enter to add a new line.')"
            />
          </div>
        </div>
      </div>
    </div>
    <div>
      <Button type="button" variant="outline" @click.prevent="addAction">{{
        i18n._('Add action')
      }}</Button>
    </div>
  </div>
</template>

<script setup>
import { toRefs, computed, watch, onMounted } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import CloseButton from '@main/components/button/CloseButton.vue'
import { useWebhookStore } from '@main/stores/webhook'
import { useUsersStore } from '@main/stores/users'
import { useTeamStore } from '@main/stores/team'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { SelectTag } from '@shared-ui/components/ui/select'
import HelpPopover from '@main/components/HelpPopover.vue'
import { useConversationFilters } from '@main/composables/useConversationFilters'
import { getTextFromHTML, formatTemplateVariables } from '@shared-ui/utils/string'
import { useI18n } from '@shared-ui/i18n'
import Editor from '@main/components/editor/ConversationEditor.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import SelectTagCombobox from '@main/components/combobox/SelectTagCombobox.vue'

const props = defineProps({
  actions: {
    type: Array,
    required: true
  }
})

const fieldClass = 'flex-1 min-w-0 max-w-xs'

// Automations send as the System user, so the Author fields would render blank.
const MESSAGE_VARIABLES = [
  '.Conversation.ReferenceNumber',
  '.Conversation.Subject',
  '.Conversation.Priority',
  '.Contact.FirstName',
  '.Contact.LastName',
  '.Contact.FullName',
  '.Contact.Email',
  '.Inbox.Name'
]

const { actions } = toRefs(props)
const i18n = useI18n()
const emit = defineEmits(['update-actions', 'add-action', 'remove-action'])
const webhookStore = useWebhookStore()
const uStore = useUsersStore()
const tStore = useTeamStore()
const { conversationActions } = useConversationFilters()
const messageVariablesHelp = computed(
  () => i18n._('Variables: {variables}', { variables: formatTemplateVariables(MESSAGE_VARIABLES) })
)

webhookStore.fetchWebhooks()

const recipientKeywords = computed(() => [
  { label: i18n._('Assignee'), value: 'assignee' },
  { label: i18n._('Assigned team'), value: 'assigned_team' }
])

const toTeamRecipients = (options) => options.map((o) => ({
    label: `${i18n._('{count, plural, one {Team} other {Teams}}')}: ${o.label}`,
    value: `team:${o.value}`
  }))

const toUserRecipients = (options) => options.map((o) => ({
    label: `${i18n._('{count, plural, one {Agent} other {Agents}}')}: ${o.label}`,
    value: `user:${o.value}`
  }))

const notifyRecipientOptions = computed(() => [
  ...recipientKeywords.value,
  ...toTeamRecipients(tStore.options),
  ...toUserRecipients(uStore.options)
])

const searchRecipients = async (query) => {
  const [teams, users] = await Promise.all([
    tStore.searchTeams(query),
    uStore.searchUsers(query)
  ])
  return [
    ...recipientKeywords.value,
    ...toTeamRecipients(teams),
    ...toUserRecipients(users)
  ]
}

// Recipients are prefixed, e.g. "user:12" / "team:3", alongside keywords like "assignee".
const recipientIDs = (prefix) =>
  actions.value.flatMap((action) =>
    (action.recipients || [])
      .filter((r) => typeof r === 'string' && r.startsWith(prefix + ':'))
      .map((r) => r.slice(prefix.length + 1))
  )

onMounted(() => {
  uStore.fetchUsers()
  tStore.fetchTeams()
})

// Keyed on the ids alone: a deep watch on actions would refire on every subject/message keystroke.
const recipientIDKey = computed(() => JSON.stringify([recipientIDs('user'), recipientIDs('team')]))

watch(
  recipientIDKey,
  () => {
    uStore.ensureUserIDs(recipientIDs('user'))
    tStore.ensureTeamIDs(recipientIDs('team'))
  },
  { immediate: true }
)

const handleNotifyRecipientsChange = (value, index) => {
  actions.value[index].recipients = value || []
  emitUpdate(index)
}

const handleNotifySubjectChange = (value, index) => {
  actions.value[index].subject = value
  emitUpdate(index)
}

const handleNotifyMessageChange = (value, index) => {
  actions.value[index].message = value
  emitUpdate(index)
}

const handleTextChange = (value, index) => {
  actions.value[index].value = [value]
  emitUpdate(index)
}

const handleWebhookChange = (value, index) => {
  if (typeof value === 'object') {
    value = value.value
  }
  const current = actions.value[index].value || []
  actions.value[index].value = [value || '', current[1] || '']
  emitUpdate(index)
}

const handleWebhookEventChange = (value, index) => {
  const current = actions.value[index].value || []
  actions.value[index].value = [current[0] || '', value]
  emitUpdate(index)
}

const placeholderForText = (type) => {
  if (type === 'snooze') return i18n._('e.g. 30m, 3h, 2h30m')
  return i18n._('Set value')
}

const handleFieldChange = (value, index) => {
  const action = actions.value[index]
  action.value = []
  action.type = value
  delete action.subject
  delete action.message
  delete action.recipients
  emitUpdate(index)
}

const handleValueChange = (value, index) => {
  if (typeof value === 'object') {
    value = value.value
  }
  actions.value[index].value = [value]
  emitUpdate(index)
}

const handleEditorChange = (value, index) => {
  // If text is empty, set HTML to empty string
  const textContent = getTextFromHTML(value)
  if (textContent.length === 0) {
    value = ''
  }
  actions.value[index].value = [value]
  emitUpdate(index)
}

const removeAction = (index) => {
  emit('remove-action', index)
}

const addAction = () => {
  emit('add-action')
}

const emitUpdate = (index) => {
  emit('update-actions', actions, index)
}
</script>
