<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <FormField v-slot="{ field }" name="site_name">
      <FormItem>
        <FormLabel>{{ t('admin.general.siteName') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormDescription>
          {{ t('admin.general.siteName.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="lang">
      <FormItem>
        <FormLabel>{{ t('globals.terms.language') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField" :modelValue="componentField.modelValue">
            <SelectTrigger>
              <SelectValue :placeholder="t('admin.general.language.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="en"> English </SelectItem>
                <SelectItem value="mr"> Marathi </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>
          {{ t('admin.general.language.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="timezone">
      <FormItem>
        <FormLabel>
          {{ t('globals.terms.timezone') }}
        </FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="t('admin.general.timezone.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="(value, label) in timeZones" :key="value" :value="value">
                  {{ label }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>
          {{ t('admin.general.timezone.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="business_hours_id">
      <FormItem>
        <FormLabel>
          {{ t('globals.terms.businessHour', 2) }}
        </FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="t('admin.general.businessHours.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="bh in businessHours" :key="bh.id" :value="bh.id">
                  {{ bh.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>
          {{ t('admin.general.businessHours.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="root_url">
      <FormItem>
        <FormLabel>
          {{ t('globals.terms.rootURL') }}
        </FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormDescription>
          {{ t('admin.general.rootURL.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="favicon_url" :value="props.initialValues.favicon_url">
      <FormItem>
        <FormLabel>{{ t('admin.general.faviconURL') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormDescription>{{ t('admin.general.faviconURL.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="logo_url" :value="props.initialValues.logo_url">
      <FormItem>
        <FormLabel>{{ t('admin.general.logoURL') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormDescription>{{ t('admin.general.logoURL.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField
      v-slot="{ field }"
      name="max_file_upload_size"
      :value="props.initialValues.max_file_upload_size"
    >
      <FormItem>
        <FormLabel>
          {{ t('admin.general.maxAllowedFileUploadSize') }}
        </FormLabel>
        <FormControl>
          <Input type="number" placeholder="10" v-bind="field" />
        </FormControl>
        <FormDescription>
          {{ t('admin.general.maxAllowedFileUploadSize.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="allowed_file_upload_extensions" v-slot="{ componentField, handleChange }">
      <FormItem>
        <FormLabel>
          {{ t('admin.general.allowedFileUploadExtensions') }}
        </FormLabel>
        <FormControl>
          <TagsInput :modelValue="componentField.modelValue" @update:modelValue="handleChange">
            <TagsInputItem v-for="item in componentField.modelValue" :key="item" :value="item">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput placeholder="jpg" />
          </TagsInput>
        </FormControl>
        <FormDescription>
          {{ t('admin.general.allowedFileUploadExtensions.description') }}
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="formLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@/components/ui/tags-input'
import { Input } from '@/components/ui/input'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { timeZones } from '@/constants/timezones.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const emitter = useEmitter()
const { t } = useI18n()
const businessHours = ref({})
const formLoading = ref(false)
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
    default: ''
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const submitLabel = props.submitLabel || t('globals.messages.save')
const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

onMounted(() => {
  fetchBusinessHours()
})

const fetchBusinessHours = async () => {
  try {
    const response = await api.getAllBusinessHours()
    // Convert business hours id to string
    response.data.data.forEach((bh) => {
      bh.id = bh.id.toString()
    })
    businessHours.value = response.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await props.submitForm(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.setting', 2)
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    // Convert business hours id to string
    if (newValues.business_hours_id)
      newValues.business_hours_id = newValues.business_hours_id.toString()
    form.setValues(newValues)
  },
  { deep: true }
)
</script>
