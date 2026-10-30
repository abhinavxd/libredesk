<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" @input="generateSlug" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="slug">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Slug', 'Slugs', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" @input="markSlugEdited" />
          </FormControl>
          <FormDescription>{{ i18n._('Used in the help center URL.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="page_title">
        <FormItem class="col-span-full">
          <FormLabel>{{ i18n._('Page title') }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormField v-slot="{ componentField }" name="template">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Template', 'Templates', 1, {}, 'Help center layout') }}</FormLabel>
        <FormControl>
          <RadioGroup v-bind="componentField">
            <div class="flex flex-col space-y-2">
              <div class="flex items-start space-x-3">
                <RadioGroupItem id="hc-template-classic" value="classic" class="mt-1" />
                <Label for="hc-template-classic" class="font-normal cursor-pointer">
                  <span class="block font-medium">{{ i18n._('Classic') }}</span>
                  <span class="block text-muted-foreground">{{
                    i18n._('Landing page layout with a hero banner, a large search bar and collection cards.')
                  }}</span>
                </Label>
              </div>
              <div class="flex items-start space-x-3">
                <RadioGroupItem id="hc-template-docs" value="docs" class="mt-1" />
                <Label for="hc-template-docs" class="font-normal cursor-pointer">
                  <span class="block font-medium">{{ i18n._('Docs') }}</span>
                  <span class="block text-muted-foreground">{{
                    i18n._('Documentation layout with a top navigation bar, a sidebar listing collections and quick search (Ctrl+K).')
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
        {{ i18n._('Cancel') }}
      </Button>
      <Button type="submit" :isLoading="isLoading">
        {{ i18n._('Create') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
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
import { useI18n } from '@shared-ui/i18n'

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

const i18n = useI18n()

const form = useForm({
  validationSchema: toTypedSchema(createHelpCenterBasicsSchema(i18n)),
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
