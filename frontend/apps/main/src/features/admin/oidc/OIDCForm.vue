<template>
  <form @submit="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="provider">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.provider') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="t('globals.messages.select', { name: t('globals.terms.provider') })" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="Google"> Google </SelectItem>
                <SelectItem value="Custom"> Custom </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Google" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="provider_url">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.providerURL') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="https://accounts.google.com" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="client_id">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.clientID') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="client_secret">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.clientSecret') }}</FormLabel>
        <FormControl>
          <Input type="password" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="redirect_uri" v-if="!isNewForm">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.callbackURL') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" readonly />
        </FormControl>
        <FormDescription>{{ $t('admin.sso.setThisUrlForCallback') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="enabled" v-slot="{ value, handleChange }" v-if="!isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>{{ $t('globals.terms.enabled') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { useI18n } from 'vue-i18n'
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
import { Input } from '@shared-ui/components/ui/input/index.js'

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
  },
  isNewForm: {
    type: Boolean
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})
const { t } = useI18n()

const submitLabel = props.submitLabel || t('globals.messages.save')

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
