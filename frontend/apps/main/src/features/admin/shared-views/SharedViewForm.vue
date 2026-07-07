<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <FormField v-slot="{ componentField }" name="name" :validate-on-blur="false">
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
          <FilterGroupBuilder :fields="filterFields" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('view.form.filters.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="grid grid-cols-2 gap-4">
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
                  <SelectItem value="all">{{ t('sharedView.allAgents') }}</SelectItem>
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
              :placeholder="t('placeholders.selectTeam')"
              type="team"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <Button type="submit" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { ref, watch, computed, provide } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { Input } from '@shared-ui/components/ui/input'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import FilterGroupBuilder from '@/components/filter/FilterGroupBuilder.vue'
import {
  normalizeToTwoLevel,
  serializeFilterTree,
  deserializeFilterTree,
  collectLeaves,
  isPartialLeaf,
  createRoot
} from '@/components/filter/filterTree'
import { useConversationFilters } from '@/composables/useConversationFilters'
import { useTeamStore } from '@/stores/team'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { useI18n } from 'vue-i18n'
import { z } from 'zod'

const { conversationsListFilters } = useConversationFilters()
const { t } = useI18n()
const formLoading = ref(false)
const validateTick = ref(0)
provide('filterValidateTick', validateTick)
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
    (props.initialValues.id ? t('globals.messages.save') : t('globals.messages.create'))
  )
})

const filterFields = computed(() =>
  Object.entries(conversationsListFilters.value).map(([field, value]) => ({
    model: value.model || 'conversations',
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
        .object({
          logic: z.string().optional(),
          rules: z.array(z.any()).optional()
        })
        .passthrough()
        .default(() => createRoot()),
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
    visibility: props.initialValues.visibility || 'all',
    filters: createRoot()
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  const leaves = collectLeaves(values.filters)
  if (leaves.length === 0) {
    form.setFieldError('filters', t('view.form.filter.selectAtLeastOne'))
    return
  }
  if (leaves.some(isPartialLeaf)) {
    validateTick.value++
    return
  }

  const payload = { ...values, filters: serializeFilterTree(values.filters) }

  if (payload.visibility === 'all') {
    payload.team_id = null
  } else {
    payload.team_id = payload.team_id ? Number(payload.team_id) : null
  }

  props.submitForm(payload)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) return

    const processedVal = { ...newValues }
    processedVal.filters = deserializeFilterTree(
      normalizeToTwoLevel(newValues.filters),
      filterFields.value
    )

    // Convert team_id to string for the select component
    if (processedVal.team_id) {
      processedVal.team_id = String(processedVal.team_id)
    }

    form.setValues(processedVal)
  },
  { immediate: true }
)
</script>
