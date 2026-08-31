<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" @input="generateSlug" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="slug">
      <FormItem>
        <FormLabel>{{ t('globals.terms.slug') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" @input="markSlugEdited" />
        </FormControl>
        <FormDescription>{{ t('helpCenter.slugHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="page_title">
      <FormItem>
        <FormLabel>{{ t('helpCenter.pageTitle') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="template">
      <FormItem>
        <FormLabel>{{ t('globals.terms.template') }}</FormLabel>
        <FormControl>
          <RadioGroup v-bind="componentField">
            <div class="flex flex-col space-y-2">
              <div class="flex items-start space-x-3">
                <RadioGroupItem id="hc-template-classic" value="classic" class="mt-1" />
                <Label for="hc-template-classic" class="font-normal cursor-pointer">
                  <span class="block font-medium">{{ t('helpCenter.templates.classic') }}</span>
                  <span class="block text-muted-foreground">{{
                    t('helpCenter.templates.classicHint')
                  }}</span>
                </Label>
              </div>
              <div class="flex items-start space-x-3">
                <RadioGroupItem id="hc-template-docs" value="docs" class="mt-1" />
                <Label for="hc-template-docs" class="font-normal cursor-pointer">
                  <span class="block font-medium">{{ t('helpCenter.templates.docs') }}</span>
                  <span class="block text-muted-foreground">{{
                    t('helpCenter.templates.docsHint')
                  }}</span>
                </Label>
              </div>
            </div>
          </RadioGroup>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="flex justify-end space-x-2 pt-4">
      <Button type="button" variant="outline" @click="$emit('cancel')">
        {{ t('globals.messages.cancel') }}
      </Button>
      <Button type="submit" :isLoading="isLoading">
        {{ t('globals.messages.create') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group/index.js'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { createHelpCenterBasicsSchema } from './helpCenterFormSchema.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  submitForm: {
    type: Function,
    required: true
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

defineEmits(['cancel'])

const { t } = useI18n()

const form = useForm({
  validationSchema: toTypedSchema(createHelpCenterBasicsSchema(t)),
  initialValues: { name: '', slug: '', page_title: '', template: 'classic' }
})

// A hand-edited slug is the admin's choice; clearing the field resumes the suggestions.
const slugEdited = ref(false)
const markSlugEdited = (e) => {
  slugEdited.value = !!e.target.value
}

// Mirrors stringutil.GenerateSlug on the backend so the suggestion matches what gets saved.
const generateSlug = () => {
  if (slugEdited.value || !form.values.name) return
  form.setFieldValue(
    'slug',
    form.values.name
      .normalize('NFD')
      .replace(/\p{Diacritic}/gu, '')
      .trim()
      .toLowerCase()
      .replace(/\s+/g, '-')
      .replace(/[^a-z0-9\-_]+/g, '')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, ''),
    false
  )
}

const onSubmit = form.handleSubmit(async (values) => {
  props.submitForm(values)
})
</script>
