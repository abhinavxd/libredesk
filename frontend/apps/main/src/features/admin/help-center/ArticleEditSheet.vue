<template>
  <Sheet :open="isOpen" @update:open="$emit('update:open', $event)">
    <SheetContent class="!max-w-[80vw] sm:!max-w-[80vw] h-full p-0 flex flex-col">
      <div class="flex-1 flex flex-col min-h-0">
        <div class="flex items-center justify-between p-6 border-b bg-card/50">
          <div>
            <SheetTitle>
              {{
                translationSource
                  ? t('helpCenter.addTranslation')
                  : article
                    ? t('helpCenter.editArticle')
                    : t('helpCenter.newArticle')
              }}
            </SheetTitle>
            <SheetDescription v-if="translationSource" class="mt-1">
              {{ t('helpCenter.translationOf', { title: translationSource.title }) }}
            </SheetDescription>
            <SheetDescription v-if="loadedArticle" class="mt-1">
              {{ t('globals.terms.lastUpdated') }}:
              {{ formatDate(loadedArticle.updated_at) }}
            </SheetDescription>
          </div>
        </div>

        <Spinner v-if="isLoadingArticle" class="flex-1" />

        <div v-else class="flex-1 flex min-h-0">
          <div class="flex-1 flex flex-col p-6 space-y-6 min-h-0">
            <form @submit="onSubmit" novalidate class="space-y-4 flex-1 flex flex-col min-h-0">
              <div ref="toolbarSlot" />

              <FormField v-slot="{ componentField }" name="title">
                <FormItem>
                  <FormControl>
                    <Input
                      ref="titleInput"
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
                        ref="editorRef"
                        :auto-focus="false"
                        v-model:textContent="editorText"
                        :htmlContent="componentField.modelValue"
                        @update:htmlContent="(value) => componentField.onChange(value)"
                        :placeholder="t('helpCenter.articlePlaceholder')"
                        enableInlineImages
                        linkedModel="help_articles"
                        :toolbarTarget="toolbarSlot"
                        class="min-h-[400px] border-0 px-0 shadow-none focus-visible:ring-0"
                      />
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
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
                    @click="onSubmit"
                    :isLoading="isLoading"
                    class="flex-1"
                  >
                    {{ submitLabel }}
                  </Button>
                </div>
              </div>

              <div v-if="loadedArticle && helpCenterLocales.length > 1" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('globals.terms.language', 2) }}
                </h3>

                <div class="overflow-hidden rounded-md border bg-card">
                  <button
                    v-for="locale in translatedLocales"
                    :key="locale"
                    type="button"
                    class="flex w-full items-center gap-3 border-b px-3 py-2.5 text-left text-sm last:border-b-0 disabled:cursor-default can-hover:hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                    :class="{ 'font-medium': locale === loadedArticle.locale }"
                    :disabled="locale === loadedArticle.locale"
                    :aria-current="locale === loadedArticle.locale ? 'page' : undefined"
                    @click="selectTranslationLocale(locale)"
                  >
                    <span class="min-w-0 flex-1 truncate">{{ languageName(locale) }}</span>
                    <span class="text-xs uppercase text-muted-foreground">{{ locale }}</span>
                    <Check
                      v-if="locale === loadedArticle.locale"
                      class="size-4 shrink-0"
                      aria-hidden="true"
                    />
                    <ChevronRight
                      v-else-if="translationByLocale.has(locale)"
                      class="size-4 shrink-0 text-muted-foreground"
                      aria-hidden="true"
                    />
                  </button>
                </div>

                <Select
                  v-if="missingLocales.length"
                  :model-value="''"
                  @update:model-value="selectTranslationLocale"
                >
                  <SelectTrigger class="w-full">
                    <div class="flex items-center gap-2">
                      <Plus class="size-4" aria-hidden="true" />
                      <SelectValue :placeholder="t('helpCenter.addTranslation')" />
                    </div>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="locale in missingLocales" :key="locale" :value="locale">
                      {{ languageName(locale) }} ({{ locale }})
                    </SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  v-else
                  type="button"
                  variant="outline"
                  size="sm"
                  class="w-full"
                  @click="emit('manage-languages')"
                >
                  <Plus class="mr-2 size-4" aria-hidden="true" />
                  {{ t('helpCenter.addTranslation') }}
                </Button>
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
                          <SelectItem value="published">{{
                            t('globals.terms.published')
                          }}</SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('globals.terms.collection') }}
                </h3>

                <p v-if="localeCollections.length === 0" class="text-sm text-muted-foreground">
                  {{ t('helpCenter.noCollectionsInLanguage') }}
                </p>
                <Button
                  v-if="localeCollections.length === 0 && translationSource"
                  type="button"
                  variant="outline"
                  size="sm"
                  @click="
                    emit('create-translation-collection', {
                      article: translationSource,
                      locale: form.values.locale
                    })
                  "
                >
                  <Plus class="mr-2 size-4" aria-hidden="true" />
                  {{ t('helpCenter.newCollection') }}
                </Button>

                <FormField v-slot="{ componentField }" name="collection_id">
                  <FormItem>
                    <FormControl v-if="localeCollections.length > 0">
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue>{{ collectionLabel }}</SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem
                            v-for="collection in localeCollections"
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

              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground">
                  {{ t('helpCenter.writtenBy') }}
                </h3>
                <FormField v-slot="{ componentField }" name="author_id">
                  <FormItem>
                    <FormControl>
                      <SelectAgentCombobox v-bind="componentField" />
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
                          <SelectItem v-for="loc in selectableLocales" :key="loc" :value="loc">
                            {{ loc }}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
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
                      <Textarea
                        :rows="3"
                        :placeholder="t('helpCenter.excerpt')"
                        v-bind="componentField"
                      />
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
                      <Input
                        type="text"
                        :placeholder="metaTitlePlaceholder"
                        v-bind="componentField"
                      />
                    </FormControl>
                    <FormDescription>{{ t('helpCenter.metaTitleHint') }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField v-slot="{ componentField }" name="meta_description">
                  <FormItem>
                    <FormLabel>{{ t('helpCenter.metaDescription') }}</FormLabel>
                    <FormControl>
                      <Textarea
                        :rows="2"
                        :placeholder="metaDescriptionPlaceholder"
                        v-bind="componentField"
                      />
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
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <div v-if="loadedArticle" class="space-y-3 text-sm border-t pt-4">
                <div v-if="loadedArticle?.created_by_name" class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('helpCenter.createdBy') }}</span>
                  <span>{{ loadedArticle.created_by_name }}</span>
                </div>
                <div
                  v-if="loadedArticle?.helpful_count !== undefined"
                  class="flex justify-between py-1"
                >
                  <span class="text-muted-foreground">{{ t('globals.terms.feedback') }}</span>
                  <span
                    >👍 {{ loadedArticle.helpful_count }} · 👎
                    {{ loadedArticle.not_helpful_count }}</span
                  >
                </div>
                <div class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.createdAt') }}</span>
                  <span>{{ formatDate(loadedArticle.created_at) }}</span>
                </div>
                <div class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.updatedAt') }}</span>
                  <span>{{ formatDate(loadedArticle.updated_at) }}</span>
                </div>
                <div
                  v-if="loadedArticle.view_count !== undefined"
                  class="flex justify-between py-1"
                >
                  <span class="text-muted-foreground">{{ t('globals.terms.view', 2) }}</span>
                  <span>{{ loadedArticle.view_count.toLocaleString() }}</span>
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
import SwitchField from '@shared-ui/components/SwitchField.vue'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import SelectAgentCombobox from '@/components/combobox/SelectAgentCombobox.vue'
import { Sheet, SheetContent, SheetTitle, SheetDescription } from '@shared-ui/components/ui/sheet'
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
import Editor from '@main/components/editor/ArticleEditor.vue'
import { highlightCodeBlocks } from '@main/components/editor/highlightCodeBlocks'
import { Spinner } from '@shared-ui/components/ui/spinner'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useUserStore } from '@/stores/user'
import { format, isValid } from 'date-fns'
import { Check, ChevronRight, Plus } from 'lucide-vue-next'

const { t, locale: uiLocale } = useI18n()

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
  translationSource: {
    type: Object,
    default: null
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

const emit = defineEmits([
  'update:open',
  'cancel',
  'create-translation',
  'create-translation-collection',
  'open-translation',
  'manage-languages'
])
const emitter = useEmitter()
const userStore = useUserStore()

const isLoadingArticle = ref(false)
const availableCollections = ref([])
const editorText = ref('')
const toolbarSlot = ref(null)
const titleInput = ref(null)
const editorRef = ref(null)
// The tree omits article bodies, so the full row is loaded when the sheet opens.
const loadedArticle = ref(null)

const submitLabel = computed(() =>
  props.article ? t('globals.messages.update') : t('globals.messages.create')
)

const linkedLocales = computed(
  () =>
    new Set(
      (loadedArticle.value?.translations || props.translationSource?.translations || []).map(
        ({ locale }) => locale
      )
    )
)
const selectableLocales = computed(() => {
  if (props.translationSource) {
    return props.helpCenterLocales.filter((locale) => !linkedLocales.value.has(locale))
  }
  if (props.article) {
    return props.helpCenterLocales.filter(
      (locale) => locale === loadedArticle.value?.locale || !linkedLocales.value.has(locale)
    )
  }
  return props.helpCenterLocales
})
const translationByLocale = computed(
  () =>
    new Map(
      (loadedArticle.value?.translations || []).map((translation) => [
        translation.locale,
        translation
      ])
    )
)
const translatedLocales = computed(() =>
  props.helpCenterLocales.filter((locale) => translationByLocale.value.has(locale))
)
const missingLocales = computed(() =>
  props.helpCenterLocales.filter((locale) => !translationByLocale.value.has(locale))
)
const languageDisplayNames = computed(
  () => new Intl.DisplayNames([uiLocale.value], { type: 'language' })
)

const languageName = (locale) => languageDisplayNames.value.of(locale) || locale
const formatDate = (value) => {
  const date = new Date(value)
  return isValid(date) ? format(date, 'PPpp') : '-'
}

const toFormValues = () => {
  const article = loadedArticle.value || props.article
  return {
    title: article?.title || '',
    content: article?.content || '',
    status: article?.status || 'draft',
    collection_id: String(article?.collection_id || props.collectionId || ''),
    sort_order: article?.sort_order || 0,
    ai_enabled: article?.ai_enabled || false,
    author_id: String(article?.author_id || (props.article ? '' : userStore.userID) || ''),
    locale:
      article?.locale ||
      props.translationSource?.locale ||
      props.defaultLocale ||
      selectableLocales.value[0] ||
      'en',
    excerpt: article?.excerpt || '',
    meta_title: article?.meta_title || '',
    meta_description: article?.meta_description || '',
    meta_image_url: article?.meta_image_url || ''
  }
}

const form = useForm({
  validationSchema: toTypedSchema(createArticleFormSchema(t)),
  initialValues: toFormValues()
})

// An article and its collection must share a language, else the article drops out of that
// language's tree.
const localeCollections = computed(() =>
  availableCollections.value.filter((collection) => collection.locale === form.values.locale)
)

// The select only learns an option's text when that option mounts, and the collection list
// arrives after the value is set, so the label is resolved here instead.
const collectionLabel = computed(
  () =>
    localeCollections.value.find(
      (collection) => String(collection.id) === String(form.values.collection_id)
    )?.name || ''
)

// A collection from another language is not a valid home for this article, so the choice is
// cleared rather than silently swapped for one the author never picked.
watch(localeCollections, (collections) => {
  const current = Number(form.values.collection_id)
  if (current && !collections.some((collection) => collection.id === current)) {
    form.setFieldValue('collection_id', '', false)
  }
})

// Placeholders preview what the public page falls back to when these fields are left blank.
const metaTitlePlaceholder = computed(() => {
  const title = (form.values.title || '').trim()
  if (!title) return ''
  return props.helpCenterName ? `${title} - ${props.helpCenterName}` : title
})

const metaDescriptionPlaceholder = computed(() => (form.values.excerpt || '').trim())

// loadSeq drops stale fetches so a slow response for a previously opened article
// can't fill the form after another article was opened.
let loadSeq = 0
watch(
  () => [props.article, props.collectionId, props.translationSource, props.isOpen],
  async () => {
    if (!props.isOpen) return
    const seq = ++loadSeq
    loadedArticle.value = null
    isLoadingArticle.value = Boolean(props.article)
    const [, article] = await Promise.all([fetchAvailableCollections(), fetchArticle()])
    if (seq !== loadSeq) return
    loadedArticle.value = article
    isLoadingArticle.value = false
    form.resetForm({ values: toFormValues() })
    await nextTick()
    if (form.values.content) editorRef.value?.focus('end')
    else titleInput.value?.$el?.focus()
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

const fetchArticle = async () => {
  if (!props.article) return null
  try {
    const { data } = await api.getArticle(props.article.collection_id, props.article.id)
    return data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
    return null
  }
}

const selectTranslationLocale = (locale) => {
  if (form.meta.value.dirty) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t(
        translationByLocale.value.has(locale)
          ? 'helpCenter.saveBeforeSwitchingTranslation'
          : 'helpCenter.saveBeforeTranslation'
      )
    })
    return
  }
  const translation = translationByLocale.value.get(locale)
  if (translation) {
    emit('open-translation', translation)
    return
  }
  emit('create-translation', { article: loadedArticle.value, locale })
}

const onSubmit = form.handleSubmit(async (values) => {
  props.submitForm({
    ...values,
    content: highlightCodeBlocks(values.content),
    author_id: values.author_id ? Number(values.author_id) : null,
    translation_of_id: props.translationSource?.id || undefined
  })
})
</script>
