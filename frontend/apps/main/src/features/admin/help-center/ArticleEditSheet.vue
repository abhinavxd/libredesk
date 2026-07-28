<template>
  <Sheet :open="isOpen" @update:open="$emit('update:open', $event)">
    <SheetContent class="!max-w-[80vw] sm:!max-w-[80vw] h-full p-0 flex flex-col">
      <div class="flex-1 flex flex-col min-h-0">
        <div class="flex items-center justify-between p-6 border-b bg-card/50">
          <div>
            <h2 class="text-lg font-semibold">
              {{ article ? t('helpCenter.editArticle') : t('helpCenter.newArticle') }}
            </h2>
            <p v-if="article" class="text-sm text-muted-foreground mt-1">
              {{ t('globals.terms.lastUpdated') }}: {{ format(new Date(article.updated_at), 'PPpp') }}
            </p>
          </div>
        </div>

        <div class="flex-1 flex min-h-0">
          <div class="flex-1 flex flex-col p-6 space-y-6 min-h-0">
            <form @submit="onSubmit" novalidate class="space-y-6 flex-1 flex flex-col min-h-0">
              <FormField v-slot="{ componentField }" name="title">
                <FormItem>
                  <FormControl>
                    <Input
                      type="text"
                      :placeholder="t('globals.terms.title')"
                      v-bind="componentField"
                      class="text-xl font-semibold border-0 px-0 py-3 shadow-none focus-visible:ring-0 placeholder:text-muted-foreground/60"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <FormField v-slot="{ componentField }" name="content">
                <FormItem class="flex-1 flex flex-col min-h-0">
                  <FormControl class="flex-1 min-h-0">
                    <div class="flex-1 flex flex-col min-h-0">
                      <Editor
                        v-model:htmlContent="componentField.modelValue"
                        @update:htmlContent="(value) => componentField.onChange(value)"
                        :placeholder="t('editor.newLine')"
                        enableInlineImages
                        linkedModel="help_articles"
                        class="min-h-[400px] border-0 px-0 shadow-none focus-visible:ring-0"
                      />
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <button type="submit" class="hidden" ref="submitButton"></button>
            </form>
          </div>

          <div class="w-80 border-l bg-muted/20 p-6 overflow-y-auto">
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

              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('globals.terms.status') }}
                </h3>

                <FormField v-slot="{ componentField }" name="status">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="draft">{{ t('globals.terms.draft') }}</SelectItem>
                          <SelectItem value="published">{{ t('helpCenter.published') }}</SelectItem>
                          <SelectItem value="archived">{{ t('helpCenter.archived') }}</SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div v-if="availableCollections.length > 0" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('globals.terms.collection') }}
                </h3>

                <FormField v-slot="{ componentField }" name="collection_id">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem
                            v-for="collection in availableCollections"
                            :key="collection.id"
                            :value="String(collection.id)"
                          >
                            {{ collection.name }}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <FormField v-slot="{ componentField, handleChange }" name="ai_enabled">
                <FormItem>
                  <SwitchField
                    :title="t('helpCenter.aiEnabled')"
                    :description="t('helpCenter.aiEnabledHint')"
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
                    <FormDescription>{{ t('helpCenter.localeHint') }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('helpCenter.excerpt') }}
                </h3>
                <FormField v-slot="{ componentField }" name="excerpt">
                  <FormItem>
                    <FormControl>
                      <Textarea :rows="3" :placeholder="derivedExcerpt || t('helpCenter.excerpt')" v-bind="componentField" />
                    </FormControl>
                    <FormDescription>{{ t('helpCenter.excerptHint') }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div class="space-y-3 border-t pt-4">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('helpCenter.seo') }}
                </h3>
                <FormField v-slot="{ componentField }" name="meta_title">
                  <FormItem>
                    <FormLabel>{{ t('helpCenter.metaTitle') }}</FormLabel>
                    <FormControl>
                      <Input type="text" :placeholder="metaTitlePlaceholder" v-bind="componentField" />
                    </FormControl>
                    <FormDescription>{{ t('helpCenter.metaTitleHint') }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="meta_description">
                  <FormItem>
                    <FormLabel>{{ t('helpCenter.metaDescription') }}</FormLabel>
                    <FormControl>
                      <Textarea :rows="2" :placeholder="metaDescriptionPlaceholder" v-bind="componentField" />
                    </FormControl>
                    <FormDescription>{{ t('helpCenter.metaDescriptionHint') }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="meta_image_url">
                  <FormItem>
                    <FormLabel>{{ t('helpCenter.metaImageURL') }}</FormLabel>
                    <FormControl>
                      <Input type="text" placeholder="https://" v-bind="componentField" />
                    </FormControl>
                    <FormDescription>{{ t('helpCenter.metaImageURLHint') }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div v-if="article" class="space-y-3 text-sm border-t pt-4">
                <div v-if="article.author_name" class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('helpCenter.author') }}</span>
                  <span>{{ article.author_name }}</span>
                </div>
                <div
                  v-if="article.helpful_count !== undefined"
                  class="flex justify-between py-1"
                >
                  <span class="text-muted-foreground">{{ t('helpCenter.feedback') }}</span>
                  <span>👍 {{ article.helpful_count }} · 👎 {{ article.not_helpful_count }}</span>
                </div>
                <div class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.createdAt') }}</span>
                  <span>{{ format(new Date(article.created_at), 'PPpp') }}</span>
                </div>
                <div class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.updatedAt') }}</span>
                  <span>{{ format(new Date(article.updated_at), 'PPpp') }}</span>
                </div>
                <div v-if="article.view_count !== undefined" class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.view', 2) }}</span>
                  <span>{{ article.view_count.toLocaleString() }}</span>
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
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { createArticleFormSchema } from './articleFormSchema.js'
import { useI18n } from 'vue-i18n'
import { getTextFromHTML } from '@shared-ui/utils/string.js'
import Editor from '@main/components/editor/ArticleEditor.vue'
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
  article: {
    type: Object,
    default: null
  },
  collectionId: {
    type: Number,
    default: null
  },
  helpCenterId: {
    type: Number,
    required: true
  },
  helpCenterName: {
    type: String,
    default: ''
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

const availableCollections = ref([])
const submitButton = ref(null)

const submitLabel = computed(() =>
  props.article ? t('globals.messages.update') : t('globals.messages.create')
)

const toFormValues = () => ({
  title: props.article?.title || '',
  content: props.article?.content || '',
  status: props.article?.status || 'draft',
  collection_id: String(props.article?.collection_id || props.collectionId || ''),
  sort_order: props.article?.sort_order || 0,
  ai_enabled: props.article?.ai_enabled || false,
  locale: props.article?.locale || props.defaultLocale || props.helpCenterLocales?.[0] || 'en',
  excerpt: props.article?.excerpt || '',
  meta_title: props.article?.meta_title || '',
  meta_description: props.article?.meta_description || '',
  meta_image_url: props.article?.meta_image_url || ''
})

const form = useForm({
  validationSchema: toTypedSchema(createArticleFormSchema(t)),
  initialValues: toFormValues()
})

// Placeholders preview what the public page falls back to when these fields are left blank.
const EXCERPT_LIMIT = 160

const derivedExcerpt = computed(() => {
  const text = getTextFromHTML(form.values.content || '').replace(/\s+/g, ' ').trim()
  if (text.length <= EXCERPT_LIMIT) return text
  const cut = text.slice(0, EXCERPT_LIMIT)
  const lastSpace = cut.lastIndexOf(' ')
  return lastSpace > 0 ? cut.slice(0, lastSpace) : cut
})

const metaTitlePlaceholder = computed(() => {
  const title = (form.values.title || '').trim()
  if (!title) return ''
  return props.helpCenterName ? `${title} - ${props.helpCenterName}` : title
})

const metaDescriptionPlaceholder = computed(
  () => (form.values.excerpt || '').trim() || derivedExcerpt.value
)

watch(
  () => [props.article, props.collectionId, props.isOpen],
  async () => {
    if (!props.isOpen) return
    await fetchAvailableCollections()
    form.resetForm({ values: toFormValues() })
  },
  { immediate: true }
)

const fetchAvailableCollections = async () => {
  try {
    const { data } = await api.getCollections(props.helpCenterId)
    availableCollections.value = data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  if (getTextFromHTML(values.content).length === 0) {
    values.content = ''
  }
  props.submitForm(values)
})

const handleSubmit = () => {
  if (submitButton.value) {
    submitButton.value.click()
  }
}
</script>
