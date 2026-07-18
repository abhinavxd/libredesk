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
          <Textarea
            rows="3"
            :placeholder="t('admin.ai.tool.descriptionPlaceholder')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ t('admin.ai.tool.descriptionHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="grid gap-6 md:grid-cols-3">
      <FormField v-slot="{ componentField }" name="url">
        <FormItem class="md:col-span-2">
          <FormLabel>{{ t('globals.terms.url') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="https://api.example.com/orders/lookup" v-bind="componentField" />
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

    <FormField v-slot="{ componentField, handleChange }" name="headers">
      <FormItem>
        <FormLabel>{{ t('admin.ai.tool.headers') }}</FormLabel>
        <FormControl>
          <ToolHeadersField :modelValue="componentField.modelValue" @update:modelValue="handleChange" />
        </FormControl>
        <FormDescription>{{ t('admin.ai.tool.headersHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="parameters">
      <FormItem>
        <FormLabel>{{ t('admin.ai.tool.parameters') }}</FormLabel>
        <FormControl>
          <Textarea
            rows="16"
            class="font-mono text-sm"
            :placeholder="parametersPlaceholder"
            v-bind="componentField"
          />
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
import ToolHeadersField from '@/features/admin/ai/ToolHeadersField.vue'
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

const methods = ['GET', 'POST']

const parametersPlaceholder = `{
  "type": "object",
  "properties": {
    "order_id": { "type": "string", "description": "The order ID to look up" }
  },
  "required": ["order_id"]
}`

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
      headers: z
        .array(z.object({ key: z.string(), value: z.string() }))
        .optional()
        .refine((headers) => (headers ?? []).every((h) => !!h.key.trim() === !!h.value.trim()), {
          message: t('admin.ai.tool.headersInvalid')
        }),
      parameters: z.string().optional(),
      enabled: z.boolean().optional()
    })
  ),
  initialValues: {
    name: '',
    description: '',
    url: '',
    method: 'POST',
    headers: [],
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
      headers: values.auth?.headers || [],
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
      form.setFieldError('parameters', t('admin.ai.tool.parametersInvalid'))
      return
    }
  }

  const headers = (values.headers || [])
    .map((h) => ({ key: h.key.trim(), value: h.value.trim() }))
    .filter((h) => h.key && h.value)

  try {
    formLoading.value = true
    await props.submitForm({
      name: values.name,
      description: values.description || '',
      url: values.url,
      method: values.method || 'POST',
      enabled: !!values.enabled,
      auth: { headers },
      parameters
    })
  } finally {
    formLoading.value = false
  }
})
</script>
