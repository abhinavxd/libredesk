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
            <div v-if="loadedArticle" class="mt-1 flex items-center gap-3">
              <SheetDescription>
                {{ t('globals.terms.lastUpdated') }}:
                {{ formatDate(loadedArticle.updated_at) }}
              </SheetDescription>
              <Button
                v-if="articleUrl"
                as="a"
                variant="link"
                :href="articleUrl"
                target="_blank"
                rel="noopener"
                class="h-auto gap-1 p-0 font-normal"
              >
                {{ t('helpCenter.viewArticle') }}
                <ExternalLink class="size-3.5" aria-hidden="true" />
              </Button>
            </div>
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
                        :accent-color="helpCenterColor"
                        :accent-color-dark="helpCenterColorDark"
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
                  {{ t('globals.terms.translation', 2) }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ t('helpCenter.translationsHint') }}
                </p>

                <div class="space-y-2">
                  <div v-for="locale in translatedLocales" :key="locale" class="flex gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      class="h-auto min-w-0 flex-1 justify-start gap-3 px-3 py-2.5 text-left font-normal disabled:opacity-100"
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
                        v-else
                        class="size-4 shrink-0 text-muted-foreground"
                        aria-hidden="true"
                      />
                    </Button>
                    <Tooltip v-if="translatedLocales.length > 1">
                      <TooltipTrigger as-child>
                        <Button
                          type="button"
                          variant="outline"
                          size="icon"
                          class="h-auto shrink-0 text-muted-foreground hover:text-destructive"
                          :aria-label="t('helpCenter.unlinkTranslation')"
                          @click="askUnlink(locale)"
                        >
                          <Unlink aria-hidden="true" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        {{ t('helpCenter.unlinkTranslation') }}
                      </TooltipContent>
                    </Tooltip>
                  </div>
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
                  v-if="missingLocales.length"
                  type="button"
                  variant="ghost"
                  size="sm"
                  class="w-full"
                  @click="openLinkDialog"
                >
                  <Link2 class="mr-2 size-4" aria-hidden="true" />
                  {{ t('helpCenter.linkExistingArticle') }}
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
                  @click="emit('create-translation-collection', { locale: form.values.locale })"
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

  <Dialog :open="showLinkDialog" @update:open="showLinkDialog = $event">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t('helpCenter.linkExistingArticle') }}</DialogTitle>
        <DialogDescription>{{ t('helpCenter.linkExistingArticleHint') }}</DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <Spinner v-if="isLoadingLinkable" />
        <p v-else-if="!linkableArticleItems.length" class="text-sm text-muted-foreground">
          {{ t('helpCenter.noLinkableArticles') }}
        </p>
        <SelectComboBox
          v-else
          v-model="linkArticleID"
          :items="linkableArticleItems"
          :placeholder="t('placeholders.selectArticle')"
        />
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" @click="showLinkDialog = false">
          {{ t('globals.messages.cancel') }}
        </Button>
        <Button type="button" :disabled="!linkArticleID || isLinking" @click="confirmLink">
          {{ t('globals.terms.link') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <AlertDialog :open="showUnlinkDialog" @update:open="showUnlinkDialog = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            t('helpCenter.unlinkTranslationConfirmation', {
              language: languageName(unlinkLocale)
            })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="confirmUnlink">{{
          t('globals.messages.unlink')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
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
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useUserStore } from '@/stores/user'
import { useAppSettingsStore } from '@/stores/appSettings'
import { format, isValid } from 'date-fns'
import { Check, ChevronRight, ExternalLink, Link2, Plus, Unlink } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'

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
  helpCenterSlug: {
    type: String,
    default: ''
  },
  helpCenterColor: {
    type: String,
    default: ''
  },
  helpCenterColorDark: {
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
  createdCollection: {
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
  'open-translation'
])
const emitter = useEmitter()
const userStore = useUserStore()
const appSettingsStore = useAppSettingsStore()

const isLoadingArticle = ref(false)
const showUnlinkDialog = ref(false)
const unlinkLocale = ref('')
const showLinkDialog = ref(false)
const isLinking = ref(false)
const isLoadingLinkable = ref(false)
const linkArticleID = ref('')
const linkableArticles = ref([])
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

const articleUrl = computed(() => {
  if (!props.helpCenterSlug || loadedArticle.value?.status !== 'published') return ''
  const root = (appSettingsStore.settings?.['app.root_url'] || window.location.origin).replace(
    /\/$/,
    ''
  )
  return `${root}/hc/${props.helpCenterSlug}/${loadedArticle.value.locale}/articles/${loadedArticle.value.slug}`
})
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

watch(
  () => props.createdCollection,
  async (collection) => {
    if (!collection || !props.isOpen) return
    await fetchAvailableCollections()
    form.setFieldValue('collection_id', String(collection.id), false)
  }
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

const linkableArticleItems = computed(() =>
  linkableArticles.value
    .filter((article) => !translationByLocale.value.has(article.locale))
    .map((article) => ({
      label: `${article.title} - ${languageName(article.locale)}`,
      value: String(article.id)
    }))
)

const openLinkDialog = async () => {
  linkArticleID.value = ''
  linkableArticles.value = []
  showLinkDialog.value = true
  isLoadingLinkable.value = true
  try {
    const { data } = await api.getLinkableArticles(props.helpCenterId, loadedArticle.value.locale)
    linkableArticles.value = data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoadingLinkable.value = false
  }
}

const confirmLink = async () => {
  isLinking.value = true
  try {
    await api.linkArticleTranslation(Number(linkArticleID.value), {
      translation_of_id: loadedArticle.value.id
    })
    loadedArticle.value = await fetchArticle()
    showLinkDialog.value = false
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('helpCenter.translationLinked')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLinking.value = false
  }
}

const askUnlink = (locale) => {
  unlinkLocale.value = locale
  showUnlinkDialog.value = true
}

const confirmUnlink = async () => {
  showUnlinkDialog.value = false
  try {
    await api.unlinkArticleTranslation(translationByLocale.value.get(unlinkLocale.value).id)
    loadedArticle.value = await fetchArticle()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('helpCenter.translationUnlinked')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
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
