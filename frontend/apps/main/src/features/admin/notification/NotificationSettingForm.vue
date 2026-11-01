<template>
  <form @submit="onSmtpSubmit" novalidate class="space-y-6">
    <FormField name="enabled" v-slot="{ value, handleChange }">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox
              :aria-label="i18n._('Enabled')"
              :checked="value"
              @update:checked="handleChange"
            />
            <Label>{{ i18n._('Enabled') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid compact>
      <FormField v-slot="{ componentField }" name="host">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('SMTP Host', 'SMTP Hosts', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="smtp.gmail.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="port">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('SMTP Port', 'SMTP Ports', 1) }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="587" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="tls_type">
        <FormItem>
          <FormLabel>TLS</FormLabel>
          <FormControl>
            <Select v-bind="componentField" v-model="componentField.modelValue">
              <SelectTrigger>
                <SelectValue :placeholder="i18n._('Select TLS type')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="none">Off</SelectItem>
                  <SelectItem value="tls">SSL/TLS</SelectItem>
                  <SelectItem value="starttls">STARTTLS</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <FormGrid>
      <FormField v-slot="{ componentField }" name="username">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Username', 'Usernames', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="admin@yourcompany.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="password">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Password', 'Passwords', 1) }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="auth_protocol">
        <FormItem>
          <FormLabel>{{ i18n._('Authentication Protocol') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField" v-model="componentField.modelValue">
              <SelectTrigger>
                <SelectValue :placeholder="i18n._('Authentication protocol to use.')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="plain">Plain</SelectItem>
                  <SelectItem value="login">Login</SelectItem>
                  <SelectItem value="cram">CRAM-MD5</SelectItem>
                  <SelectItem value="none">None</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <FormGrid>
      <FormField v-slot="{ componentField }" name="email_address">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('From email address', 'From email addresses', 1) }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="i18n._('My inbox \u003csupport@example.com\u003e')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
          <FormDescription> {{ i18n._('From email address for your inbox. e.g. My inbox \u003csupport@example.com\u003e') }}</FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="hello_hostname">
        <FormItem>
          <FormLabel>{{ i18n._('HELO Hostname') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ i18n._('The hostname to use in the HELO/EHLO command. If not set, defaults to localhost.') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <FormGrid compact>
      <FormField v-slot="{ componentField }" name="max_conns">
        <FormItem>
          <FormLabel>{{ i18n._('Max Connections') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="2" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription>{{ i18n._('Maximum number of concurrent connections to the server.') }} </FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="max_msg_retries">
        <FormItem>
          <FormLabel>{{ i18n._('Max Retries') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="3" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription> {{ i18n._('Number of times to retry when a message fails.') }} </FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="idle_timeout">
        <FormItem>
          <FormLabel>{{ i18n._('Idle Timeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="15s" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription>
            {{ i18n._('Maximum time an inactive connection must be kept alive before closing it and removing it from the pool.') }}
          </FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="wait_timeout">
        <FormItem>
          <FormLabel>{{ i18n._('Wait Timeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="5s" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription>
            {{ i18n._('Maximum time to wait to obtain a connection before timing out. Timeouts may occur when all open connections are busy sending e-mails and they\'\'re not returning to the pool fast enough. This is also the timeout used when creating new SMTP connections.') }}
          </FormDescription>
        </FormItem>
      </FormField>
    </FormGrid>

    <FormField v-slot="{ componentField, handleChange }" name="tls_skip_verify">
      <FormItem>
        <SwitchField
          :title="i18n._('Skip TLS Verification')"
          :description="i18n._('Skip hostname check on the TLS certificate.')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { watch, ref, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
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
} from '@shared-ui/components/ui/form/index.js'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { useI18n } from '@shared-ui/i18n'

const isLoading = ref(false)
const i18n = useI18n()
const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => ''
  }
})

const submitLabel = computed(() => {
  if (props.submitLabel) {
    return props.submitLabel
  }
  return i18n._('Save')
})

const smtpForm = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n))
})

const onSmtpSubmit = smtpForm.handleSubmit(async (values) => {
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
    smtpForm.setValues(newValues, false)
  },
  { deep: true, immediate: true }
)
</script>
