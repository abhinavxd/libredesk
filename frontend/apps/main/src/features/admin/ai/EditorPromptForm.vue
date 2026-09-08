<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="title">
      <FormItem>
        <FormLabel>{{ t('globals.terms.title') }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            :placeholder="t('admin.ai.editorPrompt.titlePlaceholder')"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="content">
      <FormItem>
        <FormLabel>{{ t('globals.terms.content') }}</FormLabel>
        <FormControl>
          <Textarea
            class="min-h-[240px] max-h-[60vh]"
            :placeholder="t('admin.ai.editorPrompt.contentPlaceholder')"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
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
      title: z
        .string({ required_error: t('globals.messages.required') })
        .min(1, { message: t('globals.messages.required') })
        .max(140),
      content: z
        .string({ required_error: t('globals.messages.required') })
        .min(1, { message: t('globals.messages.required') })
    })
  ),
  initialValues: { title: '', content: '' }
})

watch(
  () => props.initialValues,
  (values) => {
    form.setValues({ title: values.title || '', content: values.content || '' }, false)
    form.setErrors({})
  },
  { immediate: true, deep: true }
)

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await props.submitForm({ title: values.title, content: values.content })
  } finally {
    formLoading.value = false
  }
})
</script>
