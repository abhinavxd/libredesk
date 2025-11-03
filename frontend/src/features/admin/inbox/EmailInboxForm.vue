<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <!-- Basic Fields -->
    <FormField v-if="showFormFields" v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormDescription> {{ $t('admin.inbox.name.description') }} </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-if="showFormFields" v-slot="{ componentField }" name="from">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.fromEmailAddress') }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            :placeholder="t('admin.inbox.fromEmailAddress.placeholder')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>
          {{ $t('admin.inbox.fromEmailAddress.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Toggle Fields -->
    <FormField v-if="showFormFields" v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ $t('globals.terms.enabled') }}</FormLabel>
          <FormDescription>{{ $t('admin.inbox.enabled.description') }}</FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <FormField
      v-if="showFormFields"
      v-slot="{ componentField, handleChange }"
      name="csat_enabled"
    >
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ $t('admin.inbox.csatSurveys') }}</FormLabel>
          <FormDescription>
            {{ $t('admin.inbox.csatSurveys.description_1') }}<br />
            {{ $t('admin.inbox.csatSurveys.description_2') }}
          </FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <FormField v-if="setupMethod" v-slot="{ componentField }" name="auth_type">
      <FormItem>
        <FormControl>
          <Input
            type="hidden"
            :value="setupMethod === 'manual' ? AUTH_TYPE_PASSWORD : AUTH_TYPE_OAUTH2"
            v-bind="componentField"
          />
        </FormControl>
      </FormItem>
    </FormField>

    <!-- Setup Method Selection -->
    <div v-show="!isOAuthInbox && setupMethod === null" class="space-y-4">
      <div class="space-y-2">
        <h3 class="font-semibold text-lg">Choose Setup Method</h3>
        <p class="text-sm text-muted-foreground">
          Select how you want to connect your email account
        </p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MenuCard
          title="Google"
          subTitle="Connect with Google Workspace or Gmail"
          :icon="GoogleIcon"
          @click="connectWithGoogle()"
        />
        <MenuCard
          title="Microsoft"
          subTitle="Connect with Microsoft 365 or Outlook"
          :icon="MicrosoftIcon"
          @click="connectWithMicrosoft()"
        />
        <MenuCard
          title="Other Provider"
          subTitle="Configure IMAP and SMTP manually"
          :icon="Mail"
          @click="setupMethod = 'manual'"
        />
      </div>
    </div>

    <!-- OAuth Connected Status -->
    <div
      v-show="isOAuthInbox"
      class="box p-4 bg-green-50 dark:bg-green-950/20 border-green-200 dark:border-green-800"
    >
      <div class="flex items-start justify-between">
        <div class="flex items-center space-x-3 flex-1">
          <CheckCircle2 class="w-5 h-5 text-green-600 flex-shrink-0" />
          <div class="flex-1">
            <p class="font-semibold text-green-900 dark:text-green-100">
              Connected via OAuth - {{ oauthProvider }}
            </p>
            <p class="text-sm text-green-700 dark:text-green-300">{{ oauthEmail }}</p>
            <p
              v-show="oauthClientId"
              class="text-xs text-green-600 dark:text-green-400 font-mono mt-1"
            >
              Client ID: {{ oauthClientId.substring(0, 20) }}...{{ oauthClientId.slice(-8) }}
            </p>
          </div>
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          @click="reconnectOAuth"
          :disabled="isSubmittingOAuth"
          class="ml-2 flex-shrink-0"
        >
          <RefreshCw class="w-4 h-4 mr-1" />
          Reconnect
        </Button>
      </div>
    </div>

    <!-- OAuth IMAP Configuration -->
    <div v-show="isOAuthInbox" class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.imapConfig') }}</h3>

      <FormField v-slot="{ componentField }" name="imap.mailbox">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.mailbox') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="INBOX" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.mailbox.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.read_interval">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.imapScanInterval') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="1m" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.imapScanInterval.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.scan_inbox_since">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.imapScanInboxSince') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="48h" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.imapScanInboxSince.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <!-- OAuth SMTP Configuration -->
    <div v-show="isOAuthInbox" class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.smtpConfig') }}</h3>

      <FormField v-slot="{ componentField }" name="smtp.max_conns">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.maxConnections') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="10" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.maxConnections.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.max_msg_retries">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.maxRetries') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="3" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ $t('admin.inbox.maxRetries.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.idle_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.idleTimeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="25s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.idleTimeout.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.wait_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.waitTimeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="60s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.waitTimeout.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <!-- IMAP Section -->
    <div v-show="!isOAuthInbox && setupMethod === 'manual'" class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.imapConfig') }}</h3>

      <FormField v-slot="{ componentField }" name="imap.host">
        <FormItem>
          <FormLabel>Host</FormLabel>
          <FormControl>
            <Input type="text" placeholder="imap.gmail.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.port">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.port') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="993" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.mailbox">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.mailbox') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="INBOX" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.mailbox.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.username">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.username') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="inbox@example.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.password">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.password') }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="••••••••" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.tls_type">
        <FormItem>
          <FormLabel>TLS</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('globals.messages.selectTLS')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">OFF</SelectItem>
                <SelectItem value="tls">SSL/TLS</SelectItem>
                <SelectItem value="starttls">STARTTLS</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>{{ $t('admin.inbox.imap.tls.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.read_interval">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.imapScanInterval') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="5m" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.imapScanInterval.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.scan_inbox_since">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.imapScanInboxSince') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="48h" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.imapScanInboxSince.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="imap.tls_skip_verify">
        <FormItem class="flex flex-row items-center justify-between box p-4">
          <div class="space-y-0.5">
            <FormLabel class="text-base">{{ $t('admin.inbox.skipTLSVerification') }}</FormLabel>
            <FormDescription>
              {{ $t('admin.inbox.skipTLSVerification.description') }}
            </FormDescription>
          </div>
          <FormControl>
            <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
          </FormControl>
        </FormItem>
      </FormField>
    </div>

    <!-- SMTP Section -->
    <div v-show="!isOAuthInbox && setupMethod === 'manual'" class="box p-4 space-y-4">
      <h3 class="font-semibold">{{ $t('admin.inbox.smtpConfig') }}</h3>

      <FormField v-slot="{ componentField }" name="smtp.host">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.host') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="smtp.gmail.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.port">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.port') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="587" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.username">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.username') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="user@example.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.password">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.password') }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="••••••••" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.max_conns">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.maxConnections') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="10" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.maxConnections.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.max_msg_retries">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.maxRetries') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="3" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ $t('admin.inbox.maxRetries.description') }} </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.idle_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.idleTimeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="25s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.idleTimeout.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.wait_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.waitTimeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="60s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.waitTimeout.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.auth_protocol">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.authProtocol') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue placeholder="Select protocol" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="login">Login</SelectItem>
                <SelectItem value="cram">CRAM</SelectItem>
                <SelectItem value="plain">Plain</SelectItem>
                <SelectItem value="none">None</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription> {{ $t('admin.inbox.authProtocol.description') }} </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.tls_type">
        <FormItem>
          <FormLabel>{{ t('globals.terms.tls') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue placeholder="Select TLS" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">OFF</SelectItem>
                <SelectItem value="tls">SSL/TLS</SelectItem>
                <SelectItem value="starttls">STARTTLS</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription> {{ $t('admin.inbox.tls.description') }} </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.hello_hostname">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.heloHostname') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.heloHostname.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="smtp.tls_skip_verify">
        <FormItem class="flex flex-row items-center justify-between box p-4">
          <div class="space-y-0.5">
            <FormLabel class="text-base">{{ $t('admin.inbox.skipTLSVerification') }}</FormLabel>
            <FormDescription>
              {{ $t('admin.inbox.skipTLSVerification.description') }}
            </FormDescription>
          </div>
          <FormControl>
            <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
          </FormControl>
        </FormItem>
      </FormField>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>

  <!-- OAuth Credentials Modal -->
  <Dialog v-model:open="showOAuthModal">
    <DialogContent>
      <DialogHeader>
        <DialogTitle
          >Connect {{ selectedProvider === PROVIDER_GOOGLE ? 'Google' : 'Microsoft' }} Account</DialogTitle
        >
        <DialogDescription>
          Follow the steps below to connect your email account
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <div class="space-y-3">
          <p class="text-sm">
            1. Create OAuth app at
            <a
              :href="
                selectedProvider === PROVIDER_GOOGLE
                  ? 'https://console.cloud.google.com/apis/credentials'
                  : 'https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade'
              "
              target="_blank"
              class="text-primary underline"
            >
              {{
                selectedProvider === PROVIDER_GOOGLE ? 'Google Cloud Console' : 'Microsoft Azure Portal'
              }}
            </a>
          </p>

          <div class="space-y-1">
            <p class="text-sm">2. Add this callback URL:</p>
            <div class="flex items-center gap-2">
              <Input :model-value="callbackUrl" readonly class="font-mono text-xs" />
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="copyToClipboard(callbackUrl)"
              >
                Copy
              </Button>
            </div>
          </div>

          <p class="text-sm">3. Enter your credentials below:</p>
        </div>

        <div class="space-y-2">
          <label class="text-sm font-medium">Client ID</label>
          <Input
            v-model="oauthCredentials.client_id"
            placeholder="Enter your OAuth Client ID"
            :disabled="isSubmittingOAuth"
          />
        </div>

        <div class="space-y-2">
          <label class="text-sm font-medium">Client Secret</label>
          <Input
            v-model="oauthCredentials.client_secret"
            type="password"
            placeholder="Enter your OAuth Client Secret"
            :disabled="isSubmittingOAuth"
          />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="showOAuthModal = false" :disabled="isSubmittingOAuth">
          Cancel
        </Button>
        <Button @click="submitOAuthCredentials" :disabled="isSubmittingOAuth">
          {{ isSubmittingOAuth ? 'Connecting...' : 'Continue' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { watch, computed, ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { CheckCircle2, RefreshCw, Mail } from 'lucide-vue-next'
import MenuCard from '@/components/layout/MenuCard.vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { AUTH_TYPE_PASSWORD, AUTH_TYPE_OAUTH2, PROVIDER_GOOGLE, PROVIDER_MICROSOFT } from '@/constants/auth.js'
import { handleHTTPError } from '@/utils/http'
import { useAppSettingsStore } from '@/stores/appSettings'
import { h } from 'vue'

// Google icon component
const GoogleIcon = () =>
  h('svg', { class: 'w-6 h-6', viewBox: '0 0 24 24' }, [
    h('path', {
      fill: '#4285F4',
      d: 'M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z'
    }),
    h('path', {
      fill: '#34A853',
      d: 'M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z'
    }),
    h('path', {
      fill: '#FBBC05',
      d: 'M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z'
    }),
    h('path', {
      fill: '#EA4335',
      d: 'M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z'
    })
  ])

// Microsoft icon component
const MicrosoftIcon = () =>
  h('svg', { class: 'w-6 h-6', viewBox: '0 0 24 24' }, [
    h('path', { fill: '#f25022', d: 'M1 1h10v10H1z' }),
    h('path', { fill: '#00a4ef', d: 'M13 1h10v10H13z' }),
    h('path', { fill: '#7fba00', d: 'M1 13h10v10H1z' }),
    h('path', { fill: '#ffb900', d: 'M13 13h10v10H13z' })
  ])

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
  isLoading: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const emitter = useEmitter()
const appSettingsStore = useAppSettingsStore()

// OAuth detection
const isOAuthInbox = ref(false)

// Setup method selection: null | PROVIDER_GOOGLE | PROVIDER_MICROSOFT | 'manual'
const setupMethod = ref(null)

// OAuth modal state
const showOAuthModal = ref(false)
const selectedProvider = ref('')
const oauthCredentials = ref({
  client_id: '',
  client_secret: ''
})
const isSubmittingOAuth = ref(false)

// Computed callback URL for OAuth
const callbackUrl = computed(() => {
  const rootUrl = appSettingsStore.settings['app.root_url']
  return `${rootUrl}/api/v1/inboxes/oauth/${selectedProvider.value}/callback`
})

// Show form fields when OAuth is connected or manual setup is selected
const showFormFields = computed(
  () =>
    isOAuthInbox.value ||
    setupMethod.value === 'manual' ||
    (props.initialValues?.imap && Object.keys(props.initialValues?.imap).length > 0)
)

const form = useForm({
  validationSchema: computed(() => toTypedSchema(createFormSchema(t, isOAuthInbox.value))),
  initialValues: {
    name: '',
    from: '',
    enabled: true,
    csat_enabled: false,
    imap: {
      host: 'imap.gmail.com',
      port: 993,
      mailbox: 'INBOX',
      username: '',
      password: '',
      tls_type: 'none',
      read_interval: '5m',
      scan_inbox_since: '48h',
      tls_skip_verify: false
    },
    smtp: {
      host: 'smtp.gmail.com',
      port: 587,
      username: '',
      password: '',
      max_conns: 10,
      max_msg_retries: 3,
      idle_timeout: '25s',
      wait_timeout: '60s',
      auth_protocol: 'login',
      tls_type: 'none',
      hello_hostname: '',
      tls_skip_verify: false
    }
  }
})

// OAuth computed properties
const oauthProvider = computed(() => {
  const provider = form.values.imap?.oauth?.provider || form.values.smtp?.oauth?.provider
  return provider ? provider.charAt(0).toUpperCase() + provider.slice(1) : 'Google'
})

const oauthEmail = computed(() => {
  return form.values.imap?.username || form.values.smtp?.username || ''
})

const oauthClientId = computed(() => {
  return form.values.imap?.oauth?.client_id || form.values.smtp?.oauth?.client_id || ''
})

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.messages.save')
})

const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
})

const connectWithGoogle = () => {
  selectedProvider.value = PROVIDER_GOOGLE
  showOAuthModal.value = true
}

const connectWithMicrosoft = () => {
  selectedProvider.value = PROVIDER_MICROSOFT
  showOAuthModal.value = true
}

const reconnectOAuth = () => {
  const provider = form.values.oauth?.provider
  const clientId = form.values.oauth?.client_id

  if (!provider) return

  // Set provider and pre-fill credentials
  selectedProvider.value = provider
  oauthCredentials.value.client_id = clientId || ''
  oauthCredentials.value.client_secret = '' // Always require user to re-enter secret

  // Show modal for user to edit credentials
  showOAuthModal.value = true
}

const submitOAuthCredentials = async () => {
  if (!oauthCredentials.value.client_id || !oauthCredentials.value.client_secret) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Please provide both Client ID and Client Secret'
    })
    return
  }

  try {
    isSubmittingOAuth.value = true
    const response = await api.initiateOAuthFlow(selectedProvider.value, oauthCredentials.value)
    window.location.href = response.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmittingOAuth.value = false
  }
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: 'Callback URL copied to clipboard'
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Failed to copy to clipboard'
    })
  }
}

// Detect OAuth mode from form values
watch(
  () => form.values?.config?.auth_type,
  (authType) => {
    isOAuthInbox.value = authType === AUTH_TYPE_OAUTH2
  },
  { immediate: true }
)

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    if (Object.keys(newValues?.imap || {}).length > 0) {
      setupMethod.value = 'manual'
    }
    form.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
