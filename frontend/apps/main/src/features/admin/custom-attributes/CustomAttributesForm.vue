<template>
  <form class="space-y-6 w-full">
    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormDescription />
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="key">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Key', 'Keys', 1) }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              v-bind="componentField"
              :readonly="form.values.id && form.values.id > 0"
            />
          </FormControl>
          <FormDescription></FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="applies_to">
        <FormItem>
          <FormLabel>{{ i18n._('Applies to') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField" :modelValue="componentField.modelValue">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="contact">
                    {{ i18n.ngettext('Contact', 'Contacts', 1) }}
                  </SelectItem>
                  <SelectItem value="conversation">
                    {{ i18n.ngettext('Conversation', 'Conversations', 1) }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription> </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="data_type">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Type', 'Types', 1) }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField" :disabled="!!(form.values.id && form.values.id > 0)">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="text"> Text </SelectItem>
                  <SelectItem value="number"> Number </SelectItem>
                  <SelectItem value="checkbox"> Checkbox </SelectItem>
                  <SelectItem value="date"> Date </SelectItem>
                  <SelectItem value="link"> Link </SelectItem>
                  <SelectItem value="list"> List </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription> </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="description">
        <FormItem class="col-span-full">
          <FormLabel>{{ i18n.ngettext('Description', 'Descriptions', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" />
          </FormControl>
          <FormDescription />
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormField name="values" v-slot="{ componentField, handleChange }">
      <FormItem v-show="form.values.data_type === 'list'">
        <FormLabel>
          {{ i18n._('List values') }}
        </FormLabel>
        <FormControl>
          <TagsInput :modelValue="componentField.modelValue" @update:modelValue="handleChange">
            <TagsInputItem v-for="item in componentField.modelValue" :key="item" :value="item">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput placeholder="" />
          </TagsInput>
        </FormControl>
        <FormDescription> </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid>
      <FormField name="regex" v-slot="{ componentField }">
        <FormItem v-show="form.values.data_type === 'text'">
          <FormLabel>
            {{ i18n.ngettext('Regex', 'Regexes', 1) }} ({{ i18n.ngettext('Optional', 'Optionals', 1) }})
          </FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ i18n._('Regex to validate the value of this custom attribute. Leave empty to skip validation.') }} e.g. ^[a-zA-Z]*$</FormDescription
          >
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField name="regex_hint" v-slot="{ componentField }">
        <FormItem v-show="form.values.data_type === 'text'">
          <FormLabel>
            {{ i18n._('Regex hint') }} ({{ i18n.ngettext('Optional', 'Optionals', 1) }})
          </FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ i18n._('Regex pattern hint.') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <!-- Form submit button slot -->
    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { watch } from 'vue'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@shared-ui/components/ui/tags-input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Input } from '@shared-ui/components/ui/input'

const props = defineProps({
  form: {
    type: Object,
    required: true
  }
})

// Function to generate the key from the name
const generateKeyFromName = (name) => {
  if (!name) return ''
  // Remove invalid characters (allow only lowercase letters, numbers, and underscores)
  return (
    name
      .toLowerCase()
      .trim()
      // Replace spaces with underscores
      .replace(/\s+/g, '_')
      // Remove any other invalid characters
      .replace(/[^a-z0-9_]/g, '')
  )
}

// Watch for changes in the name field and update the key field
watch(
  () => props.form.values.name,
  (newName) => {
    // Don't update if the form is in edit mode
    if (props.form.values.id && props.form.values.id > 0) return
    const generatedKey = generateKeyFromName(newName)
    // Check if the generated key is different from the current key
    if (generatedKey !== props.form.values.key) {
      // Clear the error if it exists and set the new key
      props.form.setFieldError('key', undefined)
      props.form.setFieldValue('key', generatedKey)
    }
  }
)
</script>
