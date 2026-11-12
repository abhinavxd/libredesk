<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem v-auto-animate>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" :disabled="!isOutgoingTemplate" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="subject" v-if="!hideSubject">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Subject', 'Subjects', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormField v-slot="{ componentField, handleChange }" name="body">
      <FormItem>
        <FormLabel>{{ i18n._('Body') }}</FormLabel>
        <FormControl>
          <CodeEditor v-model="componentField.modelValue" @update:modelValue="handleChange" />
        </FormControl>
        <FormDescription v-if="isOutgoingTemplate">
          {{
            i18n._('Make sure the template has {content} only once.', {
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
            <Label>{{ i18n._('Is default') }}</Label>
          </div>
        </FormControl>
        <FormDescription>{{ i18n._('You can have only one default outgoing email template.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="flex gap-2">
      <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
      <Button
        v-if="(isOutgoingTemplate || initialValues?.is_builtin) && defaultBody"
        type="button"
        variant="outline"
        :disabled="isLoading"
        @click="resetToDefault"
      >
        {{ i18n._('Reset to default') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
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
import { useI18n } from '@shared-ui/i18n'
import { BUILT_IN_TEMPLATE_BODIES, DEFAULT_OUTGOING_TEMPLATE_BODY } from './defaults.js'

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
const i18n = useI18n()

const submitLabel = computed(() => {
  return props.submitLabel || i18n._('Save')
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n)),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

const isOutgoingTemplate = computed(() => {
  return props.initialValues?.type === 'email_outgoing'
})

const hideSubject = computed(() => {
  return isOutgoingTemplate.value || props.initialValues?.name === 'CSAT request'
})

const defaultBody = computed(() => {
  if (isOutgoingTemplate.value) return DEFAULT_OUTGOING_TEMPLATE_BODY
  return BUILT_IN_TEMPLATE_BODIES[props.initialValues?.name]
})

const resetToDefault = () => {
  form.setFieldValue('body', defaultBody.value, false)
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues, false)
  },
  { deep: true }
)
</script>
