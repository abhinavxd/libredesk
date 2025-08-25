<template>
  <Dialog :open="openDialog" @update:open="openDialog = false">
    <DialogContent class="min-w-[40%] min-h-[30%]">
      <DialogHeader class="space-y-1">
        <DialogTitle
          >{{ view?.id ? $t('globals.messages.edit') : $t('globals.messages.create') }}
          {{ $t('globals.terms.view') }}
        </DialogTitle>
        <DialogDescription>
          {{ $t('view.form.description') }}
        </DialogDescription>
      </DialogHeader>
      <form @submit.prevent="onSubmit">
        <div class="grid gap-4 py-4">
          <FormField v-slot="{ componentField }" name="name">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
              <FormControl>
                <Input
                  id="name"
                  class="col-span-3"
                  placeholder=""
                  v-bind="componentField"
                  @keydown.enter.prevent="onSubmit"
                />
              </FormControl>
              <FormDescription>{{ $t('view.form.name.description') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
          <FormField v-slot="{ componentField }" name="filters">
            <FormItem>
              <FormLabel>Filters</FormLabel>
              <FormControl>
                <FilterBuilder
                  :fields="filterFields"
                  :showButtons="false"
                  v-bind="componentField"
                />
              </FormControl>
              <FormDescription> {{ $t('view.form.filters.description') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
        <DialogFooter>
          <Button type="submit" :disabled="isSubmitting" :isLoading="isSubmitting">
            {{ isSubmitting ? t('globals.messages.saving') : t('globals.messages.save') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import FilterBuilder from '@main/components/filter/FilterBuilder.vue'
import { useConversationFilters } from '../../composables/useConversationFilters'
import { toTypedSchema } from '@vee-validate/zod'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { useEmitter } from '../../composables/useEmitter'
import { handleHTTPError } from '../../utils/http'
import { OPERATOR } from '../../constants/filterConfig.js'
import { useI18n } from 'vue-i18n'
import { z } from 'zod'
import api from '../../api'

const emitter = useEmitter()
const { t } = useI18n()
const openDialog = defineModel('openDialog', { required: false, default: false })
const view = defineModel('view', { required: false, default: {} })
const isSubmitting = ref(false)
const { conversationsListFilters } = useConversationFilters()

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
  z.object({
    id: z.number().optional(),
    name: z
      .string()
      .min(2, { message: t('view.form.name.length') })
      .max(30, { message: t('view.form.name.length') }),
    filters: z
      .array(
        z.object({
          model: z.string({
            required_error: t('globals.messages.required', {
              name: t('globals.terms.filter').toLowerCase()
            })
          }),
          field: z.string({
            required_error: t('globals.messages.required', {
              name: t('globals.terms.field').toLowerCase()
            })
          }),
          operator: z.string({
            required_error: t('globals.messages.required', {
              name: t('globals.terms.operator').toLowerCase()
            })
          }),
          value: z.union([z.string(), z.number(), z.boolean()]).optional()
        })
      )
      .default([])
      .refine((filters) => filters.length > 0, { message: t('view.form.filter.selectAtLeastOne') })
      .refine(
        (filters) =>
          filters.every(
            (f) =>
              f.model &&
              f.field &&
              f.operator &&
              ([OPERATOR.SET, OPERATOR.NOT_SET].includes(f.operator) || f.value)
          ),
        {
          message: t('view.form.filter.partiallyFilled')
        }
      )
  })
)

const form = useForm({
  validationSchema: formSchema,
  validateOnMount: false,
  validateOnInput: false,
  validateOnBlur: false
})

const onSubmit = async () => {
  const validationResult = await form.validate()
  if (!validationResult.valid) return

  if (isSubmitting.value) return
  isSubmitting.value = true

  try {
    const values = form.values
    if (values.id) {
      await api.updateView(values.id, values)
    } else {
      await api.createView(values)
    }
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'view' })
    openDialog.value = false
    form.resetForm()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmitting.value = false
  }
}

// Set form values when view prop changes
watch(
  () => view.value,
  (newVal) => {
    if (newVal && Object.keys(newVal).length) {
      form.setValues(newVal)
    }
  },
  { immediate: true }
)
</script>
