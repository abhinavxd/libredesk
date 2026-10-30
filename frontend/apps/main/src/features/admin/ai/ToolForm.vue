<template>
  <form @submit="onSubmit" novalidate class="tool-form space-y-6 w-full">
    <FormGrid class="tool-connection-grid">
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ i18n._('Letters, numbers, hyphen and underscore only, up to 64 characters.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
      <FormField v-slot="{ componentField }" name="url">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('URL', 'URLs', 1) }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              placeholder="https://api.example.com/orders/lookup"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="method">
        <FormItem>
          <FormLabel>{{ i18n._('Method') }}</FormLabel>
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
    </FormGrid>

    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Description', 'Descriptions', 1) }}</FormLabel>
        <FormControl>
          <Textarea
            rows="3"
            :placeholder="i18n._('Looks up an order\'\'s status and delivery date by order ID.')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ i18n._('The model reads this to decide when to call the tool, so describe what it does and when it should be used.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange, meta }" name="headers">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Header', 'Headers', 2, {}, 'HTTP request headers') }}</FormLabel>
        <FormControl>
          <ToolHeadersField
            :modelValue="componentField.modelValue"
            @update:modelValue="(v) => handleChange(v, meta.validated)"
          />
        </FormControl>
        <FormDescription>{{ i18n._('Sent with every request to this tool\'\'s URL. Values are encrypted at rest.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="parameters">
      <FormItem>
        <div class="flex items-center justify-between">
          <FormLabel>{{ i18n._('Parameters (JSON schema)') }}</FormLabel>
          <Button
            v-if="!componentField.modelValue"
            type="button"
            variant="ghost"
            size="sm"
            @click="insertParametersExample"
          >
            {{ i18n._('Insert example') }}
          </Button>
        </div>
        <FormControl>
          <Textarea
            rows="16"
            class="font-mono text-sm"
            :placeholder="parametersPlaceholder"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ i18n._('Arguments the model can pass when calling this tool. Leave blank for a single text input.') }}</FormDescription>
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

    <FormField v-slot="{ componentField }" name="requires_verification">
      <FormItem>
        <SwitchField
          :title="i18n._('Require verified contact')"
          :description="i18n._('The assistant emails the customer a one-time code and runs this tool only after they enter it. Copilot and Generate Reply run it without the code, since a human agent asks for it there.')"
          :checked="componentField.modelValue"
          @update:checked="onToggleVerification"
        />
      </FormItem>
    </FormField>

    <fieldset class="min-w-0">
      <legend class="ui-heading mb-4">
        {{ i18n._('Agent access') }}
      </legend>
      <div class="space-y-5">
        <FormField v-slot="{ componentField, handleChange }" name="copilot_enabled">
          <FormItem>
            <SwitchField
              :title="i18n._('Available in Copilot')"
              :checked="componentField.modelValue"
              @update:checked="handleChange"
            />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleChange }" name="generate_reply_enabled">
          <FormItem>
            <SwitchField
              :title="i18n._('Available in Generate Reply')"
              :checked="componentField.modelValue"
              @update:checked="handleChange"
            />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleChange }" name="requires_agent_approval">
          <FormItem>
            <SwitchField
              :title="i18n._('Require agent approval')"
              :description="i18n._('The tool pauses before it runs, and an agent approves or declines it. Keep this on for tools that change data.')"
              :checked="componentField.modelValue"
              :disabled="!hasAgentSurface"
              @update:checked="handleChange"
            />
          </FormItem>
        </FormField>
      </div>
    </fieldset>

    <div class="flex justify-end">
      <Button type="submit" :isLoading="formLoading">
        {{ isEditing ? i18n._('Save') : i18n._('Create') }}
      </Button>
    </div>
  </form>

  <AlertDialog :open="confirmTurnOffOpen" @update:open="onConfirmDialogOpenChange">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ i18n._('Turn off customer authentication?') }}</AlertDialogTitle>
        <AlertDialogDescription as="div" class="space-y-3">
          <p>{{ i18n._('With verification on, Libredesk emails the customer a one-time code and runs this tool only after they enter it. Turning it off lets anyone in the chat trigger the tool.') }}</p>
          <p>{{ i18n._('If the tool accesses sensitive data or performs actions, your endpoint must authenticate the customer itself. Every request carries an X-Libredesk-Contact-Verified header.') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/ai#writing-a-custom-tool"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style"
          >
            {{ i18n._('Learn more') }}
          </a>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>
          {{ i18n._('Keep authentication on') }}
        </AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="confirmTurnOff">
          {{ i18n._('Turn off') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { computed, ref, watch } from 'vue'
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
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog/index.js'
import { useI18n } from '@shared-ui/i18n'

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

const i18n = useI18n()
const formLoading = ref(false)

const form = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z
        .string({ required_error: i18n._('{name} Required') })
        .min(1, { message: i18n._('{name} Required') })
        .max(64, { message: i18n._('Letters, numbers, hyphen and underscore only, up to 64 characters.') })
        .regex(/^[A-Za-z0-9_-]+$/, { message: i18n._('Letters, numbers, hyphen and underscore only, up to 64 characters.') }),
      description: z.string().optional(),
      url: z
        .string({ required_error: i18n._('{name} Required') })
        .min(1, { message: i18n._('{name} Required') }),
      method: z.string().optional(),
      headers: z
        .array(z.object({ key: z.string(), value: z.string() }))
        .optional()
        .refine((headers) => (headers ?? []).every((h) => !!h.key.trim() === !!h.value.trim()), {
          message: i18n._('Enter both a name and a value for every header, or remove the incomplete one.')
        }),
      parameters: z.string().optional(),
      enabled: z.boolean().optional(),
      requires_verification: z.boolean(),
      copilot_enabled: z.boolean(),
      generate_reply_enabled: z.boolean(),
      requires_agent_approval: z.boolean()
    })
  ),
  initialValues: {
    name: '',
    description: '',
    url: '',
    method: 'POST',
    headers: [],
    parameters: '',
    enabled: true,
    requires_verification: true,
    copilot_enabled: false,
    generate_reply_enabled: false,
    requires_agent_approval: true
  }
})

const hasAgentSurface = computed(
  () => form.values.copilot_enabled || form.values.generate_reply_enabled
)
const insertParametersExample = () => {
  form.setFieldValue('parameters', parametersPlaceholder, false)
}

const confirmTurnOffOpen = ref(false)

const onToggleVerification = (checked) => {
  if (!checked) {
    confirmTurnOffOpen.value = true
    return
  }
  form.setFieldValue('requires_verification', true, false)
}

const confirmTurnOff = () => {
  form.setFieldValue('requires_verification', false, false)
  confirmTurnOffOpen.value = false
}

const onConfirmDialogOpenChange = (open) => {
  confirmTurnOffOpen.value = open
}

watch(
  () => props.initialValues,
  (values) => {
    form.setValues(
      {
        name: values.name || '',
        description: values.description || '',
        url: values.url || '',
        method: values.method || 'POST',
        headers: values.auth?.headers || [],
        parameters:
          values.parameters && Object.keys(values.parameters).length
            ? JSON.stringify(values.parameters, null, 2)
            : '',
        enabled: values.enabled ?? true,
        requires_verification: values.requires_verification ?? true,
        copilot_enabled: values.copilot_enabled ?? false,
        generate_reply_enabled: values.generate_reply_enabled ?? false,
        requires_agent_approval: values.requires_agent_approval ?? true
      },
      false
    )
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
      form.setFieldError('parameters', i18n._('Invalid JSON. Check for a missing comma, quote, or bracket.'))
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
      requires_verification: values.requires_verification !== false,
      copilot_enabled: !!values.copilot_enabled,
      generate_reply_enabled: !!values.generate_reply_enabled,
      requires_agent_approval: values.requires_agent_approval !== false,
      auth: { headers },
      parameters
    })
  } finally {
    formLoading.value = false
  }
})
</script>

<style scoped>
.tool-form {
  container-type: inline-size;
}

.tool-connection-grid.form-grid {
  grid-template-columns: minmax(0, 1fr);
}

@container (min-width: 40rem) {
  .tool-connection-grid.form-grid {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1.75fr) minmax(0, 1fr);
  }
}
</style>
