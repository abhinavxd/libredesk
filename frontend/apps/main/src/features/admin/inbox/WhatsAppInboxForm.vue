<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <div
      v-if="initialValues?.token_invalid"
      class="box border-destructive/40 bg-destructive/5 p-3 text-sm flex items-start gap-2"
    >
      <TriangleAlert class="size-4 mt-0.5 text-destructive shrink-0" />
      <span>{{ $t('admin.inbox.whatsapp.tokenInvalid') }}</span>
    </div>

    <!-- Basic Fields -->
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Toggle Fields -->
    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="$t('globals.terms.enabled')"
          :description="$t('admin.inbox.enabled.description')"
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

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
      <FormItem>
        <SwitchField
          :title="$t('admin.inbox.csatSurveys')"
          :description="$t('admin.inbox.csatSurveys.description_1')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
      <p class="!mt-2 text-muted-foreground text-xs flex items-start gap-1.5">
        <Lightbulb class="size-4" />
        <span>{{ $t('admin.inbox.csatSurveys.description_3') }}</span>
      </p>
    </FormField>

    <div v-show="csatEnabled" class="box p-4 space-y-4">
      <div>
        <h3 class="font-semibold">{{ $t('admin.inbox.whatsapp.csatTemplate') }}</h3>
        <p class="!mt-1 text-sm text-muted-foreground flex items-start gap-1.5">
          <Lightbulb class="size-4 mt-0.5 shrink-0" />
          <span>{{ $t('admin.inbox.whatsapp.csatTemplate.description') }}</span>
        </p>
      </div>

      <FormField v-slot="{ componentField }" name="config.csat_template_body">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.whatsapp.csatTemplateBody') }}</FormLabel>
          <FormControl>
            <Textarea rows="3" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.whatsapp.csatTemplateBody.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <div class="grid grid-cols-2 gap-4">
        <FormField v-slot="{ componentField }" name="config.csat_template_button_text">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.csatTemplateButtonText') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.csat_template_language">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.csatTemplateLanguage') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="en_US" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.whatsapp.csatTemplateLanguage.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <!-- Meta Cloud API Credentials -->
    <div class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.whatsapp.metaCredentials') }}</h3>
      <p class="text-sm text-muted-foreground flex items-start gap-1.5">
        <Lightbulb class="size-4 mt-0.5 shrink-0" />
        <span>{{ $t('admin.inbox.whatsapp.metaCredentials.description') }}</span>
      </p>

      <div class="grid grid-cols-2 gap-4">
        <FormField v-slot="{ componentField }" name="config.phone_number_id">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.phoneNumberID') }}</FormLabel>
            <FormControl>
              <Input
                type="text"
                :placeholder="t('admin.inbox.whatsapp.phoneNumberID.placeholder')"
                v-bind="componentField"
              />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.whatsapp.phoneNumberID.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.waba_id">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.wabaID') }}</FormLabel>
            <FormControl>
              <Input
                type="text"
                :placeholder="t('admin.inbox.whatsapp.wabaID.placeholder')"
                v-bind="componentField"
              />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.whatsapp.wabaID.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <FormField v-slot="{ componentField }" name="config.access_token">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.accessToken') }}</FormLabel>
            <FormControl>
              <Input type="password" placeholder="••••••••" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.whatsapp.accessToken.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.app_secret">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.appSecret') }}</FormLabel>
            <FormControl>
              <Input type="password" placeholder="••••••••" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.whatsapp.appSecret.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <FormField v-slot="{ componentField }" name="config.api_version">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.apiVersion') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="v21.0" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.whatsapp.apiVersion.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="reopen_window_hours">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.whatsapp.reopenWindow') }}</FormLabel>
            <FormControl>
              <Input type="number" min="0" placeholder="48" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ $t('admin.inbox.whatsapp.reopenWindow.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <!-- Webhook Configuration -->
    <div class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.whatsapp.webhook') }}</h3>
      <p class="text-sm text-muted-foreground flex items-start gap-1.5">
        <Lightbulb class="size-4 mt-0.5 shrink-0" />
        <span>{{ $t('admin.inbox.whatsapp.webhook.description') }}</span>
      </p>

      <FormField v-slot="{ componentField }" name="config.webhook_verify_token">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.whatsapp.verifyToken') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="t('admin.inbox.whatsapp.verifyToken.placeholder')"
              v-bind="componentField"
            />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.whatsapp.verifyToken.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <!-- Computed webhook URL from the backend; only set once the inbox has been saved. -->
      <div v-if="webhookURL" class="space-y-1">
        <label class="text-sm font-medium">{{ $t('admin.inbox.whatsapp.webhookURL') }}</label>
        <div class="flex items-center gap-2">
          <Input :model-value="webhookURL" readonly class="font-mono text-xs" />
          <Button type="button" variant="outline" size="sm" @click="copyWebhookURL">
            {{ $t('globals.terms.copy') }}
          </Button>
        </div>
        <p class="text-xs text-muted-foreground">
          {{ $t('admin.inbox.whatsapp.webhookURL.description') }}
        </p>
      </div>
      <p v-else class="text-xs text-muted-foreground">
        {{ $t('admin.inbox.whatsapp.webhookURL.afterSave') }}
      </p>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { watch, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import {
  createFormSchema,
  DEFAULT_CSAT_TEMPLATE_LANGUAGE,
  DEFAULT_CSAT_TEMPLATE_BODY,
  DEFAULT_CSAT_TEMPLATE_BUTTON_TEXT
} from './whatsappFormSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Lightbulb, TriangleAlert } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isNewForm: {
    type: Boolean,
    default: false
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const emitter = useEmitter()

const webhookURL = computed(() => props.initialValues?.webhook_url || '')

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? t('globals.messages.create') : t('globals.messages.save'))
  )
})

const form = useForm({
  validationSchema: computed(() => toTypedSchema(createFormSchema(t))),
  initialValues: {
    name: '',
    enabled: true,
    csat_enabled: false,
    prompt_tags_on_reply: false,
    reopen_window_hours: 48,
    config: {
      phone_number_id: '',
      waba_id: '',
      access_token: '',
      app_secret: '',
      webhook_verify_token: '',
      api_version: 'v21.0',
      csat_template_language: DEFAULT_CSAT_TEMPLATE_LANGUAGE,
      csat_template_body: DEFAULT_CSAT_TEMPLATE_BODY,
      csat_template_button_text: DEFAULT_CSAT_TEMPLATE_BUTTON_TEXT
    }
  }
})

const csatEnabled = computed(() => form.values.csat_enabled)

const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
})

const copyWebhookURL = async () => {
  try {
    await navigator.clipboard.writeText(webhookURL.value)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.copied')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.somethingWentWrong')
    })
  }
}

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    // An unset reopen window is off (0), not the new-inbox default of 48.
    form.setValues({
      ...newValues,
      reopen_window_hours: newValues.reopen_window_hours ?? 0,
      config: {
        ...(newValues.config || {}),
        csat_template_language:
          newValues.config?.csat_template_language || DEFAULT_CSAT_TEMPLATE_LANGUAGE,
        csat_template_body: newValues.config?.csat_template_body || DEFAULT_CSAT_TEMPLATE_BODY,
        csat_template_button_text:
          newValues.config?.csat_template_button_text || DEFAULT_CSAT_TEMPLATE_BUTTON_TEXT
      }
    })
  },
  { deep: true, immediate: true }
)
</script>
