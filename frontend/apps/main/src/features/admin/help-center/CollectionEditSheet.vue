<template>
  <Sheet :open="isOpen" @update:open="$emit('update:open', $event)">
    <SheetContent class="!max-w-[60vw] sm:!max-w-[60vw] h-full p-0 flex flex-col">
      <div class="flex-1 flex flex-col min-h-0">
        <div class="flex items-center justify-between p-6 border-b bg-card/50">
          <div>
            <SheetTitle>
              {{ collection ? t('helpCenter.editCollection') : t('helpCenter.newCollection') }}
            </SheetTitle>
            <SheetDescription v-if="collection" class="mt-1">
              {{ t('globals.terms.lastUpdated') }}:
              {{ format(new Date(collection.updated_at), 'PPpp') }}
            </SheetDescription>
          </div>
        </div>

        <div class="flex-1 flex min-h-0">
          <div class="flex-1 flex flex-col p-6 space-y-6 min-h-0 overflow-y-auto">
            <form @submit="onSubmit" novalidate class="space-y-4 flex-1 flex flex-col min-h-0">
              <FormField v-slot="{ componentField }" name="name">
                <FormItem>
                  <FormControl>
                    <Input
                      ref="nameInput"
                      type="text"
                      :placeholder="t('globals.terms.name')"
                      v-bind="componentField"
                      class="text-xl font-semibold border-0 px-0 py-3 shadow-none focus-visible:ring-0 placeholder:text-muted-foreground/60"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <FormField v-slot="{ componentField }" name="description">
                <FormItem class="flex-1 flex flex-col min-h-0">
                  <FormControl class="flex-1 min-h-0">
                    <Textarea
                      ref="descriptionInput"
                      :placeholder="t('globals.terms.description')"
                      v-bind="componentField"
                      class="h-full border-0 px-0 py-2 shadow-none focus-visible:ring-0 resize-none placeholder:text-muted-foreground/60"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </form>
          </div>

          <div class="w-72 border-l bg-muted/20 p-6 overflow-y-auto">
            <div class="space-y-6">
              <div class="space-y-4">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('globals.terms.action', 2) }}
                </h3>

                <div class="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    @click="$emit('cancel')"
                    class="flex-1"
                  >
                    {{ t('globals.messages.cancel') }}
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    @click="onSubmit"
                    :isLoading="isLoading"
                    class="flex-1"
                  >
                    {{ submitLabel }}
                  </Button>
                </div>
              </div>

              <FormField v-slot="{ componentField, handleChange }" name="is_published">
                <FormItem>
                  <SwitchField
                    :title="t('globals.terms.published')"
                    :checked="componentField.modelValue"
                    @update:checked="handleChange"
                  />
                </FormItem>
              </FormField>

              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('globals.terms.icon') }}
                </h3>
                <FormField v-slot="{ value, handleChange }" name="icon">
                  <FormItem>
                    <FormControl>
                      <IconPicker :model-value="value" @update:model-value="handleChange" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('globals.terms.language') }}
                </h3>
                <FormField v-slot="{ componentField }" name="locale">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem v-for="loc in helpCenterLocales" :key="loc" :value="loc">
                            {{ loc }}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div v-if="localeParents.length > 0" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('helpCenter.parentCollection') }}
                </h3>

                <FormField v-slot="{ componentField }" name="parent_id">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue>{{ parentLabel }}</SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
                          <SelectItem
                            v-for="parent in localeParents"
                            :key="parent.id"
                            :value="String(parent.id)"
                          >
                            {{ parent.name }}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div v-if="collection" class="space-y-3 text-sm border-t pt-4">
                <div class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.createdAt') }}</span>
                  <span>{{ format(new Date(collection.created_at), 'PPpp') }}</span>
                </div>
                <div class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.updatedAt') }}</span>
                  <span>{{ format(new Date(collection.updated_at), 'PPpp') }}</span>
                </div>
                <div v-if="collection.articles" class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.article', 2) }}</span>
                  <Badge variant="outline">{{ collection.articles.length }}</Badge>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </SheetContent>
  </Sheet>
</template>

<script setup>
import { ref, watch, computed, nextTick } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Badge } from '@shared-ui/components/ui/badge'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Sheet, SheetContent, SheetTitle, SheetDescription } from '@shared-ui/components/ui/sheet'
import {
  FormControl,
  FormField,
  FormItem,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { createCollectionFormSchema } from './collectionFormSchema.js'
import IconPicker from './IconPicker.vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { format } from 'date-fns'

const { t } = useI18n()

const props = defineProps({
  isOpen: {
    type: Boolean,
    default: false
  },
  collection: {
    type: Object,
    default: null
  },
  helpCenterId: {
    type: Number,
    required: true
  },
  parentId: {
    type: Number,
    default: null
  },
  helpCenterLocales: {
    type: Array,
    default: () => ['en']
  },
  defaultLocale: {
    type: String,
    default: ''
  },
  submitForm: {
    type: Function,
    required: true
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

defineEmits(['update:open', 'cancel'])
const emitter = useEmitter()

// Mirrors maxCollectionDepth on the backend.
const MAX_DEPTH = 3

const availableParents = ref([])
const nameInput = ref(null)
const descriptionInput = ref(null)

const submitLabel = computed(() =>
  props.collection ? t('globals.messages.update') : t('globals.messages.create')
)

const toFormValues = () => ({
  name: props.collection?.name || '',
  description: props.collection?.description || '',
  icon: props.collection?.icon || '',
  parent_id: String(props.collection?.parent_id || props.parentId || 0),
  is_published: props.collection?.is_published ?? true,
  sort_order: props.collection?.sort_order || 0,
  locale: props.collection?.locale || props.defaultLocale || props.helpCenterLocales?.[0] || 'en'
})

const form = useForm({
  validationSchema: toTypedSchema(createCollectionFormSchema(t)),
  initialValues: toFormValues()
})

// A collection and its parent must share a language, else the child drops out of that
// language's tree.
const localeParents = computed(() =>
  availableParents.value.filter((parent) => parent.locale === form.values.locale)
)

// The select only learns an option's text when that option mounts, and the parent list
// arrives after the value is set, so the label is resolved here instead.
const parentLabel = computed(() => {
  const id = String(form.values.parent_id ?? '0')
  if (id === '0') return t('globals.terms.none')
  return localeParents.value.find((parent) => String(parent.id) === id)?.name || ''
})

watch(localeParents, (parents) => {
  const current = Number(form.values.parent_id)
  if (current && !parents.some((parent) => parent.id === current)) {
    form.setFieldValue('parent_id', '0', false)
  }
})

watch(
  () => [props.collection, props.parentId, props.isOpen],
  async () => {
    if (!props.isOpen) return
    await fetchAvailableParents()
    form.resetForm({ values: toFormValues() })
    await nextTick()
    if (form.values.description) descriptionInput.value?.$el?.focus()
    else nameInput.value?.$el?.focus()
  },
  { immediate: true }
)

const fetchAvailableParents = async () => {
  try {
    const { data } = await api.getCollections(props.helpCenterId)
    const collections = data.data || []
    // Exclude the collection itself and every descendant, since re-parenting into
    // its own subtree would form a cycle.
    const excluded = new Set()
    if (props.collection) {
      excluded.add(props.collection.id)
      let grew = true
      while (grew) {
        grew = false
        for (const collection of collections) {
          if (
            collection.parent_id &&
            excluded.has(collection.parent_id) &&
            !excluded.has(collection.id)
          ) {
            excluded.add(collection.id)
            grew = true
          }
        }
      }
    }
    // Nesting is capped at MAX_DEPTH levels, so a parent is only offered when it can still
    // hold this collection's whole subtree beneath it.
    const byId = new Map(collections.map((collection) => [collection.id, collection]))
    const depthOf = (collection) => {
      let depth = 1
      let current = collection
      while (current?.parent_id && byId.has(current.parent_id)) {
        current = byId.get(current.parent_id)
        depth++
      }
      return depth
    }
    const heightOf = (id) => {
      const children = collections.filter((collection) => collection.parent_id === id)
      return children.length ? 1 + Math.max(...children.map((child) => heightOf(child.id))) : 1
    }
    const subtreeHeight = props.collection ? heightOf(props.collection.id) : 1
    availableParents.value = collections.filter(
      (collection) =>
        !excluded.has(collection.id) && depthOf(collection) + subtreeHeight <= MAX_DEPTH
    )
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  props.submitForm({ ...values, parent_id: values.parent_id ? values.parent_id : null })
})
</script>
