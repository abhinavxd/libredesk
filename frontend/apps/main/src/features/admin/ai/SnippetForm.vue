<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="title">
      <FormItem>
        <FormLabel>{{ t('globals.terms.title') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="content">
      <FormItem>
        <FormLabel>{{ t('globals.terms.content') }}</FormLabel>
        <FormControl>
          <Textarea class="min-h-[420px] max-h-[60vh]" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="t('globals.terms.enabled')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <DialogFooter class="mt-10">
      <Button type="submit" :isLoading="formLoading">
        {{ isEditing ? t('globals.messages.save') : t('globals.messages.create') }}
      </Button>
    </DialogFooter>
  </form>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import { DialogFooter } from '@shared-ui/components/ui/dialog/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  initialValues: { type: Object, default: () => ({}) },
  isEditing: { type: Boolean, default: false },
  submitForm: { type: Function, required: true }
})

const { t } = useI18n()
const formLoading = ref(false)

const form = useForm({
  validationSchema: toTypedSchema(
    z.object({
      title: z.string({ required_error: t('globals.messages.required') }).min(1, {
        message: t('globals.messages.required')
      }),
      content: z.string().optional(),
      enabled: z.boolean().optional()
    })
  ),
  initialValues: {
    title: '',
    content: '',
    enabled: true
  }
})

watch(
  () => props.initialValues,
  (values) => {
    form.setValues({
      title: values.title || '',
      content: values.content || '',
      enabled: values.enabled ?? true
    })
    form.setErrors({})
  },
  { immediate: true, deep: true }
)

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await props.submitForm({
      title: values.title,
      content: values.content || '',
      enabled: !!values.enabled
    })
  } finally {
    formLoading.value = false
  }
})
</script>
