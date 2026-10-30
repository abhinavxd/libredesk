<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }} </FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="message_content">
      <FormItem>
        <FormLabel>{{ i18n._('Response to be sent when macro is used (optional)') }}</FormLabel>
        <FormControl>
          <div class="box p-2 h-96 min-h-96">
            <Editor
              v-model:htmlContent="componentField.modelValue"
              @update:htmlContent="(value) => componentField.onChange(value)"
              :placeholder="i18n._('Shift + Enter to add a new line.')"
            />
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField
      v-slot="{ componentField }"
      name="actions"
      :validate-on-blur="false"
      :validate-on-change="false"
    >
      <FormItem>
        <FormLabel>
          {{ i18n.ngettext('Action', 'Actions', 2) }} ({{ i18n.ngettext('Optional', 'Optionals', 1) }})
        </FormLabel>
        <FormControl>
          <ActionBuilder
            v-model:actions="componentField.modelValue"
            :config="actionConfig"
            @update:actions="(value) => componentField.onChange(value)"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid>
      <FormField v-slot="{ componentField, handleChange }" name="visible_when">
        <FormItem>
          <FormLabel>{{ i18n._('Visible when') }}</FormLabel>
          <FormControl>
            <SelectTag
              :items="[
                { label: i18n._('Replying'), value: 'replying' },
                {
                  label: i18n._('Starting conversation'),
                  value: 'starting_conversation'
                },
                {
                  label: i18n._('Adding private notes'),
                  value: 'adding_private_note'
                }
              ]"
              v-model="componentField.modelValue"
              @update:modelValue="handleChange"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField
        v-slot="{ componentField }"
        name="visibility"
        :validate-on-blur="false"
        :validate-on-change="false"
        :validate-on-input="false"
        :validate-on-mount="false"
        :validate-on-model-update="false"
      >
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Visibility', 'Visibilities', 1) }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="all">{{ i18n._('All agents') }}</SelectItem>
                  <SelectItem value="team">{{ i18n.ngettext('Team', 'Teams', 1) }}</SelectItem>
                  <SelectItem value="user">{{ i18n.ngettext('Agent', 'Agents', 1) }}</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField
        v-if="form.values.visibility === 'team'"
        v-slot="{ componentField }"
        name="team_id"
      >
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Team', 'Teams', 1) }}</FormLabel>
          <FormControl>
            <SelectTeamCombobox v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField
        v-if="form.values.visibility === 'user'"
        v-slot="{ componentField }"
        name="user_id"
      >
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Agent', 'Agents', 1) }}</FormLabel>
          <FormControl>
            <SelectAgentCombobox v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <Button type="submit" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { ref, watch, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Spinner } from '@shared-ui/components/ui/spinner/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import ActionBuilder from '@/features/admin/macros/ActionBuilder.vue'
import { useConversationFilters } from '@main/composables/useConversationFilters.js'
import { getTextFromHTML } from '@shared-ui/utils/string'
import { createFormSchema } from './formSchema.js'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SelectTag
} from '@shared-ui/components/ui/select/index.js'
import { useI18n } from '@shared-ui/i18n'
import Editor from '@main/components/editor/ConversationEditor.vue'

const { macroActions } = useConversationFilters()
const i18n = useI18n()
const formLoading = ref(false)
const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.initialValues.id ? i18n._('Save') : i18n._('Create'))
  )
})
const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n)),
  initialValues: {
    visible_when: props.initialValues.visible_when || [
      'replying',
      'starting_conversation',
      'adding_private_note'
    ],
    visibility: props.initialValues.visibility || 'all'
  }
})

const actionConfig = ref({
  actions: macroActions,
  typePlaceholder: i18n._('Select action'),
  valuePlaceholder: i18n._('Select value'),
  addButtonText: i18n._('Add action')
})

const onSubmit = form.handleSubmit(async (values) => {
  // If the text of HTML is empty then set the HTML to empty string
  const textContent = getTextFromHTML(values.message_content)
  if (textContent.length === 0) {
    values.message_content = ''
  }
  props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) return
    form.setValues(newValues, false)
  },
  { immediate: true }
)
</script>
