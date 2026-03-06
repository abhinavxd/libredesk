<template>
  <form @submit="onSubmit" class="space-y-6">
    <!-- Endpoint URL -->
    <FormField v-slot="{ componentField }" name="endpoint_url">
      <FormItem>
        <FormLabel>{{ $t('admin.ai.endpointUrl') }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            placeholder="https://api.openai.com/v1/chat/completions"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ $t('admin.ai.endpointUrl.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Model -->
    <FormField v-slot="{ componentField }" name="model">
      <FormItem>
        <FormLabel>{{ $t('admin.ai.model') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="gpt-4o-mini" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ $t('admin.ai.model.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- API Key -->
    <FormField v-slot="{ componentField }" name="api_key">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.apiKey') }}</FormLabel>
        <FormControl>
          <Input type="password" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          <span v-if="hasAPIKey" class="flex items-center gap-1.5 mb-1">
            <span class="inline-block w-2 h-2 rounded-full bg-green-500"></span>
            {{ $t('admin.ai.apiKeyConfigured') }}
          </span>
          {{ $t('admin.ai.apiKey.settingsDescription') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading">
      {{ $t('globals.messages.save') }}
    </Button>
  </form>
</template>

<script setup>
import { watch, ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

const isLoading = ref(false)
const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  }
})

const hasAPIKey = computed(() => props.initialValues?.has_api_key || false)

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const onSubmit = form.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    await props.submitForm(values)
  } finally {
    isLoading.value = false
  }
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    if (newValues) {
      form.setValues({
        endpoint_url: newValues.endpoint_url || '',
        model: newValues.model || '',
        api_key: ''
      })
    }
  },
  { deep: true, immediate: true }
)
</script>
