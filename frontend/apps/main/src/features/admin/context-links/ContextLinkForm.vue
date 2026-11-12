<template>
  <form class="space-y-6 w-full">
    <FormField name="is_active" v-slot="{ value, handleChange }" v-if="!isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox
              :aria-label="i18n._('Active')"
              :checked="value"
              @update:checked="handleChange"
            />
            <Label>{{ i18n._('Active') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="i18n._('My App')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="url_template">
        <FormItem>
          <FormLabel>{{ i18n._('URL template') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              placeholder="https://tools.example.com/lookup?token={{token}}"
              v-bind="componentField"
            />
          </FormControl>
          <FormDescription>{{ i18n._(literalExampleMessage1) }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormGrid>
      <FormField v-slot="{ componentField }" name="secret">
        <FormItem>
          <FormLabel>{{ i18n.pgettext('Context link encryption key', 'Secret') }}</FormLabel>
          <FormControl>
            <Input type="password" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ i18n._(literalExampleMessage2) }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="token_expiry_seconds">
        <FormItem>
          <FormLabel>{{ i18n._('Token expiry (seconds)') }}</FormLabel>
          <FormControl>
            <Input type="number" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ i18n._('The external app should reject tokens older than this. Included as \'\'exp\'\' (expiry timestamp) inside the encrypted token.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

const literalExampleMessage1 = msg({ message: '\'{{token}}\' is a base64-encoded AES-256-GCM encrypted blob containing all contact and agent fields (requires secret). Individual variables like \'{{email}}\', \'{{phone}}\', \'{{external_user_id}}\', \'{{contact_id}}\', \'{{first_name}}\', \'{{last_name}}\', \'{{conversation_uuid}}\' are passed as plain text.' })
const literalExampleMessage2 = msg({ message: 'A 32-character key shared between Libredesk and the external app. Used to encrypt the \'{{token}}\' so only the external app can read the contact details. Not needed if you only use plain variables like \'{{email}}\'.' })


import FormGrid from '@shared-ui/components/FormGrid.vue'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Label } from '@shared-ui/components/ui/label'
import { useI18n } from '@shared-ui/i18n'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'

defineProps({
  form: {
    type: Object,
    required: true
  },
  isNewForm: {
    type: Boolean
  }
})

const i18n = useI18n()
</script>
