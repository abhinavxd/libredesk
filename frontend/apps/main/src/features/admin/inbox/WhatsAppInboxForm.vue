<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <div
      v-if="initialValues?.token_invalid"
      class="box border-destructive/40 bg-destructive/5 p-3 text-sm flex items-start gap-2"
    >
      <TriangleAlert class="size-4 mt-0.5 text-destructive shrink-0" />
      <span>{{ i18n._('Meta rejected this inbox\'\'s access token. Messages and media will fail to send or download until you paste a new token below and save.') }}</span>
    </div>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
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

    <FormField v-slot="{ componentField, handleChange }" name="prompt_tags_on_reply">
      <FormItem>
        <SwitchField
          :title="i18n._('Prompt to tag before replying')"
          :description="i18n._('Warn agents before they send a reply on a conversation that has no tags. Agents can still send without tagging by confirming the prompt.')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
      <FormItem>
        <SwitchField
          :title="i18n._('CSAT Surveys')"
          :description="i18n._('Send customer satisfaction surveys when a conversation is marked as resolved.')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
      <p class="!mt-2 text-muted-foreground text-sm flex items-start gap-1.5">
        <Lightbulb class="size-4 mt-0.5 shrink-0" />
        <span>{{ i18n._('CSAT surveys are only sent once per conversation.') }}</span>
      </p>
    </FormField>

    <div class="max-w-xl">
      <FormField v-slot="{ componentField }" name="reopen_window_hours">
        <FormItem>
          <FormLabel>{{ i18n._('Reopen window (hours)') }}</FormLabel>
          <FormControl>
            <Input type="number" min="0" placeholder="48" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ i18n._('If a contact replies within this many hours of a conversation being resolved, it reopens instead of starting a new one. Set to 0 to always start a new conversation.') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div v-show="csatEnabled" class="box p-4 space-y-4">
      <div>
        <h3 class="ui-heading">{{ i18n._('CSAT survey template') }}</h3>
        <p class="!mt-1 ui-description flex items-start gap-1.5">
          <Lightbulb class="size-4 mt-0.5 shrink-0" />
          <span>{{ i18n._('This message is registered with Meta as a template and sent when a WhatsApp conversation is resolved.') }}</span>
        </p>
      </div>

      <FormField v-slot="{ componentField }" name="config.csat_template_body">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Message', 'Messages', 1) }}</FormLabel>
          <FormControl>
            <Textarea rows="3" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ i18n._('Shown above the rating button. Meta reviews template content before approving it.') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormGrid>
        <FormField v-slot="{ componentField }" name="config.csat_template_button_text">
          <FormItem>
            <FormLabel>{{ i18n._('Button text') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.csat_template_language">
          <FormItem>
            <FormLabel>{{ i18n.ngettext('Language', 'Languages', 1) }}</FormLabel>
            <FormControl>
              <ComboBox
                v-bind="componentField"
                :items="WHATSAPP_TEMPLATE_LANGUAGES"
                :placeholder="i18n._('Search')"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </FormGrid>
    </div>

    <div class="box p-4 space-y-4">
      <h3 class="ui-heading">{{ i18n._('Meta Cloud API credentials') }}</h3>
      <FormGrid>
        <FormField v-slot="{ componentField }" name="config.phone_number_id">
          <FormItem>
            <FormLabel>{{ i18n._('Phone number ID') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ i18n._('Found on the WhatsApp API Setup page in your Meta app.') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.waba_id">
          <FormItem>
            <FormLabel>{{ i18n._('WhatsApp Business Account ID') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </FormGrid>

      <FormGrid>
        <FormField v-slot="{ componentField }" name="config.access_token">
          <FormItem>
            <FormLabel>{{ i18n._('Access token') }}</FormLabel>
            <FormControl>
              <Input type="password" placeholder="••••••••" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ i18n._('Use a permanent System User token. Temporary tokens expire in 24 hours.') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.app_secret">
          <FormItem>
            <FormLabel>{{ i18n._('App secret') }}</FormLabel>
            <FormControl>
              <Input type="password" placeholder="••••••••" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ i18n._('Found under App settings \u003e Basic in your Meta app.') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </FormGrid>

      <div class="max-w-xl">
        <FormField v-slot="{ componentField }" name="config.api_version">
          <FormItem>
            <FormLabel>{{ i18n._('API version') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="v26.0" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div class="box p-4 space-y-4">
      <h3 class="ui-heading">{{ i18n.ngettext('Webhook', 'Webhooks', 1) }}</h3>
      <p v-if="webhookURL" class="ui-description flex items-start gap-1.5">
        <Lightbulb class="size-4 mt-0.5 shrink-0" />
        <span>{{ i18n._('Paste this URL and the verify token into Meta \u003e Configuration \u003e Webhooks, then subscribe to the messages and message_template_status_update fields.') }}</span>
      </p>

      <div class="max-w-xl">
        <FormField v-slot="{ componentField }" name="config.webhook_verify_token">
          <FormItem>
            <FormLabel>{{ i18n._('Webhook verify token') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="componentField" />
            </FormControl>
            <FormDescription>
              {{ i18n._('Any random string. Paste the same value into Meta\'\'s webhook configuration.') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <div v-if="webhookURL" class="space-y-2">
          <Label class="text-muted-foreground">{{ i18n._('Callback URL') }}</Label>
          <div class="flex items-center gap-2">
            <Input :model-value="webhookURL" readonly class="font-mono text-xs" />
            <CopyButton :text="webhookURL" />
          </div>
        </div>
      </div>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
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
import { Label } from '@shared-ui/components/ui/label'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import { WHATSAPP_TEMPLATE_LANGUAGES } from '@main/features/admin/whatsapp/whatsappLanguages.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import CopyButton from '@/components/button/CopyButton.vue'
import { Lightbulb, TriangleAlert } from 'lucide-vue-next'
import { useI18n } from '@shared-ui/i18n'

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

const i18n = useI18n()

const webhookURL = computed(() => props.initialValues?.webhook_url || '')

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? i18n._('Create') : i18n._('Save'))
  )
})

const form = useForm({
  validationSchema: computed(() => toTypedSchema(createFormSchema(i18n))),
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
      api_version: 'v26.0',
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

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    // An unset reopen window is off (0), not the new-inbox default of 48.
    form.setValues(
      {
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
      },
      false
    )
  },
  { deep: true, immediate: true }
)
</script>
