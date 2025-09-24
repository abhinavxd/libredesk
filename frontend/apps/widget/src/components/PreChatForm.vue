<template>
  <div class="bg-background flex-1 flex flex-col">
    <div v-if="showForm" class="flex-1 flex flex-col max-h-full">
      <div
        class="flex-1 overflow-y-auto scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50 p-4 space-y-4"
      >
        <!-- Form title -->
        <div v-if="formTitle" class="text-xl text-foreground mb-2 text-center">
          {{ formTitle }}
        </div>

        <!-- Default message -->
        <div v-else class="text-lg font-semibold text-foreground mb-2">
          {{ $t('globals.terms.helpUsServeYouBetter') }}
        </div>

        <form @submit.prevent="submitForm" class="space-y-4">
          <!-- Dynamic fields -->
          <div v-for="field in sortedFields" :key="field.key" class="space-y-2">
            <!-- Text input -->
            <FormField v-if="field.type === 'text'" v-slot="{ componentField }" :name="field.key">
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    v-bind="componentField"
                    type="text"
                    :placeholder="field.placeholder || ''"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Email input -->
            <FormField
              v-else-if="field.type === 'email'"
              v-slot="{ componentField }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    v-bind="componentField"
                    type="email"
                    :placeholder="field.placeholder || ''"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Number input -->
            <FormField
              v-else-if="field.type === 'number'"
              v-slot="{ componentField }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    v-bind="componentField"
                    type="number"
                    :placeholder="field.placeholder || ''"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Date input -->
            <FormField
              v-else-if="field.type === 'date'"
              v-slot="{ componentField }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    v-bind="componentField"
                    type="date"
                    :placeholder="field.placeholder || ''"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Link/URL input -->
            <FormField
              v-else-if="field.type === 'link'"
              v-slot="{ componentField }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    v-bind="componentField"
                    type="url"
                    :placeholder="field.placeholder || 'https://'"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Checkbox input -->
            <FormField
              v-else-if="field.type === 'checkbox'"
              v-slot="{ componentField, handleChange }"
              :name="field.key"
            >
              <FormItem class="flex flex-row items-start space-x-3 space-y-0">
                <FormControl>
                  <Checkbox :checked="componentField.modelValue" @update:checked="handleChange" />
                </FormControl>
                <div class="space-y-1 leading-none">
                  <FormLabel class="text-sm font-medium">
                    {{ field.label }}
                    <span v-if="field.required" class="text-destructive">*</span>
                  </FormLabel>
                  <FormMessage />
                </div>
              </FormItem>
            </FormField>

            <!-- List/Select input -->
            <FormField
              v-else-if="field.type === 'list'"
              v-slot="{ componentField }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue :placeholder="field.placeholder || $t('globals.terms.select')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem
                        v-for="option in getFieldOptions(field)"
                        :key="option.value"
                        :value="option.value"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>
        </form>
      </div>

      <!-- Submit button - fixed at bottom -->
      <div class="p-4 border-t">
        <Button @click="submitForm" class="w-full" :disabled="!requiredFieldsFilled || !meta.valid">
          {{ $t('globals.terms.continue') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
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
  FormMessage
} from '@shared-ui/components/ui/form'
import { useWidgetStore } from '../store/widget.js'
import { useI18n } from 'vue-i18n'
import { createPreChatFormSchema } from './preChatFormSchema.js'

const props = defineProps({
  excludeDefaultFields: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['submit'])
const { t } = useI18n()
const widgetStore = useWidgetStore()

const config = computed(() => widgetStore.config?.prechat_form || {})
const preChatFormEnabled = computed(() => config.value.enabled || false)
const formTitle = computed(() => config.value.title || '')
const formFields = computed(() => config.value.fields || [])

// Sort and filter enabled fields, excluding default fields if user has session token
const sortedFields = computed(() => {
  let fields = formFields.value.filter((field) => field.enabled)

  // If user has session token, exclude default name and email fields
  if (props.excludeDefaultFields) {
    fields = fields.filter((field) => !['name', 'email'].includes(field.key))
  }

  return fields.sort((a, b) => (a.order || 0) - (b.order || 0))
})

const showForm = computed(() => preChatFormEnabled.value && sortedFields.value.length > 0)

// Create form with dynamic schema based on fields
const formSchema = computed(() => toTypedSchema(createPreChatFormSchema(t, sortedFields.value)))

// Generate initial values dynamically
const initialValues = computed(() => {
  const values = {}
  sortedFields.value.forEach((field) => {
    if (field.type === 'checkbox') {
      values[field.key] = false
    } else {
      values[field.key] = ''
    }
  })
  return values
})

const { handleSubmit, meta, values } = useForm({
  validationSchema: formSchema,
  initialValues
})

const requiredFieldsFilled = computed(() => {
  return sortedFields.value
    .filter((field) => field.required)
    .every((field) => {
      const value = values[field.key]
      if (field.type === 'checkbox') return true
      return value && String(value).trim() !== ''
    })
})

const submitForm = handleSubmit((values) => {
  // Filter out empty values (except for checkboxes)
  const filteredValues = {}
  Object.keys(values).forEach((key) => {
    const field = sortedFields.value.find((f) => f.key === key)
    if (field?.type === 'checkbox' || (values[key] && String(values[key]).trim())) {
      filteredValues[key] = values[key]
    }
  })

  emit('submit', filteredValues)
})

// Get options for list fields
const getFieldOptions = (field) => {
  if (field.type === 'list' && field.custom_attribute_id) {
    const customAttr = widgetStore.config?.custom_attributes?.[field.custom_attribute_id]
    if (customAttr?.values) {
      return customAttr.values.map((value) => ({
        value: value,
        label: value
      }))
    }
  }
  return []
}

// Auto-submit for disabled mode
watch(
  showForm,
  (newValue) => {
    if (!newValue) {
      emit('submit', {})
    }
  },
  { immediate: true }
)
</script>
