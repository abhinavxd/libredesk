<template>
  <div class="space-y-6" :class="{ 'box p-5': actions.length > 0 }">
    <div class="space-y-6">
      <div v-for="(action, index) in actions" :key="index" class="space-y-6">
        <div v-if="index > 0">
          <hr class="border-t-2 border-dotted border-border" />
        </div>

        <div class="space-y-3">
          <div class="flex items-start justify-between gap-5">
            <div class="flex gap-5 flex-1 min-w-0">
              <div class="w-56 shrink-0">
                <!-- Type -->
                <Select
                  v-model="action.type"
                  @update:modelValue="(value) => handleFieldChange(value, index)"
                >
                  <SelectTrigger class="m-auto">
                    <SelectValue :placeholder="t('placeholders.selectAction')" />
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

              <!-- Value -->
              <div
                v-if="action.type && conversationActions[action.type]?.type === 'tag'"
                class="flex-1 min-w-0"
              >
                <SelectTag
                  v-model="action.value"
                  :items="tagsStore.tagNames.map((tag) => ({ label: tag, value: tag }))"
                  :placeholder="t('placeholders.selectTags')"
                />
              </div>

              <div
                v-if="action.type && conversationActions[action.type]?.type === 'recipients'"
                class="flex-1 min-w-0 space-y-3"
              >
                <SelectTag
                  :modelValue="action.recipients || []"
                  @update:modelValue="(value) => handleNotifyRecipientsChange(value, index)"
                  :items="notifyRecipientOptions"
                  :placeholder="t('placeholders.selectRecipients')"
                />
                <Input
                  type="text"
                  :placeholder="t('globals.terms.subject')"
                  :modelValue="action.subject || ''"
                  @update:modelValue="(value) => handleNotifySubjectChange(value, index)"
                />
                <Textarea
                  :placeholder="t('globals.terms.message')"
                  :modelValue="action.message || ''"
                  @update:modelValue="(value) => handleNotifyMessageChange(value, index)"
                  class="h-36"
                />
              </div>

              <div
                class="flex-1 min-w-0"
                v-if="action.type && conversationActions[action.type]?.type === 'select'"
              >
                <SelectComboBox
                  v-model="action.value[0]"
                  :items="conversationActions[action.type]?.options"
                  :placeholder="t('placeholders.selectValue')"
                  @select="handleValueChange($event, index)"
                  :type="action.type === 'assign_user' ? 'user' : 'team'"
                />
              </div>

              <div
                class="flex gap-3 flex-1 min-w-0"
                v-if="action.type && conversationActions[action.type]?.type === 'webhook'"
              >
                <div class="flex-1 min-w-0">
                  <SelectComboBox
                    v-model="action.value[0]"
                    :items="webhookStore.options"
                    :placeholder="t('placeholders.selectWebhook')"
                    @select="handleWebhookChange($event, index)"
                  />
                </div>
                <div class="flex-1 min-w-0">
                  <Input
                    type="text"
                    :placeholder="t('placeholders.webhookEventName')"
                    :modelValue="action.value[1] || ''"
                    @update:modelValue="(value) => handleWebhookEventChange(value, index)"
                  />
                  <p class="text-xs text-muted-foreground mt-1">
                    {{ $t('admin.automation.webhookEventNameHint') }}
                  </p>
                </div>
              </div>

              <div
                class="flex-1 min-w-0"
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
              :placeholder="t('editor.newLine')"
            />
          </div>
        </div>
      </div>
    </div>
    <div>
      <Button type="button" variant="outline" @click.prevent="addAction">{{
        $t('actions.addAction')
      }}</Button>
    </div>
  </div>
</template>

<script setup>
import { toRefs, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import CloseButton from '@main/components/button/CloseButton.vue'
import { useTagStore } from '@main/stores/tag'
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
import { useConversationFilters } from '@main/composables/useConversationFilters'
import { getTextFromHTML } from '@shared-ui/utils/string'
import { useI18n } from 'vue-i18n'
import Editor from '@main/components/editor/ConversationEditor.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'

const props = defineProps({
  actions: {
    type: Array,
    required: true
  }
})

const { actions } = toRefs(props)
const { t } = useI18n()
const emit = defineEmits(['update-actions', 'add-action', 'remove-action'])
const tagsStore = useTagStore()
const webhookStore = useWebhookStore()
const uStore = useUsersStore()
const tStore = useTeamStore()
const { conversationActions } = useConversationFilters()

webhookStore.fetchWebhooks()

const notifyRecipientOptions = computed(() => [
  { label: t('globals.terms.assignee'), value: 'assignee' },
  { label: t('globals.terms.assignedTeam'), value: 'assigned_team' },
  ...tStore.options.map((o) => ({
    label: `${t('globals.terms.team')}: ${o.label}`,
    value: `team:${o.value}`
  })),
  ...uStore.options.map((o) => ({
    label: `${t('globals.terms.agent')}: ${o.label}`,
    value: `user:${o.value}`
  }))
])

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
  if (type === 'snooze') return t('placeholders.snoozeDuration')
  return t('actions.setValue')
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
