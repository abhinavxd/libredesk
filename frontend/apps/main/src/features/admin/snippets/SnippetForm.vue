<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <!-- Type Field (Hidden, fixed to 'snippet') -->
    <FormField v-slot="{ componentField }" name="type">
      <input type="hidden" v-bind="componentField" />
    </FormField>

    <!-- Content Field -->
    <FormField v-slot="{ componentField }" name="content">
      <FormItem>
        <FormLabel>{{ t('globals.terms.content') }} <span class="text-red-500">*</span></FormLabel>
        <FormControl>
          <Editor
            v-model:htmlContent="componentField.modelValue"
            @update:htmlContent="(value) => componentField.onChange(value)"
            editorType="article"
            class="border rounded-md p-2"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Enabled Field -->
    <FormField v-slot="{ value, handleChange }" name="enabled" type="checkbox">
      <FormItem class="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
        <FormControl>
          <Checkbox :checked="value" @update:checked="handleChange" />
        </FormControl>
        <div class="space-y-1 leading-none">
          <FormLabel>{{ t('globals.terms.enabled') }}</FormLabel>
          <FormDescription>{{ t('ai.snippet.enabledDescription') }}</FormDescription>
        </div>
      </FormItem>
    </FormField>

    <!-- Submit Button -->
    <Button type="submit" :disabled="formLoading">
      <Spinner v-if="formLoading" />
      {{ t('globals.messages.save') }}
    </Button>
  </form>
</template>

<script setup>
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Spinner } from '@shared-ui/components/ui/spinner'
import Editor from '@main/components/editor/TextEditor.vue'
import { createFormSchema } from './formSchema.js'

const { t } = useI18n()
const props = defineProps({
  snippet: {
    type: Object,
    default: null
  },
  formLoading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['submit'])

const formSchema = toTypedSchema(createFormSchema(t))

const { handleSubmit, setValues } = useForm({
  validationSchema: formSchema,
  initialValues: {
    type: 'snippet',
    content: '',
    enabled: true
  }
})

const onSubmit = handleSubmit((values) => {
  emit('submit', values)
})

// Watch for changes in snippet prop and update form values
watch(
  () => props.snippet,
  (newSnippet) => {
    if (newSnippet) {
      setValues({
        type: newSnippet.type || 'snippet',
        content: newSnippet.content || '',
        enabled: newSnippet.enabled !== undefined ? newSnippet.enabled : true
      })
    }
  },
  { immediate: true }
)
</script>
