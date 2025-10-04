<template>
  <div class="space-y-5 rounded" :class="{ 'box p-5': actions.length > 0 }">
    <div class="space-y-5">
      <div v-for="(action, index) in actions" :key="index" class="space-y-5">
        <div v-if="index > 0">
          <hr class="border-t-2 border-dotted border-gray-300" />
        </div>

        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="flex gap-5">
              <div class="w-48">
                <!-- Type -->
                <Select
                  v-model="action.type"
                  @update:modelValue="(value) => handleFieldChange(value, index)"
                >
                  <SelectTrigger class="m-auto">
                    <SelectValue :placeholder="t('globals.messages.select', { name: t('globals.terms.action').toLowerCase() })" />
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
                class="w-full"
              >
                <SelectTag
                  v-model="action.value"
                  :items="tagsStore.tagNames.map((tag) => ({ label: tag, value: tag }))"
                  :placeholder="t('globals.messages.select', { name: t('globals.terms.tag', 2).toLowerCase() })"
                />
              </div>

              <div
                class="w-48"
                v-if="action.type && conversationActions[action.type]?.type === 'select'"
              >
                <SelectComboBox
                  v-model="action.value[0]"
                  :items="conversationActions[action.type]?.options"
                  :placeholder="t('globals.messages.select', { name: '' })"
                  @select="handleValueChange($event, index)"
                  :type="action.type === 'assign_team' ? 'team' : 'user'"
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
              :placeholder="t('editor.newLine') + t('editor.send') + t('editor.ctrlK')"
            />
          </div>
        </div>
      </div>
    </div>
    <div>
      <Button variant="outline" @click.prevent="addAction">{{
        $t('globals.messages.add', {
          name: $t('globals.terms.action')
        })
      }}</Button>
    </div>
  </div>
</template>

<script setup>
import { toRefs } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import CloseButton from '@main/components/button/CloseButton.vue'
import { useTagStore } from '../../../stores/tag'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { SelectTag } from '@shared-ui/components/ui/select'
import { useConversationFilters } from '../../../composables/useConversationFilters'
import { getTextFromHTML } from '../../../utils/strings.js'
import { useI18n } from 'vue-i18n'
import Editor from '@main/components/editor/TextEditor.vue'
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
const { conversationActions } = useConversationFilters()

const handleFieldChange = (value, index) => {
  actions.value[index].value = []
  actions.value[index].type = value
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
