<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <FormField v-slot="{ field }" name="site_name">
      <FormItem>
        <FormLabel>Site Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Site Name" v-bind="field" />
        </FormControl>
        <FormDescription>Enter the site name</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="lang">
      <FormItem>
        <FormLabel>Language</FormLabel>
        <FormControl>
          <Select v-bind="componentField" :modelValue="componentField.modelValue">
            <SelectTrigger>
              <SelectValue placeholder="Select a language" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="en"> English </SelectItem>
                <SelectItem value="es"> Spanish </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>Select language for the app.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="timezone">
      <FormItem>
        <FormLabel>Timezone</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select a timezone" />
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
        <FormDescription>Default timezone for your desk.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="business_hours_id">
      <FormItem>
        <FormLabel>Business hours</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select business hours" />
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
        <FormDescription>Default business hours for your desk.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="root_url">
      <FormItem>
        <FormLabel>Root URL</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Root URL" v-bind="field" />
        </FormControl>
        <FormDescription>Root URL of the app. (No trailing slash)</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="favicon_url" :value="props.initialValues.favicon_url">
      <FormItem>
        <FormLabel>Favicon URL</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Favicon URL" v-bind="field" />
        </FormControl>
        <FormDescription>Favicon URL for the app.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="logo_url" :value="props.initialValues.logo_url">
      <FormItem>
        <FormLabel>Logo URL</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Logo URL" v-bind="field" />
        </FormControl>
        <FormDescription>Logo URL for the app.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField
      v-slot="{ field }"
      name="max_file_upload_size"
      :value="props.initialValues.max_file_upload_size"
    >
      <FormItem>
        <FormLabel>Max allowed file upload size</FormLabel>
        <FormControl>
          <Input type="number" placeholder="10" v-bind="field" />
        </FormControl>
        <FormDescription>In megabytes.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="allowed_file_upload_extensions" v-slot="{ componentField, handleChange }">
      <FormItem>
        <FormLabel>Allowed file upload extensions</FormLabel>
        <FormControl>
          <TagsInput :modelValue="componentField.modelValue" @update:modelValue="handleChange">
            <TagsInputItem v-for="item in componentField.modelValue" :key="item" :value="item">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput placeholder="jpg" />
          </TagsInput>
        </FormControl>
        <FormDescription>Use `*` to allow any file.</FormDescription>
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
import { formSchema } from './formSchema.js'
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
import api from '@/api'

const emitter = useEmitter()
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
    default: () => 'Submit'
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
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
    // If unauthorized (no permission), show a toast message.
    if (error.response.status === 403) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Unauthorized',
        variant: 'destructive',
        description: 'You do not have permission to view business hours.'
      })
    } else {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Could not fetch business hours',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await props.submitForm(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      description: 'Settings updated successfully'
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
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
