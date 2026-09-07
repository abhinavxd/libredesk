<template>
  <Dialog :open="openDialog" @update:open="openDialog = false">
    <DialogContent class="w-[min(92vw,960px)] max-w-4xl">
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
        <div class="grid gap-5 py-4">
          <FormField v-slot="{ componentField }" name="name" :validate-on-blur="false">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
              <FormControl>
                <Input
                  ref="nameInputRef"
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
              <FormLabel>{{ $t('globals.terms.filter', 2) }}</FormLabel>
              <FormControl>
                <FilterGroupBuilder :fields="filterFields" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{ $t('view.form.filters.description') }}</FormDescription>
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
import { ref, computed, watch, nextTick, provide } from 'vue'
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
import FilterGroupBuilder from '@main/components/filter/FilterGroupBuilder.vue'
import {
  normalizeToTwoLevel,
  serializeFilterTree,
  deserializeFilterTree,
  collectLeaves,
  isPartialLeaf,
  createRoot
} from '@main/components/filter/filterTree'
import { useConversationFilters } from '../../composables/useConversationFilters'
import { toTypedSchema } from '@vee-validate/zod'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { useEmitter } from '../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import { z } from 'zod'
import api from '@/api'

const emitter = useEmitter()
const { t } = useI18n()
const nameInputRef = ref(null)
const openDialog = defineModel('openDialog', { required: false, default: false })
watch(openDialog, (isOpen) => {
  if (isOpen) {
    // A cancelled edit leaves the previous view in the form; reset before a fresh create.
    if (!view.value?.id) form.resetForm()
    nextTick(() => {
      nameInputRef.value?.$el?.focus()
    })
  }
})
const view = defineModel('view', { required: false, default: {} })
const isSubmitting = ref(false)
const validateTick = ref(0)
provide('filterValidateTick', validateTick)
const { conversationsListFilters } = useConversationFilters()

const filterFields = computed(() =>
  Object.entries(conversationsListFilters.value).map(([field, value]) => ({
    model: value.model || 'conversations',
    label: value.label,
    field,
    type: value.type,
    operators: value.operators,
    options: value.options ?? [],
    entity: value.entity
  }))
)
const formSchema = toTypedSchema(
  z.object({
    id: z.number().optional(),
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
      .default(() => createRoot())
  })
)

const form = useForm({
  validationSchema: formSchema,
  initialValues: {
    filters: createRoot()
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  if (isSubmitting.value) return

  const leaves = collectLeaves(values.filters)
  if (leaves.length === 0) {
    form.setFieldError('filters', t('view.form.filter.selectAtLeastOne'))
    return
  }
  if (leaves.some(isPartialLeaf)) {
    validateTick.value++
    return
  }

  isSubmitting.value = true

  try {
    const payload = { ...values, filters: serializeFilterTree(values.filters) }

    let viewID = payload.id
    if (viewID) {
      await api.updateView(viewID, payload)
    } else {
      const resp = await api.createView(payload)
      viewID = resp?.data?.data?.id
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'view', id: viewID })
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
})

watch(
  () => view.value,
  (newVal) => {
    if (newVal && Object.keys(newVal).length) {
      const processedVal = { ...newVal }
      processedVal.filters = deserializeFilterTree(
        normalizeToTwoLevel(newVal.filters),
        filterFields.value
      )
      form.setValues(processedVal, false)
    }
  },
  { immediate: true }
)
</script>
