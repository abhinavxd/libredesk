<template>
  <form @submit="onSubmit" class="space-y-6 w-full max-w-lg">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="My Telegram Bot" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          {{ $t('admin.inbox.telegram.name.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="bot_token">
      <FormItem>
        <FormLabel>{{ $t('admin.inbox.telegram.botToken') }}</FormLabel>
        <FormControl>
          <Input type="password" placeholder="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          {{ $t('admin.inbox.telegram.botToken.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="bot_name">
      <FormItem>
        <FormLabel>{{ $t('admin.inbox.telegram.botName') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="@my_support_bot" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          {{ $t('admin.inbox.telegram.botName.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="$t('globals.terms.enabled')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
      <FormItem>
        <SwitchField
          :title="$t('admin.inbox.csatSurveys')"
          :description="$t('admin.inbox.csatSurveys.description_1')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="prompt_tags_on_reply">
      <FormItem>
        <SwitchField
          :title="$t('admin.inbox.promptTagsOnReply')"
          :description="$t('admin.inbox.promptTagsOnReply.description')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <!-- Webhook URL info (shown only in edit mode) -->
    <div v-if="!isNewForm && webhookUrl" class="rounded-md border p-4 space-y-2">
      <p class="text-sm font-medium">{{ $t('admin.inbox.telegram.webhookUrl') }}</p>
      <div class="flex items-center gap-2">
        <code class="text-xs bg-muted px-2 py-1 rounded flex-1 break-all">{{ webhookUrl }}</code>
        <Button type="button" variant="outline" size="sm" @click="copyWebhookUrl">
          {{ $t('globals.terms.copy') }}
        </Button>
      </div>
      <p class="text-xs text-muted-foreground">
        {{ $t('admin.inbox.telegram.webhookUrl.description') }}
      </p>
    </div>

    <Button type="submit" :disabled="isLoading">
      <Loader2 v-if="isLoading" class="w-4 h-4 mr-2 animate-spin" />
      {{ isNewForm ? $t('globals.messages.create') : $t('globals.messages.save') }}
    </Button>
  </form>
</template>

<script setup>
import { computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Loader2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'

const { t } = useI18n()
const emitter = useEmitter()

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  isLoading: {
    type: Boolean,
    default: false
  },
  isNewForm: {
    type: Boolean,
    default: false
  }
})

const formSchema = toTypedSchema(
  z.object({
    name: z.string().min(1, t('globals.messages.required')),
    bot_token: z.string().min(1, t('globals.messages.required')),
    bot_name: z.string().optional().default(''),
    enabled: z.boolean().default(true),
    csat_enabled: z.boolean().default(false),
    prompt_tags_on_reply: z.boolean().default(false)
  })
)

const form = useForm({
  validationSchema: formSchema,
  initialValues: {
    name: props.initialValues?.name || '',
    bot_token: props.initialValues?.config?.bot_token || '',
    bot_name: props.initialValues?.config?.bot_name || '',
    enabled: props.initialValues?.enabled ?? true,
    csat_enabled: props.initialValues?.csat_enabled ?? false,
    prompt_tags_on_reply: props.initialValues?.prompt_tags_on_reply ?? false
  }
})

const webhookUrl = computed(() => {
  if (props.initialValues?.id) {
    const baseUrl = window.location.origin
    return `${baseUrl}/api/v1/inboxes/telegram/${props.initialValues.id}/webhook`
  }
  return ''
})

const copyWebhookUrl = async () => {
  try {
    await navigator.clipboard.writeText(webhookUrl.value)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.copied')
    })
  } catch (error) {
    console.error('Failed to copy:', error)
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
})
</script>
