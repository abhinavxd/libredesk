<template>
  <form class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="applies_to">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.appliesTo') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField" :modelValue="componentField.modelValue">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="contact">
                  {{ $t('globals.terms.contact') }}
                </SelectItem>
                <SelectItem value="conversation">
                  {{ $t('globals.terms.conversation') }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription> </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormDescription />
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="key">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.key') }}</FormLabel>
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

    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.description') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormDescription />
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="data_type">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.type') }}</FormLabel>
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

    <FormField name="values" v-slot="{ componentField, handleChange }">
      <FormItem v-show="form.values.data_type === 'list'">
        <FormLabel>
          {{ $t('globals.terms.listValues') }}
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

    <FormField name="regex" v-slot="{ componentField }">
      <FormItem v-show="form.values.data_type === 'text'">
        <FormLabel>
          {{ $t('globals.terms.regex') }} ({{ $t('globals.terms.optional') }})
        </FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          {{ $t('admin.customAttributes.regex.description') }} e.g. ^[a-zA-Z]*$</FormDescription
        >
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="regex_hint" v-slot="{ componentField }">
      <FormItem v-show="form.values.data_type === 'text'">
        <FormLabel>
          {{ $t('globals.terms.regexHint') }} ({{ $t('globals.terms.optional') }})
        </FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          {{ $t('admin.customAttributes.regexHint.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Form submit button slot -->
    <slot name="footer"></slot>
  </form>
</template>

<script setup>
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
