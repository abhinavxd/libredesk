<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <!-- Basic Fields -->
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Inbox name" v-bind="componentField" />
        </FormControl>
        <FormDescription> Enter the name of the inbox. </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="from">
      <FormItem>
        <FormLabel>From Email Address</FormLabel>
        <FormControl>
          <Input
            type="text"
            placeholder="My Support <support@example.com>"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription> Enter the from email address. e.g. John Doe &lt;john@libredesk.io&gt; </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Toggle Fields -->
    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">Enabled</FormLabel>
          <FormDescription>Enable scanning inbox and sending emails</FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">CSAT Surveys</FormLabel>
          <FormDescription
            >Send customer satisfaction surveys when conversation is marked as resolved. <br />
            For better control on when to send surveys, disable this option and create an automation
            rule to send surveys.
          </FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <!-- IMAP Section -->
    <div class="box p-4 space-y-4">
      <h3 class="font-semibold">IMAP Configuration</h3>

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
          <FormLabel>Port</FormLabel>
          <FormControl>
            <Input type="number" placeholder="993" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.mailbox">
        <FormItem>
          <FormLabel>Mailbox</FormLabel>
          <FormControl>
            <Input type="text" placeholder="INBOX" v-bind="componentField" :defaultValue="'INBOX'" />
          </FormControl>
          <FormDescription>
            Mailbox (folder) to scan for incoming emails. Default is INBOX (usually no need to
            change).
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.username">
        <FormItem>
          <FormLabel>Username</FormLabel>
          <FormControl>
            <Input type="text" placeholder="user@example.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.password">
        <FormItem>
          <FormLabel>Password</FormLabel>
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
                <SelectValue placeholder="Select TLS" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">OFF</SelectItem>
                <SelectItem value="tls">SSL/TLS</SelectItem>
                <SelectItem value="starttls">STARTTLS</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>Choose the encryption method for IMAP.</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.read_interval">
        <FormItem>
          <FormLabel>Scan Interval</FormLabel>
          <FormControl>
            <Input type="text" placeholder="120s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            Interval to scan the inbox for new emails. Format: 120s, 1m, 1h
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="imap.scan_inbox_since">
        <FormItem>
          <FormLabel>Scan Inbox Since</FormLabel>
          <FormControl>
            <Input type="text" placeholder="48h" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            To improve performance in large helpdesks with high email volume, this limits scans to
            emails received since the specified duration (e.g., `2h`, `48h`) by subtracting it from
            the current time.
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="imap.tls_skip_verify">
        <FormItem class="flex flex-row items-center justify-between box p-4">
          <div class="space-y-0.5">
            <FormLabel class="text-base">Skip TLS Verification</FormLabel>
            <FormDescription> Skip hostname check on the TLS certificate. </FormDescription>
          </div>
          <FormControl>
            <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
          </FormControl>
        </FormItem>
      </FormField>
    </div>

    <!-- SMTP Section -->
    <div class="box p-4 space-y-4">
      <h3 class="font-semibold">SMTP Configuration</h3>

      <FormField v-slot="{ componentField }" name="smtp.host">
        <FormItem>
          <FormLabel>Host</FormLabel>
          <FormControl>
            <Input type="text" placeholder="smtp.gmail.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.port">
        <FormItem>
          <FormLabel>Port</FormLabel>
          <FormControl>
            <Input type="number" placeholder="587" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.username">
        <FormItem>
          <FormLabel>Username</FormLabel>
          <FormControl>
            <Input type="text" placeholder="user@example.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.password">
        <FormItem>
          <FormLabel>Password</FormLabel>
          <FormControl>
            <Input type="password" placeholder="••••••••" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.max_conns">
        <FormItem>
          <FormLabel>Max Connections</FormLabel>
          <FormControl>
            <Input type="number" placeholder="10" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            Maximum number of concurrent connections to the server.
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.max_msg_retries">
        <FormItem>
          <FormLabel>Max Retries</FormLabel>
          <FormControl>
            <Input type="number" placeholder="3s" v-bind="componentField" />
          </FormControl>
          <FormDescription> Number of times to retry when a message fails. </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.idle_timeout">
        <FormItem>
          <FormLabel>Idle Timeout</FormLabel>
          <FormControl>
            <Input type="text" placeholder="25s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            IdleTimeout is the maximum time to wait for new activity on a connection before closing
            it and removing it from the pool.
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.wait_timeout">
        <FormItem>
          <FormLabel>Wait Timeout</FormLabel>
          <FormControl>
            <Input type="text" placeholder="60s" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            PoolWaitTimeout is the maximum time to wait to obtain a connection from a pool before
            timing out. This may happen when all open connections are busy sending e-mails and
            they're not returning to the pool fast enough. This is also the timeout used when
            creating new SMTP connections.
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.auth_protocol">
        <FormItem>
          <FormLabel>Auth Protocol</FormLabel>
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
          <FormDescription> Authentication protocol to use. </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.tls_type">
        <FormItem>
          <FormLabel>TLS</FormLabel>
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
          <FormDescription> TLS/SSL encryption, STARTTLS is commonly used. </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="smtp.hello_hostname">
        <FormItem>
          <FormLabel>HELO Hostname</FormLabel>
          <FormControl>
            <Input type="text" placeholder="smtp.example.com" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            The hostname to use in the HELO/EHLO command. If not set, defaults to localhost.
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="smtp.tls_skip_verify">
        <FormItem class="flex flex-row items-center justify-between box p-4">
          <div class="space-y-0.5">
            <FormLabel class="text-base">Skip TLS Verification</FormLabel>
            <FormDescription> Skip hostname check on the TLS certificate. </FormDescription>
          </div>
          <FormControl>
            <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
          </FormControl>
        </FormItem>
      </FormField>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ props.submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
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
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'

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
    default: 'Submit'
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
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
</script>
