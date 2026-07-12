<template>
  <Card>
    <CardHeader>
      <CardTitle>{{ title }}</CardTitle>
      <CardDescription>{{ description }}</CardDescription>
    </CardHeader>
    <CardContent>
      <form @submit="onSubmit" novalidate class="space-y-6 w-full">
        <div class="space-y-2">
          <p class="text-sm font-medium leading-none">{{ t('admin.ai.preset') }}</p>
          <Select v-model="selectedPreset" @update:modelValue="applyPreset">
            <SelectTrigger>
              <SelectValue :placeholder="t('admin.ai.presetCustom')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="p in presets" :key="p.value" :value="p.value">
                {{ p.label }}
              </SelectItem>
            </SelectContent>
          </Select>
          <p class="text-sm text-muted-foreground">{{ t('admin.ai.presetHint') }}</p>
        </div>

        <div class="grid gap-6 md:grid-cols-2">
          <FormField v-slot="{ componentField }" name="model">
            <FormItem>
              <FormLabel>{{ t('globals.terms.model') }}</FormLabel>
              <FormControl>
                <Input type="text" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="base_url">
            <FormItem>
              <FormLabel>{{ t('admin.ai.baseUrl') }}</FormLabel>
              <FormControl>
                <Input type="text" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{ t('admin.ai.baseUrlHint') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="api_key">
            <FormItem>
              <FormLabel>{{ t('admin.ai.apiKey') }}</FormLabel>
              <FormControl>
                <Input
                  type="password"
                  autocomplete="new-password"
                  :placeholder="t('admin.ai.apiKeyPlaceholder')"
                  v-bind="componentField"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <template v-if="showCompletionFields">
            <FormField v-slot="{ componentField }" name="temperature">
              <FormItem>
                <FormLabel>{{ t('admin.ai.temperature') }}</FormLabel>
                <FormControl>
                  <Input type="number" step="0.1" min="0" max="2" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="max_tokens">
              <FormItem>
                <FormLabel>{{ t('admin.ai.maxTokens') }}</FormLabel>
                <FormControl>
                  <Input type="number" min="1" step="1" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="reasoning_effort">
              <FormItem>
                <FormLabel>{{ t('admin.ai.reasoningEffort') }}</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="none" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ t('admin.ai.reasoningEffortHint') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="instructions">
              <FormItem class="md:col-span-2">
                <FormLabel>{{ t('admin.ai.instructions') }}</FormLabel>
                <FormControl>
                  <Textarea rows="4" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ t('admin.ai.instructionsHint') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField, handleChange }" name="vision">
              <FormItem class="md:col-span-2">
                <SwitchField
                  :title="t('admin.ai.vision')"
                  :description="t('admin.ai.visionHint')"
                  :checked="componentField.modelValue"
                  @update:checked="handleChange"
                />
              </FormItem>
            </FormField>
          </template>

          <template v-if="showEmbeddingFields">
            <FormField v-slot="{ componentField }" name="dimensions">
              <FormItem>
                <FormLabel>{{ t('admin.ai.dimensions') }}</FormLabel>
                <FormControl>
                  <Input type="number" min="1" step="1" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </template>
        </div>

        <Button type="submit" :isLoading="formLoading">{{ t('globals.messages.save') }}</Button>
      </form>
    </CardContent>
  </Card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
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
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@shared-ui/components/ui/card/index.js'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const props = defineProps({
  type: { type: String, required: true },
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  showCompletionFields: { type: Boolean, default: false },
  showEmbeddingFields: { type: Boolean, default: false }
})

const { t } = useI18n()
const emitter = useEmitter()
const formLoading = ref(false)
const hasApiKey = ref(false)
const selectedPreset = ref('')

// Sensible starting values shown when a provider has none saved yet.
const fieldDefaults = {
  temperature: props.showCompletionFields ? '0.3' : '',
  maxTokens: props.showCompletionFields ? '1024' : '',
  dimensions: props.showEmbeddingFields ? '1536' : ''
}

// OpenAI-compatible provider presets: pre-fill base URL + model (+ dims for embedding).
const completionPresets = [
  { value: 'openai', label: 'OpenAI', base_url: 'https://api.openai.com/v1', model: 'gpt-4o-mini' },
  {
    value: 'openrouter',
    label: 'OpenRouter',
    base_url: 'https://openrouter.ai/api/v1',
    model: 'openai/gpt-4o-mini'
  },
  {
    value: 'groq',
    label: 'Groq',
    base_url: 'https://api.groq.com/openai/v1',
    model: 'llama-3.3-70b-versatile'
  },
  {
    value: 'together',
    label: 'Together AI',
    base_url: 'https://api.together.xyz/v1',
    model: 'meta-llama/Llama-3.3-70B-Instruct-Turbo'
  },
  {
    value: 'ollama',
    label: 'Ollama (local)',
    base_url: 'http://localhost:11434/v1',
    model: 'llama3.1'
  }
]
const embeddingPresets = [
  {
    value: 'openai-small',
    label: 'OpenAI text-embedding-3-small',
    base_url: 'https://api.openai.com/v1',
    model: 'text-embedding-3-small',
    dimensions: '1536'
  },
  {
    value: 'openai-large',
    label: 'OpenAI text-embedding-3-large',
    base_url: 'https://api.openai.com/v1',
    model: 'text-embedding-3-large',
    dimensions: '3072'
  },
  {
    value: 'ollama',
    label: 'Ollama (local)',
    base_url: 'http://localhost:11434/v1',
    model: 'nomic-embed-text',
    dimensions: '768'
  }
]

const presets = computed(() => (props.showEmbeddingFields ? embeddingPresets : completionPresets))

const applyPreset = (value) => {
  const preset = presets.value.find((p) => p.value === value)
  if (!preset) return
  form.setFieldValue('base_url', preset.base_url)
  form.setFieldValue('model', preset.model)
  if (props.showEmbeddingFields && preset.dimensions != null) {
    form.setFieldValue('dimensions', preset.dimensions)
  }
}

// Numeric fields stay string-or-number (what the input emits); blank = not set.
const numberField = () => z.union([z.string(), z.number()]).optional()
const isBlank = (v) => v === '' || v == null
const inRange = (v, min, max) => {
  const n = Number(v)
  return !Number.isNaN(n) && n >= min && n <= max
}
const isPositiveInt = (v) => {
  const n = Number(v)
  return Number.isInteger(n) && n >= 1
}

const form = useForm({
  validationSchema: toTypedSchema(
    z.object({
      model: z.string({ required_error: t('globals.messages.required') }).min(1, {
        message: t('globals.messages.required')
      }),
      base_url: z.string().optional(),
      api_key: z.string().optional(),
      instructions: z.string().optional(),
      reasoning_effort: z.string().optional(),
      temperature: numberField().refine((v) => isBlank(v) || inRange(v, 0, 2), {
        message: t('admin.ai.temperatureRange')
      }),
      max_tokens: numberField().refine((v) => isBlank(v) || isPositiveInt(v), {
        message: t('admin.ai.positiveNumber')
      }),
      dimensions: numberField().refine((v) => isBlank(v) || isPositiveInt(v), {
        message: t('admin.ai.positiveNumber')
      }),
      vision: z.boolean().optional()
    })
  ),
  initialValues: {
    model: '',
    base_url: '',
    api_key: '',
    instructions: '',
    reasoning_effort: '',
    temperature: fieldDefaults.temperature,
    max_tokens: fieldDefaults.maxTokens,
    dimensions: fieldDefaults.dimensions,
    vision: false
  }
})

onMounted(async () => {
  try {
    const resp = await api.getAIConfig(props.type)
    const data = resp.data.data || {}
    hasApiKey.value = !!data.has_api_key
    // Pass false so loading an unconfigured provider doesn't flash a "Required" error before the admin has typed anything.
    form.setValues(
      {
        model: data.model || '',
        base_url: data.base_url || '',
        api_key: data.api_key || '',
        instructions: data.instructions || '',
        reasoning_effort: data.reasoning_effort || '',
        // Fall back to sensible defaults when the provider has nothing saved (0/absent).
        temperature:
          data.temperature != null ? String(data.temperature) : fieldDefaults.temperature,
        max_tokens: data.max_tokens ? String(data.max_tokens) : fieldDefaults.maxTokens,
        dimensions: data.dimensions ? String(data.dimensions) : fieldDefaults.dimensions,
        vision: !!data.vision
      },
      false
    )
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  const payload = {
    model: values.model,
    base_url: values.base_url || ''
  }
  if (values.api_key) payload.api_key = values.api_key
  if (props.showCompletionFields) {
    payload.instructions = values.instructions || ''
    payload.vision = !!values.vision
    payload.reasoning_effort = (values.reasoning_effort || '').trim()
    if (values.temperature !== '' && values.temperature != null)
      payload.temperature = Number(values.temperature)
    if (values.max_tokens !== '' && values.max_tokens != null)
      payload.max_tokens = Number(values.max_tokens)
  }
  if (props.showEmbeddingFields) {
    if (values.dimensions !== '' && values.dimensions != null)
      payload.dimensions = Number(values.dimensions)
  }

  try {
    formLoading.value = true
    await api.updateAIConfig(props.type, payload)
    hasApiKey.value = hasApiKey.value || !!values.api_key
    form.setFieldValue('api_key', hasApiKey.value ? '•'.repeat(10) : '')
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})
</script>
