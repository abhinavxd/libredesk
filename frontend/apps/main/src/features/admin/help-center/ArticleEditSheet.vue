<template>
  <Sheet :open="isOpen" @update:open="$emit('update:open', $event)">
    <SheetContent class="!max-w-[80vw] sm:!max-w-[80vw] h-full p-0 flex flex-col">
      <div class="flex-1 flex flex-col min-h-0">
        <div class="flex items-center justify-between p-6 border-b bg-card/50">
          <div>
            <SheetTitle>
              {{ article ? t('helpCenter.editArticle') : t('helpCenter.newArticle') }}
            </SheetTitle>
            <SheetDescription v-if="article" class="mt-1">
              {{ t('globals.terms.lastUpdated') }}:
              {{ format(new Date(article.updated_at), 'PPpp') }}
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
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue>{{ authorLabel }}</SelectValue>
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup v-if="agentOptions.length">
                            <SelectLabel>{{ t('globals.terms.agent', 2) }}</SelectLabel>
                            <SelectItem v-for="o in agentOptions" :key="o.value" :value="o.value">
                              {{ o.label }}
                            </SelectItem>
                          </SelectGroup>
                          <SelectGroup v-if="assistantOptions.length">
                            <SelectLabel>{{ t('admin.ai.assistants') }}</SelectLabel>
                            <SelectItem
                              v-for="o in assistantOptions"
                              :key="o.value"
                              :value="o.value"
                            >
                              {{ o.label }}
                            </SelectItem>
                          </SelectGroup>
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

              <div v-if="article" class="space-y-3 text-sm border-t pt-4">
                <div v-if="loadedArticle?.created_by_name" class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('helpCenter.createdBy') }}</span>
                  <span>{{ loadedArticle.created_by_name }}</span>
                </div>
                <div v-if="loadedArticle?.helpful_count !== undefined" class="flex justify-between py-1">
                  <span class="text-muted-foreground">{{ t('globals.terms.feedback') }}</span>
                  <span>👍 {{ loadedArticle.helpful_count }} · 👎 {{ loadedArticle.not_helpful_count }}</span>
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
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
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
import { useUsersStore } from '@/stores/users'
import { useUserStore } from '@/stores/user'
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
const usersStore = useUsersStore()
const userStore = useUserStore()

const agentOptions = computed(() => usersStore.options.filter((o) => o.type === 'agent'))
const assistantOptions = computed(() => usersStore.options.filter((o) => o.type === 'ai_assistant'))

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
    locale: article?.locale || props.defaultLocale || props.helpCenterLocales?.[0] || 'en',
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

const authorLabel = computed(
  () => usersStore.options.find((o) => o.value === String(form.values.author_id))?.label || ''
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
  () => [props.article, props.collectionId, props.isOpen],
  async () => {
    if (!props.isOpen) return
    const seq = ++loadSeq
    loadedArticle.value = null
    isLoadingArticle.value = Boolean(props.article)
    const [, , article] = await Promise.all([
      usersStore.fetchUsers(),
      fetchAvailableCollections(),
      fetchArticle()
    ])
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

const onSubmit = form.handleSubmit(async (values) => {
  props.submitForm({
    ...values,
    content: highlightCodeBlocks(values.content),
    author_id: values.author_id ? Number(values.author_id) : null
  })
})
</script>
