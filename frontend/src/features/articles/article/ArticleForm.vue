<template>
  <form @submit.prevent="onSubmit" class="space-y-6 w-full">
    <div class="flex" v-if="!props.isNewForm">
      <Button
        v-if="!isPublished"
        class="ml-auto"
        type="button"
        :isLoading="isPublishLoading"
        @click="publishArticle"
        :disabled="isDirty"
      >
        Publish
        <Rocket />
      </Button>
      <Button
        v-else
        class="ml-auto outline-dotted outline-2"
        type="button"
        :isLoading="isPublishLoading"
        @click="revertArticle"
        variant="outline"
      >
        Revert to Draft
        <Ban />
      </Button>
    </div>

    <!-- Article Title -->
    <FormField v-slot="{ componentField }" name="title">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.title') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" class="text-lg font-medium" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Section Selection -->
    <FormField v-slot="{ componentField }" name="section_id">
      <FormItem>
        <FormLabel class="flex items-center whitespace-nowrap">
          {{ $t('globals.terms.section') }}
        </FormLabel>
        <FormControl>
          <div class="relative">
            <ComboBox
              v-bind="componentField"
              :items="sections"
              :placeholder="$t('globals.messages.select', { name: $t('globals.terms.section') })"
              :disabled="isSectionLoading"
            >
              <template #item="{ item }">
                <div class="flex items-center gap-2">
                  <span class="text-sm">{{ item.name }}</span>
                </div>
              </template>

              <template #selected="{ selected }">
                <div class="flex items-center">
                  <span v-if="selected">{{ selected.name }}</span>
                </div>
              </template>
            </ComboBox>
            <div v-if="isSectionLoading" class="absolute inset-0 flex items-center justify-center">
              <Loader2 class="h-4 w-4 animate-spin" />
            </div>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="content">
      <FormItem>
        <FormControl>
          <CodeHtmlPreview
            v-model="componentField.modelValue"
            @update:modelValue="handleChange"
            :customScript="customScript"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <Button type="submit" :isLoading="isLoading"> Submit </Button>
  </form>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api/index.js'
import CodeHtmlPreview from '@/components/editor/CodeHtmlPreview.vue'
import Button from '@/components/ui/button/Button.vue'
import { Loader2, Rocket, Ban } from 'lucide-vue-next'

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
  isNewForm: {
    type: Boolean,
    required: false,
    default: false
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})

const { t } = useI18n()
const emitter = useEmitter()
const sections = ref([])
const isSectionLoading = ref(false)
const isPublished = ref(props.initialValues?.is_published ?? false)
const isPublishLoading = ref(false)
const customScript = ref('')

// Store the last saved values for isDirty check
const lastSavedValues = ref({})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: props.initialValues
})

// save form
const onSubmit = form.handleSubmit(async (values) => {
  await props.submitForm(values)
  // Update last saved values after successful save
  lastSavedValues.value = {
    title: values.title,
    section_id: values.section_id,
    content: values.content
  }
})

const isDirty = computed(() => {
  if (props.isNewForm) return true
  if (!form.values || !lastSavedValues.value) return true

  const formFields = ['title', 'section_id', 'content']

  return formFields.some((field) => {
    const currentValue = form.values[field]
    const savedValue = lastSavedValues.value[field]

    // Handle null, undefined, and empty strings as equivalent
    const normalize = (val) => {
      if (val === null || val === undefined || val === '') return ''
      return String(val).trim()
    }

    return normalize(currentValue) !== normalize(savedValue)
  })
})

// Watch for changes in props.initialValues (in case parent updates them)
watch(
  () => props.initialValues,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      lastSavedValues.value = {
        title: newValues.title,
        section_id: newValues.section_id,
        content: newValues.content
      }
    }
  },
  { deep: true }
)

// Fetch sections
const fetchData = async () => {
  try {
    isSectionLoading.value = true
    const [sectionsResp, settingsResp] = await Promise.all([
      api.getArticleSections(),
      api.getSettings('article')
    ])

    // Process the responses
    customScript.value = settingsResp?.data?.data?.['article.custom_script'] ?? ''

    sections.value =
      sectionsResp?.data?.data?.map((sec) => ({
        value: sec.id,
        name: sec.name
      })) ?? []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.errorFetching')
    })
  } finally {
    isSectionLoading.value = false
  }
}

onMounted(() => {
  fetchData()

  // Set initial values if provided
  if (Object.keys(props.initialValues).length > 0) {
    form.setValues(props.initialValues)
    // Initialize lastSavedValues with the initial values
    lastSavedValues.value = {
      title: props.initialValues.title,
      section_id: props.initialValues.section_id,
      content: props.initialValues.content
    }
  }
})

const publishArticle = async () => {
  if (isDirty.value) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Please save the current form first.'
    })
    return
  }

  try {
    isPublishLoading.value = true
    await api.publishArticle(props.initialValues.id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.successfullyUpdated', { name: t('globals.terms.article') })
    })
    isPublished.value = true
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.errorUpdating', { name: t('globals.terms.article') })
    })
  } finally {
    isPublishLoading.value = false
  }
}

const revertArticle = async () => {
  try {
    isPublishLoading.value = true
    await api.unpublishArticle(props.initialValues.id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.successfullyUpdated', { name: t('globals.terms.article') })
    })
    isPublished.value = false
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.errorUpdating', { name: t('globals.terms.article') })
    })
  } finally {
    isPublishLoading.value = false
  }
}
</script>
