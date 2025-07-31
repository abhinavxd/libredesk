<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <!-- Question Field -->
    <FormField v-slot="{ componentField }" name="question">
      <FormItem>
        <FormLabel>{{ t('globals.terms.question') }} <span class="text-red-500">*</span></FormLabel>
        <FormControl>
          <Textarea
            :placeholder="t('ai.customAnswer.questionPlaceholder')"
            v-bind="componentField"
            rows="3"
          />
        </FormControl>
        <FormDescription>{{ t('ai.customAnswer.questionDescription') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Answer Field -->
    <FormField v-slot="{ componentField }" name="answer">
      <FormItem>
        <FormLabel>{{ t('globals.terms.answer') }} <span class="text-red-500">*</span></FormLabel>
        <FormControl>
          <Textarea
            :placeholder="t('ai.customAnswer.answerPlaceholder')"
            v-bind="componentField"
            rows="6"
          />
        </FormControl>
        <FormDescription>{{ t('ai.customAnswer.answerDescription') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Enabled Field -->
    <FormField v-slot="{ value, handleChange }" name="enabled" type="checkbox">
      <FormItem class="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
        <FormControl>
          <Checkbox :checked="value" @update:checked="handleChange" />
        </FormControl>
        <div class="space-y-1 leading-none">
          <FormLabel>{{ t('globals.terms.enabled') }}</FormLabel>
          <FormDescription>{{ t('ai.customAnswer.enabledDescription') }}</FormDescription>
        </div>
      </FormItem>
    </FormField>

    <!-- Submit Button -->
    <div class="flex gap-4">
      <Button type="submit" class="min-w-[120px]" :disabled="formLoading">
        <Spinner v-if="formLoading" class="mr-2 h-4 w-4" />
        {{ t('globals.buttons.save') }}
      </Button>
      <Button type="button" variant="outline" @click="$emit('cancel')">
        {{ t('globals.buttons.cancel') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'

import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { createFormSchema } from './formSchema.js'

const { t } = useI18n()

const props = defineProps({
  customAnswer: {
    type: Object,
    default: null
  },
  formLoading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['submit', 'cancel'])

const formSchema = toTypedSchema(createFormSchema(t))

const { handleSubmit, setValues } = useForm({
  validationSchema: formSchema,
  initialValues: {
    question: '',
    answer: '',
    enabled: true
  }
})

const onSubmit = handleSubmit((values) => {
  emit('submit', values)
})

// Watch for changes in customAnswer prop and update form values
watch(
  () => props.customAnswer,
  (newCustomAnswer) => {
    if (newCustomAnswer) {
      setValues({
        question: newCustomAnswer.question || '',
        answer: newCustomAnswer.answer || '',
        enabled: newCustomAnswer.enabled !== undefined ? newCustomAnswer.enabled : true
      })
    }
  },
  { immediate: true }
)
</script>