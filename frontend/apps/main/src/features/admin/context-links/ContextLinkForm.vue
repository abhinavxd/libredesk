<template>
  <form class="space-y-6 w-full">
    <FormField name="is_active" v-slot="{ value, handleChange }" v-if="!isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox
              :aria-label="$t('globals.terms.active')"
              :checked="value"
              @update:checked="handleChange"
            />
            <Label>{{ $t('globals.terms.active') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="t('contextLink.namePlaceholder')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="url_template">
        <FormItem>
          <FormLabel>{{ $t('contextLink.urlTemplate') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              placeholder="https://tools.example.com/lookup?token={{token}}"
              v-bind="componentField"
            />
          </FormControl>
          <FormDescription>{{ $t('contextLink.urlTemplateHelp') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormGrid>
      <FormField v-slot="{ componentField }" name="secret">
        <FormItem>
          <FormLabel>{{ $t('contextLink.secret') }}</FormLabel>
          <FormControl>
            <Input type="password" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ $t('contextLink.secretHelp') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="token_expiry_seconds">
        <FormItem>
          <FormLabel>{{ $t('contextLink.tokenExpiry') }}</FormLabel>
          <FormControl>
            <Input type="number" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ $t('contextLink.tokenExpiryHelp') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Label } from '@shared-ui/components/ui/label'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()
</script>
