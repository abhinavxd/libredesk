<template>
  <form @submit="onSubmit" class="space-y-6">
    <FormField name="enabled" v-slot="{ value, handleChange }" v-if="!isNewForm">
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

    <FormGrid>
      <FormField v-slot="{ componentField }" name="provider">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Provider', 'Providers', 1) }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="i18n._('Select provider')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="Google"> Google </SelectItem>
                  <SelectItem value="Microsoft"> Microsoft </SelectItem>
                  <SelectItem value="Custom"> Custom </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="Google" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <FormGrid>
      <FormField v-slot="{ componentField }" name="provider_url">
        <FormItem>
          <FormLabel>{{ i18n._('Provider URL') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="https://accounts.google.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField
        v-slot="{ componentField }"
        name="logo_url"
        v-if="form.values.provider === 'Custom'"
      >
        <FormItem>
          <FormLabel>{{ i18n._('Logo URL') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="https://example.com/logo.svg" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ i18n._('Custom logo URL to display on the login page.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <FormGrid>
      <FormField v-slot="{ componentField }" name="client_id">
        <FormItem>
          <FormLabel>{{ i18n._('Client ID') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="client_secret">
        <FormItem>
          <FormLabel>{{ i18n._('Client secret') }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <FormField v-slot="{ componentField }" name="redirect_uri" v-if="!isNewForm">
      <FormItem>
        <FormLabel>{{ i18n._('Callback URL') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" readonly />
        </FormControl>
        <FormDescription>{{ i18n._('Set this URI for callback.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { watch, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { useI18n } from '@shared-ui/i18n'
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
const i18n = useI18n()

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? i18n._('Create') : i18n._('Save'))
  )
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n))
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    if (newValues && newValues.provider !== 'Custom') {
      newValues.logo_url = ''
    }
    form.setValues(newValues, false)
  },
  { deep: true, immediate: true }
)

// Clear logo_url when switching to Custom if current value is a built-in logo.
watch(
  () => form.values.provider,
  () => {
    if (form.values.provider === 'Custom' && form.values.logo_url?.startsWith('/images/')) {
      form.setFieldValue('logo_url', '', false)
    }
  }
)
</script>
