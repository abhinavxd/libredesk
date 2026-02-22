<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            v-bind="componentField"
            :disabled="!isOutgoingTemplate"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="subject" v-if="!isOutgoingTemplate">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.subject') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="body">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.body') }}</FormLabel>
        <FormControl>
          <CodeEditor v-model="componentField.modelValue" @update:modelValue="handleChange" />
        </FormControl>
        <FormDescription v-if="isOutgoingTemplate">
          {{
            $t('admin.template.makeSureTemplateHasContent', {
              content: '\u007b\u007b template "content" . \u007d\u007d'
            })
          }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="is_default" v-slot="{ value, handleChange }" v-if="isOutgoingTemplate">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>{{ $t('globals.terms.isDefault') }}</Label>
          </div>
        </FormControl>
        <FormDescription>{{ $t('admin.template.onlyOneDefaultOutgoingTemplate') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import CodeEditor from '@main/components/editor/CodeEditor.vue'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => ''
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})
const { t } = useI18n()

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.messages.save')
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

const isOutgoingTemplate = computed(() => {
  return props.initialValues?.type === 'email_outgoing'
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
  },
  { deep: true }
)
</script>
