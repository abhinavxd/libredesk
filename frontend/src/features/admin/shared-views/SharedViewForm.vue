<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('view.form.name.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="filters">
      <FormItem>
        <FormLabel>{{ t('globals.terms.filter', 2) }}</FormLabel>
        <FormControl>
          <FilterBuilder :fields="filterFields" :showButtons="false" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('view.form.filters.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField
      v-slot="{ componentField }"
      name="visibility"
      :validate-on-blur="false"
      :validate-on-change="false"
      :validate-on-input="false"
      :validate-on-mount="false"
      :validate-on-model-update="false"
    >
      <FormItem>
        <FormLabel>{{ t('globals.terms.visibility') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="all">{{
                  t('globals.messages.all', {
                    name: t('globals.terms.agent', 2).toLowerCase()
                  })
                }}</SelectItem>
                <SelectItem value="team">{{ t('globals.terms.team') }}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-if="form.values.visibility === 'team'" v-slot="{ componentField }" name="team_id">
      <FormItem>
        <FormLabel>{{ t('globals.terms.team') }}</FormLabel>
        <FormControl>
          <SelectComboBox
            v-bind="componentField"
            :items="tStore.options"
            :placeholder="t('globals.messages.select', { name: t('globals.terms.team').toLowerCase() })"
            type="team"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { Input } from '@/components/ui/input'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import FilterBuilder from '@/components/filter/FilterBuilder.vue'
import { useConversationFilters } from '@/composables/useConversationFilters'
import { useTeamStore } from '@/stores/team'
import { OPERATOR, FIELD_TYPE } from '@/constants/filterConfig.js'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { useI18n } from 'vue-i18n'
import { z } from 'zod'

const { conversationsListFilters } = useConversationFilters()
const { t } = useI18n()
const formLoading = ref(false)
const tStore = useTeamStore()
const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.initialValues.id ? t('globals.messages.update') : t('globals.messages.create'))
  )
})

const filterFields = computed(() =>
  Object.entries(conversationsListFilters.value).map(([field, value]) => ({
    model: 'conversations',
    label: value.label,
    field,
    type: value.type,
    operators: value.operators,
    options: value.options ?? []
  }))
)

const formSchema = toTypedSchema(
  z
    .object({
      name: z
        .string({
          required_error: t('globals.messages.required')
        })
        .min(2, { message: t('view.form.name.length') })
        .max(140, { message: t('view.form.name.length') }),
      filters: z
        .array(
          z.object({
            model: z.string().optional(),
            field: z.string().optional(),
            operator: z.string().optional(),
            value: z
              .union([
                z.string(),
                z.number(),
                z.boolean(),
                z.array(z.union([z.string(), z.number()]))
              ])
              .optional()
          })
        )
        .default([]),
      visibility: z.enum(['all', 'team']),
      team_id: z.string().nullable().optional()
    })
    .refine(
      (data) => {
        if (data.visibility === 'team') return !!data.team_id
        return true
      },
      { message: t('globals.messages.required'), path: ['team_id'] }
    )
)

const form = useForm({
  validationSchema: formSchema,
  initialValues: {
    visibility: props.initialValues.visibility || 'all'
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  // Make sure at least one filter is selected
  if (!values.filters || values.filters.length === 0) {
    form.setFieldError('filters', t('view.form.filter.selectAtLeastOne'))
    return
  }

  // Check for partial filters
  const hasPartialFilters = values.filters.some(
    (f) =>
      !f.field ||
      !f.operator ||
      (![OPERATOR.SET, OPERATOR.NOT_SET].includes(f.operator) &&
        (!f.value || (Array.isArray(f.value) && f.value.length === 0)))
  )
  if (hasPartialFilters) {
    form.setFieldError('filters', t('view.form.filter.partiallyFilled'))
    return
  }

  // Serialize array values to JSON strings for backend
  if (values.filters) {
    values.filters = values.filters.map((filter) => {
      if (Array.isArray(filter.value)) {
        const numericValues = filter.value.map((v) => {
          const num = Number(v)
          return isNaN(num) ? v : num
        })
        return { ...filter, value: JSON.stringify(numericValues) }
      }
      return filter
    })
  }

  // Clear team_id if visibility is 'all', otherwise convert to number
  if (values.visibility === 'all') {
    values.team_id = null
  } else {
    values.team_id = values.team_id ? Number(values.team_id) : null
  }

  props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) return

    // Deserialize multi-select filter values from JSON strings to arrays
    const processedVal = { ...newValues }
    if (processedVal.filters) {
      processedVal.filters = processedVal.filters.map((filter) => {
        const field = filterFields.value.find((f) => f.field === filter.field)
        const isMultiSelectField = field?.type === FIELD_TYPE.MULTI_SELECT

        if (isMultiSelectField && typeof filter.value === 'string') {
          try {
            const parsed = JSON.parse(filter.value)
            const stringValues = Array.isArray(parsed) ? parsed.map((v) => String(v)) : parsed
            return { ...filter, value: stringValues }
          } catch (e) {
            return filter
          }
        }
        return filter
      })
    }

    // Convert team_id to string for the select component
    if (processedVal.team_id) {
      processedVal.team_id = String(processedVal.team_id)
    }

    form.setValues(processedVal)
  },
  { immediate: true }
)
</script>
