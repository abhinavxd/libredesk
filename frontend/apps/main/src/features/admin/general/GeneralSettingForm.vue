<template>
  <form @submit="onSubmit" novalidate class="space-y-8 w-full">
    <fieldset class="workspace-fields min-w-0">
      <legend :class="SECTION_HEADING_CLASS">{{ i18n._('Workspace') }}</legend>
      <FormGrid class="workspace-grid">
        <FormField v-slot="{ field }" name="site_name">
          <FormItem>
            <FormLabel>{{ i18n._('Site name') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="field" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ field }" name="root_url">
          <FormItem>
            <FormLabel>
              {{ i18n._('Root URL') }}
            </FormLabel>
            <FormControl>
              <Input type="text" v-bind="field" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="lang">
          <FormItem>
            <FormLabel>{{ i18n.ngettext('Language', 'Languages', 1) }}</FormLabel>
            <FormControl>
              <Select v-bind="componentField" :modelValue="componentField.modelValue">
                <SelectTrigger>
                  <SelectValue :placeholder="i18n.pgettext('Application language setting', 'Select language')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem
                      v-for="lang in availableLanguages"
                      :key="lang.code"
                      :value="lang.code"
                    >
                      {{ lang.name }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="timezone">
          <FormItem>
            <FormLabel>
              {{ i18n.ngettext('Timezone', 'Timezones', 1) }}
            </FormLabel>
            <FormControl>
              <Select v-bind="componentField">
                <SelectTrigger>
                  <SelectValue :placeholder="i18n._('Select timezone')" />
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
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="business_hours_id">
          <FormItem>
            <FormLabel>
              {{ i18n.ngettext('Business hour', 'Business hours', 2) }}
            </FormLabel>
            <FormControl>
              <Select v-bind="componentField">
                <SelectTrigger>
                  <SelectValue :placeholder="i18n._('Select business hours')" />
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
            <FormMessage />
          </FormItem>
        </FormField>
      </FormGrid>
      <FormField v-slot="{ componentField, handleChange }" name="show_conversation_subject">
        <FormItem class="mt-5">
          <SwitchField
            :title="i18n._('Show subject in conversation list')"
            :description="i18n._('Display the conversation subject line in the conversation list when available.')"
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
        </FormItem>
      </FormField>
    </fieldset>
    <fieldset class="min-w-0">
      <legend :class="SECTION_HEADING_CLASS">{{ i18n._('Branding') }}</legend>
      <FormGrid>
        <FormField v-slot="{ field }" name="logo_url" :value="props.initialValues.logo_url">
          <FormItem>
            <FormLabel>{{ i18n._('Logo URL') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="field" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ field }" name="favicon_url" :value="props.initialValues.favicon_url">
          <FormItem>
            <FormLabel>{{ i18n._('Favicon URL') }}</FormLabel>
            <FormControl>
              <Input type="text" v-bind="field" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </FormGrid>
    </fieldset>
    <fieldset class="min-w-0">
      <legend :class="SECTION_HEADING_CLASS">{{ i18n.ngettext('File', 'Files', 2) }}</legend>
      <FormGrid>
        <FormField
          v-slot="{ field }"
          name="max_file_upload_size"
          :value="props.initialValues.max_file_upload_size"
        >
          <FormItem>
            <FormLabel>
              {{ i18n._('Max allowed file upload size') }}
            </FormLabel>
            <FormControl>
              <Input type="number" placeholder="10" v-bind="field" />
            </FormControl>
            <FormDescription>
              {{ i18n._('Max allowed file upload size in MB.') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField name="allowed_file_upload_extensions" v-slot="{ componentField, handleChange }">
          <FormItem>
            <FormLabel>
              {{ i18n._('Allowed file upload extensions') }}
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
              {{ i18n._('Use `*` to permit all file types. For example: `jpg, png, pdf`') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </FormGrid>
    </fieldset>
    <div class="flex border-t border-border pt-4">
      <Button type="submit" :isLoading="formLoading">{{ submitLabel }}</Button>
    </div>
  </form>
</template>

<script setup>
const SECTION_HEADING_CLASS = 'mb-4 ui-heading'

import FormGrid from '@shared-ui/components/FormGrid.vue'
import { watch, ref, onMounted } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
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
} from '@shared-ui/components/ui/form/index.js'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@shared-ui/components/ui/tags-input/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { timeZones } from '@main/constants/timezones.js'
import { useI18n } from '@shared-ui/i18n'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import api from '@main/api/index.js'

const emitter = useEmitter()
const i18n = useI18n()
const businessHours = ref({})
const formLoading = ref(false)
const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  availableLanguages: {
    type: Array,
    default: () => []
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

const submitLabel = props.submitLabel || i18n._('Save')
const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n))
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
      description: i18n._('Changes saved')
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
    form.setValues(newValues, false)
  },
  { deep: true }
)
</script>

<style scoped>
.workspace-fields {
  container-type: inline-size;
}

.workspace-grid.form-grid {
  grid-template-columns: minmax(0, 1fr);
}

@container (min-width: 32rem) {
  .workspace-grid.form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
