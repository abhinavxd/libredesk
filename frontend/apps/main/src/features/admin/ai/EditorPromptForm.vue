<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="title">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Title', 'Titles', 1) }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            :placeholder="i18n._('Make it friendlier')"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="content">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Content', 'Contents', 1) }}</FormLabel>
        <FormControl>
          <Textarea
            class="min-h-[240px] max-h-[60vh]"
            :placeholder="i18n._('Rewrite the text in a warm, friendly tone while keeping the original meaning.')"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <DialogFooter class="mt-10">
      <Button type="submit" :isLoading="formLoading">
        {{ isEditing ? i18n._('Save') : i18n._('Create') }}
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
import { useI18n } from '@shared-ui/i18n'

const props = defineProps({
  initialValues: { type: Object, default: () => ({}) },
  isEditing: { type: Boolean, default: false },
  submitForm: { type: Function, required: true }
})

const i18n = useI18n()
const formLoading = ref(false)

const form = useForm({
  validationSchema: toTypedSchema(
    z.object({
      title: z
        .string({ required_error: i18n._('{name} Required') })
        .min(1, { message: i18n._('{name} Required') })
        .max(140),
      content: z
        .string({ required_error: i18n._('{name} Required') })
        .min(1, { message: i18n._('{name} Required') })
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
