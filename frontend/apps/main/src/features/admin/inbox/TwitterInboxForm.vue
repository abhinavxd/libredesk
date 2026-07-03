<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <div class="box p-4 border-amber-200 bg-amber-50 text-amber-950 dark:bg-amber-950/20 dark:text-amber-100 dark:border-amber-900">
      <div class="flex gap-3">
        <AlertTriangle class="size-5 flex-shrink-0" />
        <div class="space-y-1 text-sm">
          <p class="font-medium">X API access can be billed per read and per send.</p>
          <p class="text-amber-900/80 dark:text-amber-100/80">
            Polling DMs and mentions can each incur reads. Use webhook delivery when your X API access and public callback URL support it.
          </p>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
          <FormControl><Input type="text" v-bind="componentField" /></FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="screen_name">
        <FormItem>
          <FormLabel>X handle</FormLabel>
          <FormControl><Input type="text" placeholder="@support" v-bind="componentField" /></FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="account_user_id">
        <FormItem>
          <FormLabel>Account user ID</FormLabel>
          <FormControl><Input type="text" v-bind="componentField" /></FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="provider">
        <FormItem>
          <FormLabel>Provider</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger><SelectValue placeholder="Select provider" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="official">Official X API</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="base_url">
        <FormItem class="md:col-span-2">
          <FormLabel>Base URL</FormLabel>
          <FormControl><Input type="url" placeholder="https://api.x.com" v-bind="componentField" /></FormControl>
          <FormDescription>Leave blank for the official X API endpoint.</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="delivery_mode">
        <FormItem>
          <FormLabel>Delivery mode</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger><SelectValue placeholder="Select delivery mode" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="polling">Polling</SelectItem>
                <SelectItem value="webhook">Webhook</SelectItem>
                <SelectItem value="filtered_stream">Filtered stream</SelectItem>
                <SelectItem value="activity_stream">X Activity stream</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>
            Non-polling modes do not start poll loops. Configure the callback or stream worker separately.
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="space-y-4">
      <h3 class="font-semibold">Non-polling delivery config</h3>
      <FormField v-slot="{ componentField }" name="filtered_stream_rules_json">
        <FormItem>
          <FormLabel>Filtered stream rules</FormLabel>
          <FormControl>
            <Textarea
              rows="6"
              placeholder='[{"value":"@support -is:retweet","tag":"mentions","enabled":true}]'
              v-bind="componentField"
            />
          </FormControl>
          <FormDescription>
            JSON array of X filtered stream rules. Each rule supports value, tag, id, and enabled.
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <FormField v-slot="{ componentField }" name="webhook_url">
          <FormItem>
            <FormLabel>Webhook URL</FormLabel>
            <FormControl><Input type="url" placeholder="https://example.com/webhook/twitter" v-bind="componentField" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="webhook_id">
          <FormItem>
            <FormLabel>Webhook ID</FormLabel>
            <FormControl><Input type="text" v-bind="componentField" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="webhook_consumer_secret">
          <FormItem>
            <FormLabel>Webhook consumer secret</FormLabel>
            <FormControl><Input type="password" v-bind="componentField" /></FormControl>
            <FormDescription>Used for CRC challenge responses and signature verification.</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="activity_subscription_id">
          <FormItem>
            <FormLabel>X Activity subscription ID</FormLabel>
            <FormControl><Input type="text" v-bind="componentField" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="activity_event_types">
          <FormItem class="md:col-span-2">
            <FormLabel>X Activity event types</FormLabel>
            <FormControl><Input type="text" placeholder="dm,mention,reply" v-bind="componentField" /></FormControl>
            <FormDescription>Comma-separated event labels used by the worker/subscription setup.</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
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
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="ingest_dms">
        <FormItem>
          <SwitchField
            title="Ingest Direct Messages"
            description="Create conversations from private 1:1 DMs."
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="ingest_mentions">
        <FormItem>
          <SwitchField
            title="Ingest public mentions"
            description="Create conversations from public replies and mentions."
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="prompt_tags_on_reply">
        <FormItem>
          <SwitchField
            :title="$t('admin.inbox.promptTagsOnReply')"
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
        </FormItem>
      </FormField>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <FormField v-slot="{ componentField }" name="poll_interval">
        <FormItem>
          <FormLabel>Poll interval</FormLabel>
          <FormControl><Input type="text" placeholder="5m" v-bind="componentField" /></FormControl>
          <FormDescription>Minimum enforced by the backend is 30s.</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="scan_since">
        <FormItem>
          <FormLabel>Initial scan window</FormLabel>
          <FormControl><Input type="text" placeholder="48h" v-bind="componentField" /></FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="space-y-4">
      <h3 class="font-semibold">OAuth credentials</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <FormField v-slot="{ componentField }" name="oauth.client_id">
          <FormItem>
            <FormLabel>{{ $t('globals.terms.clientID') }}</FormLabel>
            <FormControl><Input type="text" v-bind="componentField" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="oauth.client_secret">
          <FormItem>
            <FormLabel>{{ $t('globals.terms.clientSecret') }}</FormLabel>
            <FormControl><Input type="password" v-bind="componentField" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="oauth.access_token">
          <FormItem>
            <FormLabel>Access token</FormLabel>
            <FormControl><Input type="password" v-bind="componentField" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="oauth.refresh_token">
          <FormItem>
            <FormLabel>Refresh token</FormLabel>
            <FormControl><Input type="password" v-bind="componentField" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createTwitterFormSchema } from './twitterFormSchema.js'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import { AlertTriangle } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { AUTH_TYPE_OAUTH2 } from '@/constants/auth.js'

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

const form = useForm({
  validationSchema: computed(() => toTypedSchema(createTwitterFormSchema(t))),
  initialValues: {
    name: '',
    enabled: true,
    csat_enabled: false,
    prompt_tags_on_reply: false,
    account_user_id: '',
    screen_name: '',
    auth_type: AUTH_TYPE_OAUTH2,
    provider: 'official',
    base_url: '',
    delivery_mode: 'polling',
    filtered_stream_rules_json: '[]',
    webhook_id: '',
    webhook_url: '',
    webhook_consumer_secret: '',
    activity_subscription_id: '',
    activity_event_types: '',
    ingest_dms: true,
    ingest_mentions: true,
    poll_interval: '5m',
    scan_since: '48h',
    dm_cursor: '',
    mentions_cursor: '',
    oauth: {
      provider: 'twitter',
      client_id: '',
      client_secret: '',
      access_token: '',
      refresh_token: '',
      expires_at: ''
    }
  }
})

const submitLabel = computed(() => {
  return props.submitLabel || (props.isNewForm ? t('globals.messages.create') : t('globals.messages.save'))
})

const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (!newValues || Object.keys(newValues).length === 0) {
      return
    }
    const cfg = newValues.config || {}
    form.setValues({
      name: newValues.name || '',
      enabled: newValues.enabled ?? true,
      csat_enabled: newValues.csat_enabled ?? false,
      prompt_tags_on_reply: newValues.prompt_tags_on_reply ?? false,
      account_user_id: cfg.account_user_id || '',
      screen_name: cfg.screen_name || '',
      auth_type: cfg.auth_type || AUTH_TYPE_OAUTH2,
      provider: cfg.provider || 'official',
      base_url: cfg.base_url || '',
      delivery_mode: cfg.delivery_mode || 'polling',
      filtered_stream_rules_json: JSON.stringify(cfg.filtered_stream?.rules || [], null, 2),
      webhook_id: cfg.webhook?.id || '',
      webhook_url: cfg.webhook?.url || '',
      webhook_consumer_secret: cfg.webhook?.consumer_secret || '',
      activity_subscription_id: cfg.activity?.subscription_id || '',
      activity_event_types: (cfg.activity?.event_types || []).join(', '),
      ingest_dms: cfg.ingest_dms ?? true,
      ingest_mentions: cfg.ingest_mentions ?? true,
      poll_interval: cfg.poll_interval || '5m',
      scan_since: cfg.scan_since || '48h',
      dm_cursor: cfg.dm_cursor || '',
      mentions_cursor: cfg.mentions_cursor || '',
      oauth: {
        provider: cfg.oauth?.provider || 'twitter',
        client_id: cfg.oauth?.client_id || '',
        client_secret: cfg.oauth?.client_secret || '',
        access_token: cfg.oauth?.access_token || '',
        refresh_token: cfg.oauth?.refresh_token || '',
        expires_at: cfg.oauth?.expires_at || ''
      }
    })
  },
  { deep: true, immediate: true }
)
</script>
