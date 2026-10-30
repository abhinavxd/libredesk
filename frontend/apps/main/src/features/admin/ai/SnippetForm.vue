<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="title">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Title', 'Titles', 1) }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="i18n._('Refund policy')" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="content">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Content', 'Contents', 1) }}</FormLabel>
        <FormControl>
          <Textarea
            class="min-h-[420px] max-h-[60vh]"
            :placeholder="i18n._('We offer refunds within 30 days of purchase. To request one, ...')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ i18n._('The assistant searches this text to answer customer questions, so write it as a self-contained answer rather than a fragment.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="i18n._('Enabled')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
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
import SwitchField from '@shared-ui/components/SwitchField.vue'
import {
  FormControl,
  FormDescription,
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
      title: z.string({ required_error: i18n._('{name} Required') }).min(1, {
        message: i18n._('{name} Required')
      }),
      content: z.string({ required_error: i18n._('{name} Required') }).min(1, {
        message: i18n._('{name} Required')
      }),
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
    form.setValues(
      {
        title: values.title || '',
        content: values.content || '',
        enabled: values.enabled ?? true
      },
      false
    )
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
