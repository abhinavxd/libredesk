<template>
  <div :class="props.handoffMode ? 'flex flex-col' : 'bg-background flex-1 flex flex-col'">
    <div
      v-if="showForm"
      :class="props.handoffMode ? 'flex flex-col' : 'flex-1 flex flex-col max-h-full'"
    >
      <div
        :class="[
          'space-y-4',
          props.handoffMode
            ? 'py-1'
            : 'flex-1 overflow-y-auto scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50 p-4'
        ]"
      >
        <div
          v-if="formTitle"
          :class="
            props.handoffMode
              ? 'text-sm font-medium text-foreground'
              : 'text-xl text-foreground mb-2 text-center'
          "
        >
          {{ formTitle }}
        </div>

        <form ref="formRef" @submit.prevent="submitForm" novalidate class="space-y-4">
          <!-- Dynamic fields -->
          <div v-for="field in sortedFields" :key="field.key" class="space-y-2">
            <!-- Text input -->
            <FormField
              v-if="field.type === 'text'"
              v-slot="{ componentField, handleChange, meta }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    :name="field.key"
                    :model-value="componentField.modelValue"
                    type="text"
                    :placeholder="field.placeholder || ''"
                    @update:model-value="(value) => handleChange(value, meta.validated)"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Email input -->
            <FormField
              v-else-if="field.type === 'email'"
              v-slot="{ componentField, handleChange, meta }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    :name="field.key"
                    :model-value="componentField.modelValue"
                    type="email"
                    :placeholder="field.placeholder || ''"
                    @update:model-value="(value) => handleChange(value, meta.validated)"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Number input -->
            <FormField
              v-else-if="field.type === 'number'"
              v-slot="{ componentField, handleChange, meta }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    :name="field.key"
                    :model-value="componentField.modelValue"
                    type="number"
                    :placeholder="field.placeholder || ''"
                    @update:model-value="
                      (value) => handleChange(value === '' ? '' : Number(value), meta.validated)
                    "
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Date input -->
            <FormField
              v-else-if="field.type === 'date'"
              v-slot="{ componentField, handleChange, meta }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    :name="field.key"
                    :model-value="componentField.modelValue"
                    type="date"
                    :placeholder="field.placeholder || ''"
                    @update:model-value="(value) => handleChange(value, meta.validated)"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Link/URL input -->
            <FormField
              v-else-if="field.type === 'link'"
              v-slot="{ componentField, handleChange, meta }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Input
                    :name="field.key"
                    :model-value="componentField.modelValue"
                    type="url"
                    :placeholder="field.placeholder || 'https://'"
                    @update:model-value="(value) => handleChange(value, meta.validated)"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Checkbox input -->
            <FormField
              v-else-if="field.type === 'checkbox'"
              v-slot="{ componentField, handleChange, meta }"
              :name="field.key"
            >
              <FormItem class="flex flex-row items-start space-x-3 space-y-0">
                <FormControl>
                  <Checkbox
                    :checked="componentField.modelValue"
                    @update:checked="(value) => handleChange(value, meta.validated)"
                  />
                </FormControl>
                <div class="space-y-1 leading-none">
                  <FormLabel class="text-sm font-medium">
                    {{ field.label }}
                  </FormLabel>
                  <FormMessage />
                </div>
              </FormItem>
            </FormField>

            <!-- Phone input -->
            <PhoneNumberInput
              v-else-if="field.type === 'phone'"
              :phone-number-name="field.key"
              :country-code-name="countryCodeKey(field.key)"
              :label="field.label"
              :placeholder="field.placeholder || ''"
              :required="field.required"
              defer-validation
            />

            <!-- List/Select input -->
            <FormField
              v-else-if="field.type === 'list'"
              v-slot="{ componentField, handleChange, meta }"
              :name="field.key"
            >
              <FormItem>
                <FormLabel class="text-sm font-medium">
                  {{ field.label }}
                  <span v-if="field.required" class="text-destructive">*</span>
                </FormLabel>
                <FormControl>
                  <Select
                    :model-value="componentField.modelValue"
                    @update:model-value="(value) => handleChange(value, meta.validated)"
                  >
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

          <div v-if="!props.handoffMode" class="space-y-2">
            <label class="text-sm font-medium">
              {{ $t('globals.terms.message') }}
              <span class="text-destructive">*</span>
            </label>
            <Textarea
              v-model="messageText"
              :placeholder="$t('globals.terms.typeMessage')"
              class="w-full min-h-32 max-h-48 resize-none"
            />
          </div>
        </form>
      </div>

      <div :class="props.handoffMode ? 'pt-3' : 'p-4 border-t'">
        <Button
          type="button"
          @click="submitForm"
          class="w-full"
          :disabled="
            !requiredFieldsFilled ||
            (!props.handoffMode && !messageText.trim()) ||
            props.isSubmitting
          "
        >
          <div
            v-if="props.isSubmitting"
            class="w-4 h-4 border-2 border-background border-t-current rounded-full animate-spin mr-2"
          ></div>
          {{
            props.handoffMode
              ? $t('widget.prechatForm.continueToHuman')
              : $t('widget.prechatForm.startChat')
          }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
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
import PhoneNumberInput from '@shared-ui/components/PhoneNumberInput.vue'
import { countryCodeKey, defaultCountryCode } from '@shared-ui/utils/phone.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import { resolvePreChatForm } from '@widget/utils/preChatForm.js'
import { useI18n } from 'vue-i18n'
import { createPreChatFormSchema } from './preChatFormSchema.js'
import api from '@widget/api/index.js'

const props = defineProps({
  excludeDefaultFields: {
    type: Boolean,
    default: false
  },
  isSubmitting: {
    type: Boolean,
    default: false
  },
  handoffMode: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['submit'])
const { t } = useI18n()
const widgetStore = useWidgetStore()
const chat = useChatStore()
const userStore = useUserStore()
const messageText = computed({
  get: () => chat.preChatDraft.message || '',
  set: (value) => {
    chat.preChatDraft.message = value
  }
})
const formRef = ref(null)

const config = computed(() =>
  resolvePreChatForm(widgetStore.config?.prechat_form, userStore.isVisitor)
)
const preChatFormEnabled = computed(() => config.value.enabled || false)
const formTitle = computed(() => config.value.title || '')
const formFields = computed(() => config.value.fields || [])

// Sort and filter enabled fields, excluding default fields if user has session token
const sortedFields = computed(() => {
  let fields = formFields.value.filter((field) => field.enabled)

  // If user has session token, exclude default name, email and phone fields
  if (props.excludeDefaultFields) {
    fields = fields.filter((field) => !['name', 'email', 'phone'].includes(field.key))
  }

  return fields.sort((a, b) => (a.order || 0) - (b.order || 0))
})

const showForm = computed(() => preChatFormEnabled.value && sortedFields.value.length > 0)

// Create form with dynamic schema based on fields
const formSchema = computed(() => toTypedSchema(createPreChatFormSchema(t, sortedFields.value)))

// Generate initial values dynamically
const draft = props.handoffMode ? chat.handoffDraft : chat.preChatDraft
const savedFields = { ...draft.fields }
const initialValues = computed(() => {
  const values = {}
  sortedFields.value.forEach((field) => {
    if (field.type === 'checkbox') {
      values[field.key] = false
    } else if (field.type === 'phone') {
      values[field.key] = ''
      values[countryCodeKey(field.key)] = defaultCountryCode()
    } else {
      values[field.key] = ''
    }
  })
  return { ...values, ...savedFields }
})

const { handleSubmit, values } = useForm({
  validationSchema: formSchema,
  initialValues
})

watch(
  values,
  (fields) => {
    draft.fields = { ...fields }
  },
  { deep: true }
)

const requiredFieldsFilled = computed(() => {
  return sortedFields.value
    .filter((field) => field.required && field.type !== 'checkbox')
    .every((field) => {
      const value = values[field.key]
      return value !== undefined && value !== null && String(value).trim() !== ''
    })
})

const submitForm = handleSubmit((values) => {
  // Filter out empty values (except for checkboxes)
  const filteredValues = {}
  Object.keys(values).forEach((key) => {
    const field = sortedFields.value.find((f) => f.key === key)
    const value = values[key]
    if (
      (field?.type === 'checkbox' && value === true) ||
      (field?.type !== 'checkbox' &&
        value !== undefined &&
        value !== null &&
        String(value).trim() !== '')
    ) {
      filteredValues[key] = value
    }
  })

  emit('submit', {
    formData: filteredValues,
    message: props.handoffMode ? '' : messageText.value.trim()
  })
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

const focusFirstField = () => {
  nextTick(() => {
    const firstInput = formRef.value?.querySelector('input, textarea, select')
    firstInput?.focus()
  })
}

onMounted(focusFirstField)
watch(
  () => widgetStore.isOpen,
  (open) => {
    if (open) focusFirstField()
  }
)

// The server asked for this form with its current settings, so an empty form here means the widget's settings are stale.
let settingsRefreshed = false
const refreshSettings = async () => {
  if (settingsRefreshed) return
  settingsRefreshed = true
  try {
    const inboxID = new URLSearchParams(window.location.search).get('inbox_id')
    const resp = await api.getWidgetSettings(inboxID)
    widgetStore.updateConfig(resp.data.data)
  } catch (error) {
    console.error('Error refreshing widget settings:', error)
  }
}

// Auto-submit when no fields to show (e.g., all fields excluded)
watch(
  showForm,
  (newValue) => {
    if (newValue) return
    if (props.handoffMode) {
      refreshSettings()
      return
    }
    emit('submit', { formData: {}, message: '' })
  },
  { immediate: true }
)
</script>
