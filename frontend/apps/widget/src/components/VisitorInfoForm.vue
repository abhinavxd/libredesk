<template>
  <div class="border-t bg-background p-4">
    <div v-if="showForm" class="space-y-4">
      <!-- Custom message -->
      <div v-if="contactInfoMessage" class="text-sm text-muted-foreground">
        {{ contactInfoMessage }}
      </div>
      
      <!-- Default message -->
      <div v-else class="text-sm text-muted-foreground">
        {{ $t('globals.placeholders.helpUsServeYouBetter') }}
      </div>

      <form @submit.prevent="submitForm" class="space-y-4">
        <!-- Name input -->
        <FormField v-slot="{ componentField }" name="name">
          <FormItem>
            <FormLabel class="text-sm font-medium">
              {{ $t('globals.terms.name') }}
              <span v-if="isRequired" class="text-destructive">*</span>
            </FormLabel>
            <FormControl>
              <Input
                v-bind="componentField"
                type="text"
                :placeholder="$t('globals.placeholders.name')"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <!-- Email input -->
        <FormField v-slot="{ componentField }" name="email">
          <FormItem>
            <FormLabel class="text-sm font-medium">
              {{ $t('globals.terms.email') }}
              <span v-if="isRequired" class="text-destructive">*</span>
            </FormLabel>
            <FormControl>
              <Input
                v-bind="componentField"
                type="email"
                :placeholder="$t('globals.placeholders.email')"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <!-- Submit button -->
        <Button 
          type="submit"
          class="w-full"
          :disabled="!meta.valid"
        >
          {{ $t('globals.terms.continue') }}
        </Button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@shared-ui/components/ui/form'
import { useWidgetStore } from '../store/widget.js'
import { useI18n } from 'vue-i18n'
import { createVisitorInfoSchema } from './visitorInfoFormSchema.js'

const emit = defineEmits(['submit'])
const { t } = useI18n()
const widgetStore = useWidgetStore()

const config = computed(() => widgetStore.config?.visitors || {})
const requireContactInfo = computed(() => config.value.require_contact_info || 'disabled')
const contactInfoMessage = computed(() => config.value.contact_info_message || '')

const showForm = computed(() => requireContactInfo.value !== 'disabled')
const isRequired = computed(() => requireContactInfo.value === 'required')

// Create form with dynamic schema based on requirements
const formSchema = computed(() => 
  toTypedSchema(createVisitorInfoSchema(t, requireContactInfo.value))
)

const { handleSubmit, meta, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: {
    name: '',
    email: ''
  }
})

const submitForm = handleSubmit((values) => {
  emit('submit', {
    name: values.name?.trim() || '',
    email: values.email?.trim() || ''
  })
})

// Auto-submit for disabled mode
watch(showForm, (newValue) => {
  if (!newValue) {
    emit('submit', { name: '', email: '' })
  }
}, { immediate: true })

// Reset form when requirements change
watch(requireContactInfo, () => {
  resetForm({
    values: { name: '', email: '' }
  })
})
</script>