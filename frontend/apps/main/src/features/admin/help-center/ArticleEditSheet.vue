<template>
  <Sheet :open="isOpen" @update:open="$emit('update:open', $event)">
    <SheetContent class="!max-w-[80vw] sm:!max-w-[80vw] h-full p-0 flex flex-col">
      <div class="flex-1 flex flex-col min-h-0">
        <!-- Header -->
        <div class="flex items-center justify-between p-6 border-b bg-card/50">
          <div>
            <h2 class="text-lg font-semibold">
              {{ article ? 'Edit Article' : 'Create Article' }}
            </h2>
            <p class="text-sm text-muted-foreground mt-1">
              {{ article ? `Last updated ${formatDatetime(new Date(article.updated_at))}` : 'Create a new help article' }}
            </p>
          </div>
        </div>

        <!-- Content -->
        <div class="flex-1 flex min-h-0">
          <!-- Main Content Area (75%) -->
          <div class="flex-1 flex flex-col p-6 space-y-6 overflow-y-auto">
            <Spinner v-if="formLoading" />
            
            <form v-else @submit="onSubmit" class="space-y-6 flex-1 flex flex-col">
              <!-- Title -->
              <FormField v-slot="{ componentField }" name="title">
                <FormItem>
                  <FormControl>
                    <Input 
                      type="text" 
                      placeholder="Enter article title..." 
                      v-bind="componentField" 
                      class="text-xl font-semibold border-0 px-0 py-3 shadow-none focus-visible:ring-0 placeholder:text-muted-foreground/60"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Content Editor -->
              <FormField v-slot="{ componentField }" name="content">
                <FormItem class="flex-1 flex flex-col">
                  <FormControl class="flex-1">
                    <div class="flex-1 flex flex-col">
                      <Editor
                        v-model:htmlContent="componentField.modelValue"
                        @update:htmlContent="(value) => componentField.onChange(value)"
                        :placeholder="t('editor.newLine')"
                        editorType="article"
                        class="min-h-[400px] border-0 px-0 shadow-none focus-visible:ring-0"
                      />
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Submit Button (Hidden - controlled by sidebar) -->
              <button type="submit" class="hidden" ref="submitButton"></button>
            </form>
          </div>

          <!-- Sidebar (25%) -->
          <div class="w-80 border-l bg-muted/20 p-6 overflow-y-auto">
            <div class="space-y-6">
              <!-- Publish Actions -->
              <div class="space-y-4">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Actions
                </h3>
                
                <div class="flex gap-2">
                  <Button 
                    type="button" 
                    variant="outline" 
                    size="sm" 
                    @click="$emit('cancel')"
                    class="flex-1"
                  >
                    Cancel
                  </Button>
                  <Button 
                    type="button" 
                    size="sm" 
                    @click="handleSubmit"
                    :disabled="isLoading"
                    class="flex-1"
                  >
                    <Loader2Icon v-if="isLoading" class="h-4 w-4 mr-2 animate-spin" />
                    {{ submitLabel }}
                  </Button>
                </div>
              </div>

              <!-- Status -->
              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Status
                </h3>
                
                <FormField v-slot="{ componentField }" name="status">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="draft">Draft</SelectItem>
                          <SelectItem value="published">Published</SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormDescription class="text-xs">
                      Only published articles are visible to users
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Collection -->
              <div v-if="availableCollections.length > 0" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Collection
                </h3>
                
                <FormField v-slot="{ componentField }" name="collection_id">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue placeholder="Select collection" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem
                            v-for="collection in availableCollections"
                            :key="collection.id"
                            :value="collection.id"
                          >
                            {{ collection.name }}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormDescription class="text-xs">
                      Move this article to a different collection
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- AI Settings -->
              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  AI Settings
                </h3>
                
                <FormField v-slot="{ componentField }" name="ai_enabled">
                  <FormItem class="flex flex-row items-start space-x-3 space-y-0 border rounded-lg p-3">
                    <FormControl>
                      <Checkbox
                        :checked="componentField.modelValue"
                        @update:checked="componentField.onChange"
                      />
                    </FormControl>
                    <div class="space-y-1 leading-none flex-1">
                      <FormLabel class="text-sm font-medium">
                        Allow AI assistants to use this article
                      </FormLabel>
                      <FormDescription class="text-xs">
                        Article must be published for this to take effect
                      </FormDescription>
                    </div>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Metadata -->
              <div v-if="article" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Metadata
                </h3>
                
                <div class="space-y-3 text-sm">
                  <div class="flex justify-between py-2 border-b border-border/50">
                    <span class="text-muted-foreground">Created</span>
                    <span>{{ formatDatetime(new Date(article.created_at)) }}</span>
                  </div>
                  <div class="flex justify-between py-2 border-b border-border/50">
                    <span class="text-muted-foreground">Updated</span>
                    <span>{{ formatDatetime(new Date(article.updated_at)) }}</span>
                  </div>
                  <div v-if="article.view_count !== undefined" class="flex justify-between py-2 border-b border-border/50">
                    <span class="text-muted-foreground">Views</span>
                    <span>{{ article.view_count.toLocaleString() }}</span>
                  </div>
                  <div class="flex justify-between py-2">
                    <span class="text-muted-foreground">ID</span>
                    <span class="font-mono text-xs">#{{ article.id }}</span>
                  </div>
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
import { ref, watch, onMounted, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import {
  Sheet,
  SheetContent,
} from '@shared-ui/components/ui/sheet'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { Loader2 as Loader2Icon } from 'lucide-vue-next'
import { createArticleFormSchema } from './articleFormSchema.js'
import { useI18n } from 'vue-i18n'
import { getTextFromHTML } from '../../../utils/strings.js'
import Editor from '@main/components/editor/TextEditor.vue'
import api from '../../../api'
import { handleHTTPError } from '../../../utils/http'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { formatDatetime } from '@shared-ui/utils/datetime.js'

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
  },
  locale: {
    type: String,
    default: 'en'
  }
})

defineEmits(['update:open', 'cancel'])
const emitter = useEmitter()

const formLoading = ref(false)
const availableCollections = ref([])
const submitButton = ref(null)

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.article ? t('globals.messages.update') : t('globals.messages.create'))
  )
})

const form = useForm({
  validationSchema: toTypedSchema(createArticleFormSchema(t)),
  initialValues: {
    title: props.article?.title || '',
    content: props.article?.content || '',
    status: props.article?.status || 'draft',
    collection_id: props.article?.collection_id || props.collectionId || null,
    sort_order: props.article?.sort_order || 0,
    ai_enabled: props.article?.ai_enabled || false
  }
})

onMounted(async () => {
  await fetchAvailableCollections()
})

watch(
  () => [props.article, props.collectionId, props.locale],
  async (newValues) => {
    const [newArticle, newCollectionId] = newValues
    
    // Re-fetch available collections when article, collectionId, or locale changes
    await fetchAvailableCollections()
    
    if (newArticle && Object.keys(newArticle).length > 0) {
      form.setValues({
        title: newArticle.title || '',
        content: newArticle.content || '',
        status: newArticle.status || 'draft',
        collection_id: newArticle.collection_id || newCollectionId || null,
        sort_order: newArticle.sort_order || 0,
        ai_enabled: newArticle.ai_enabled || false
      })
    }
  },
  { immediate: true }
)

const fetchAvailableCollections = async () => {
  try {
    let helpCenterId = null
    if (props.article?.collection_id) {
      // Editing existing article - get its collection first to find help center
      const { data: collection } = await api.getCollection(props.article.collection_id)
      helpCenterId = collection.data.help_center_id
    } else if (props.collectionId) {
      // Creating new article - get help center from provided collection
      const { data: collection } = await api.getCollection(props.collectionId)
      helpCenterId = collection.data.help_center_id
    }
    
    if (helpCenterId) {
      // Filter collections by current locale
      const { data: collections } = await api.getCollections(helpCenterId, { locale: props.locale })
      // Allow selecting all published collections for the current locale
      availableCollections.value = collections.data.filter((c) => c.is_published)
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  const textContent = getTextFromHTML(values.content)
  if (textContent.length === 0) {
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