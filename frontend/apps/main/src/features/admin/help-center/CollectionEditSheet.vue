<template>
  <Sheet :open="isOpen" @update:open="$emit('update:open', $event)">
    <SheetContent class="!max-w-[60vw] sm:!max-w-[60vw] h-full p-0 flex flex-col">
      <div class="flex-1 flex flex-col min-h-0">
        <div class="flex items-center justify-between p-6 border-b bg-card/50">
          <div>
            <h2 class="text-lg font-semibold">
              {{ collection ? t('helpCenter.editCollection') : t('helpCenter.newCollection') }}
            </h2>
            <p v-if="collection" class="text-sm text-muted-foreground mt-1">
              {{ t('globals.terms.lastUpdated') }}:
              {{ format(new Date(collection.updated_at), 'PPpp') }}
            </p>
          </div>
        </div>

        <div class="flex-1 flex min-h-0">
          <div class="flex-1 flex flex-col p-6 space-y-6 overflow-y-auto">
            <form @submit="onSubmit" novalidate class="space-y-6 flex-1 flex flex-col">
              <FormField v-slot="{ componentField }" name="name">
                <FormItem>
                  <FormControl>
                    <Input
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
                <FormItem class="flex-1">
                  <FormControl>
                    <Textarea
                      :placeholder="t('globals.terms.description')"
                      rows="6"
                      v-bind="componentField"
                      class="border-0 px-0 py-2 shadow-none focus-visible:ring-0 resize-none placeholder:text-muted-foreground/60"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <button type="submit" class="hidden" ref="submitButton"></button>
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
                    @click="handleSubmit"
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
                    :title="t('helpCenter.published')"
                    :checked="componentField.modelValue"
                    @update:checked="handleChange"
                  />
                </FormItem>
              </FormField>

              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('helpCenter.language') }}
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

              <div v-if="availableParents.length > 0" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('helpCenter.parentCollection') }}
                </h3>

                <FormField v-slot="{ componentField }" name="parent_id">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
                          <SelectItem
                            v-for="parent in availableParents"
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
import { ref, watch, computed } from 'vue'
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
import { Sheet, SheetContent } from '@shared-ui/components/ui/sheet'
import {
  FormControl,
  FormField,
  FormItem,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { createCollectionFormSchema } from './collectionFormSchema.js'
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

const availableParents = ref([])
const submitButton = ref(null)

const submitLabel = computed(() =>
  props.collection ? t('globals.messages.update') : t('globals.messages.create')
)

const toFormValues = () => ({
  name: props.collection?.name || '',
  description: props.collection?.description || '',
  parent_id: String(props.collection?.parent_id || props.parentId || 0),
  is_published: props.collection?.is_published ?? true,
  sort_order: props.collection?.sort_order || 0,
  locale: props.collection?.locale || props.defaultLocale || props.helpCenterLocales?.[0] || 'en'
})

const form = useForm({
  validationSchema: toTypedSchema(createCollectionFormSchema(t)),
  initialValues: toFormValues()
})

watch(
  () => [props.collection, props.parentId, props.isOpen],
  async () => {
    if (!props.isOpen) return
    await fetchAvailableParents()
    form.resetForm({ values: toFormValues() })
  },
  { immediate: true }
)

const fetchAvailableParents = async () => {
  try {
    const { data } = await api.getCollections(props.helpCenterId)
    availableParents.value = (data.data || []).filter((collection) => {
      if (props.collection && collection.id === props.collection.id) return false
      if (props.collection && collection.parent_id === props.collection.id) return false
      return true
    })
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

const handleSubmit = () => {
  if (submitButton.value) {
    submitButton.value.click()
  }
}
</script>
