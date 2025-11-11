<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <FormField v-slot="{ field }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="$t('globals.terms.name')" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="website">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.website') }}</FormLabel>
        <FormControl>
          <Input type="url" :placeholder="$t('globals.terms.website')" v-bind="field" />
        </FormControl>
        <FormDescription>{{ $t('globals.messages.optional') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="email_domain">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.emailDomain') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="example.com" v-bind="field" />
        </FormControl>
        <FormDescription>
          {{ $t('globals.messages.emailDomainHelper') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="phone">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.phone') }}</FormLabel>
        <FormControl>
          <Input type="tel" :placeholder="$t('globals.terms.phone')" v-bind="field" />
        </FormControl>
        <FormDescription>{{ $t('globals.messages.optional') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: 'Submit'
  },
  isNewForm: {
    type: Boolean,
    required: false,
    default: false
  },
  isLoading: {
    type: Boolean,
    required: false,
    default: false
  }
})

const { t } = useI18n()

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const onSubmit = form.handleSubmit((values) => {
  // Clean up empty optional fields
  const cleanedValues = { ...values }
  if (!cleanedValues.website) cleanedValues.website = null
  if (!cleanedValues.email_domain) cleanedValues.email_domain = null
  if (!cleanedValues.phone) cleanedValues.phone = null

  props.submitForm(cleanedValues)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      setTimeout(() => {
        form.setValues(newValues)
      }, 0)
    }
  },
  { deep: true, immediate: true }
)
</script>
