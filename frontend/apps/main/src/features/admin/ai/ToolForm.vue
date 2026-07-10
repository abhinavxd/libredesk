<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('admin.ai.tool.nameHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>{{ t('globals.terms.description') }}</FormLabel>
        <FormControl>
          <Textarea rows="3" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="grid gap-6 md:grid-cols-3">
      <FormField v-slot="{ componentField }" name="url">
        <FormItem class="md:col-span-2">
          <FormLabel>{{ t('globals.terms.url') }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="method">
        <FormItem>
          <FormLabel>{{ t('admin.ai.tool.method') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="m in methods" :key="m" :value="m">{{ m }}</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField }" name="auth_header">
        <FormItem>
          <FormLabel>{{ t('admin.ai.tool.authHeader') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="t('admin.ai.tool.authHeaderPlaceholder')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="auth_value">
        <FormItem>
          <FormLabel>{{ t('admin.ai.tool.authValue') }}</FormLabel>
          <FormControl>
            <Input
              type="password"
              autocomplete="new-password"
              :placeholder="t('admin.ai.tool.authValuePlaceholder')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <FormField v-slot="{ componentField }" name="parameters">
      <FormItem>
        <FormLabel>{{ t('admin.ai.tool.parameters') }}</FormLabel>
        <FormControl>
          <Textarea rows="16" class="font-mono text-sm" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('admin.ai.tool.parametersHint') }}</FormDescription>
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

    <div class="flex justify-end mt-10">
      <Button type="submit" :isLoading="formLoading">
        {{ isEditing ? t('globals.messages.save') : t('globals.messages.create') }}
      </Button>
    </div>
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
import SwitchField from '@shared-ui/components/SwitchField.vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { useI18n } from 'vue-i18n'

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

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
      name: z
        .string({ required_error: t('globals.messages.required') })
        .min(1, { message: t('globals.messages.required') })
        .regex(/^[A-Za-z0-9_-]+$/, { message: t('admin.ai.tool.nameHint') }),
      description: z.string().optional(),
      url: z
        .string({ required_error: t('globals.messages.required') })
        .min(1, { message: t('globals.messages.required') }),
      method: z.string().optional(),
      auth_header: z.string().optional(),
      auth_value: z.string().optional(),
      parameters: z.string().optional(),
      enabled: z.boolean().optional()
    })
  ),
  initialValues: {
    name: '',
    description: '',
    url: '',
    method: 'POST',
    auth_header: '',
    auth_value: '',
    parameters: '',
    enabled: true
  }
})

watch(
  () => props.initialValues,
  (values) => {
    form.setValues({
      name: values.name || '',
      description: values.description || '',
      url: values.url || '',
      method: values.method || 'POST',
      auth_header: values.auth?.header || '',
      auth_value: values.auth?.value || '',
      parameters:
        values.parameters && Object.keys(values.parameters).length
          ? JSON.stringify(values.parameters, null, 2)
          : '',
      enabled: values.enabled ?? true
    })
    form.setErrors({})
  },
  { immediate: true, deep: true }
)

const onSubmit = form.handleSubmit(async (values) => {
  let parameters = {}
  if (values.parameters && values.parameters.trim()) {
    try {
      parameters = JSON.parse(values.parameters)
    } catch {
      form.setFieldError('parameters', t('admin.ai.tool.parametersHint'))
      return
    }
  }

  try {
    formLoading.value = true
    await props.submitForm({
      name: values.name,
      description: values.description || '',
      url: values.url,
      method: values.method || 'POST',
      enabled: !!values.enabled,
      auth: {
        header: values.auth_header || '',
        value: values.auth_value || ''
      },
      parameters
    })
  } finally {
    formLoading.value = false
  }
})
</script>
