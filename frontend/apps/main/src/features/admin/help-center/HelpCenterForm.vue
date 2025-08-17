<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }} *</FormLabel>
        <FormControl>
          <Input
            type="text"
            placeholder="Enter help center name"
            v-bind="componentField"
            @input="generateSlug"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="slug">
      <FormItem>
        <FormLabel>Slug *</FormLabel>
        <FormControl>
          <Input type="text" placeholder="help-center-slug" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          This will be used in the URL: /help/{{ form.values.slug || 'your-slug' }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="page_title">
      <FormItem>
        <FormLabel>Page Title *</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Enter page title" v-bind="componentField" />
        </FormControl>
        <FormDescription> This will appear in the browser tab and search results </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="default_locale">
      <FormItem>
        <FormLabel>Default Language *</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select default language" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="language in LANGUAGES" :key="language.code" :value="language.code">
                {{ language.nativeName }}
              </SelectItem>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>
          This will be the default language for new articles and collections
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="flex justify-end space-x-2 pt-4">
      <Button type="button" variant="outline" @click="$emit('cancel')"> Cancel </Button>
      <Button type="submit" :isLoading="isLoading">
        {{ submitLabel }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { LANGUAGES } from '@shared-ui/constants'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { createHelpCenterFormSchema } from './helpCenterFormSchema.js'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  helpCenter: {
    type: Object,
    default: null
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

defineEmits(['cancel'])

const formLoading = ref(false)

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.helpCenter ? t('globals.messages.update') : t('globals.messages.create'))
  )
})

const form = useForm({
  validationSchema: toTypedSchema(createHelpCenterFormSchema(t)),
  initialValues: {
    name: props.helpCenter?.name || '',
    slug: props.helpCenter?.slug || '',
    page_title: props.helpCenter?.page_title || '',
    default_locale: props.helpCenter?.default_locale || 'en'
  }
})

const generateSlug = () => {
  if (!props.helpCenter && form.values.name) {
    form.setFieldValue(
      'slug',
      form.values.name
        .toLowerCase()
        .replace(/[^a-z0-9]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '')
    )
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  props.submitForm(values)
})

watch(
  () => props.helpCenter,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      form.setValues({
        name: newValues.name || '',
        slug: newValues.slug || '',
        page_title: newValues.page_title || '',
        default_locale: newValues.default_locale || 'en'
      })
    }
  },
  { immediate: true }
)
</script>
