<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <!-- Basic Fields -->
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormDescription> {{ $t('admin.inbox.name.description') }} </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="help_center_id">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.helpCenter') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue
                :placeholder="
                  t('globals.messages.select', {
                    name: $t('globals.terms.helpCenter').toLowerCase()
                  })
                "
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="0">{{ $t('globals.terms.none') }}</SelectItem>
              <SelectItem
                v-for="helpCenter in helpCenters"
                :key="helpCenter.id"
                :value="helpCenter.id"
              >
                {{ helpCenter.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ $t('admin.inbox.helpCenter.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="from">
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
    <FormField v-slot="{ componentField, handleChange }" name="enabled">
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

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
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

    <!-- IMAP Section -->
    <div class="box p-4 space-y-4">
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
    <div class="box p-4 space-y-4">
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
</template>

<script setup>
import { watch, computed, ref, onMounted } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import api from '@main/api'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Switch } from '@shared-ui/components/ui/switch/index.js'
import { Button } from '@shared-ui/components/ui/button/index.js'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import { useI18n } from 'vue-i18n'

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
const helpCenters = ref([])

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    name: '',
    help_center_id: 0,
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

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.messages.save')
})

const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    form.setValues(newValues)
  },
  { deep: true, immediate: true }
)

onMounted(async () => {
  try {
    const { data } = await api.getHelpCenters()
    helpCenters.value = data.data
  } catch (error) {
    console.error('Error fetching help centers:', error)
  }
})
</script>
