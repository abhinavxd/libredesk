<template>
  <Sheet :open="isOpen" @update:open="$emit('update:open', $event)">
    <SheetContent class="!max-w-[60vw] sm:!max-w-[60vw] h-full p-0 flex flex-col">
      <div class="flex-1 flex flex-col min-h-0">
        <!-- Header -->
        <div class="flex items-center justify-between p-6 border-b bg-card/50">
          <div>
            <h2 class="text-lg font-semibold">
              {{ collection ? 'Edit Collection' : 'Create Collection' }}
            </h2>
            <p class="text-sm text-muted-foreground mt-1">
              {{ collection ? `Last updated ${formatDatetime(new Date(collection.updated_at))}` : 'Create a new help collection' }}
            </p>
          </div>
        </div>

        <!-- Content -->
        <div class="flex-1 flex min-h-0">
          <!-- Main Content Area (70%) -->
          <div class="flex-1 flex flex-col p-6 space-y-6 overflow-y-auto">
            <Spinner v-if="formLoading" />
            
            <form v-else @submit="onSubmit" class="space-y-6 flex-1 flex flex-col">
              <!-- Name -->
              <FormField v-slot="{ componentField }" name="name">
                <FormItem>
                  <FormControl>
                    <Input 
                      type="text" 
                      placeholder="Enter collection name..." 
                      v-bind="componentField" 
                      class="text-xl font-semibold border-0 px-0 py-3 shadow-none focus-visible:ring-0 placeholder:text-muted-foreground/60"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Description -->
              <FormField v-slot="{ componentField }" name="description">
                <FormItem class="flex-1">
                  <FormControl>
                    <Textarea
                      placeholder="Describe what this collection contains..."
                      rows="6"
                      v-bind="componentField"
                      class="border-0 px-0 py-2 shadow-none focus-visible:ring-0 resize-none placeholder:text-muted-foreground/60"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Submit Button (Hidden - controlled by sidebar) -->
              <button type="submit" class="hidden" ref="submitButton"></button>
            </form>
          </div>

          <!-- Sidebar (30%) -->
          <div class="w-72 border-l bg-muted/20 p-6 overflow-y-auto">
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

              <!-- Visibility -->
              <div class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Visibility
                </h3>
                
                <FormField v-slot="{ componentField }" name="is_published">
                  <FormItem class="flex flex-row items-start space-x-3 space-y-0 border rounded-lg p-3">
                    <FormControl>
                      <Checkbox
                        :checked="componentField.modelValue"
                        @update:checked="componentField.onChange"
                      />
                    </FormControl>
                    <div class="space-y-1 leading-none flex-1">
                      <FormLabel class="text-sm font-medium">
                        Published
                      </FormLabel>
                      <FormDescription class="text-xs">
                        Published collections are visible to users
                      </FormDescription>
                    </div>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Parent Collection -->
              <div v-if="availableParents.length > 0" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Parent Collection
                </h3>
                
                <FormField v-slot="{ componentField }" name="parent_id">
                  <FormItem>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue placeholder="Select parent (optional)" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem :value="0">No parent (root level)</SelectItem>
                          <SelectItem v-for="parent in availableParents" :key="parent.id" :value="parent.id">
                            {{ parent.name }}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormDescription class="text-xs">
                      Collections can be nested up to 3 levels deep
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Articles Count -->
              <div v-if="collection && collection.articles" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Articles
                </h3>
                
                <div class="border rounded-lg p-3">
                  <div class="flex items-center justify-between">
                    <span class="text-sm font-medium">Total Articles</span>
                    <Badge variant="outline">{{ collection.articles.length }}</Badge>
                  </div>
                  <p class="text-xs text-muted-foreground mt-2">
                    {{ collection.articles.filter(a => a.status === 'published').length }} published, 
                    {{ collection.articles.filter(a => a.status === 'draft').length }} draft
                  </p>
                </div>
              </div>

              <!-- Metadata -->
              <div v-if="collection" class="space-y-3">
                <h3 class="font-medium text-sm text-muted-foreground uppercase tracking-wider">
                  Metadata
                </h3>
                
                <div class="space-y-3 text-sm">
                  <div class="flex justify-between py-2 border-b border-border/50">
                    <span class="text-muted-foreground">Created</span>
                    <span>{{ formatDatetime(new Date(collection.created_at)) }}</span>
                  </div>
                  <div class="flex justify-between py-2 border-b border-border/50">
                    <span class="text-muted-foreground">Updated</span>
                    <span>{{ formatDatetime(new Date(collection.updated_at)) }}</span>
                  </div>
                  <div v-if="collection.view_count !== undefined" class="flex justify-between py-2 border-b border-border/50">
                    <span class="text-muted-foreground">Views</span>
                    <span>{{ collection.view_count.toLocaleString() }}</span>
                  </div>
                  <div class="flex justify-between py-2">
                    <span class="text-muted-foreground">ID</span>
                    <span class="font-mono text-xs">#{{ collection.id }}</span>
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
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Badge } from '@shared-ui/components/ui/badge'
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
import { createCollectionFormSchema } from './collectionFormSchema.js'
import { useI18n } from 'vue-i18n'
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
const availableParents = ref([])
const submitButton = ref(null)

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.collection ? t('globals.messages.update') : t('globals.messages.create'))
  )
})

const form = useForm({
  validationSchema: toTypedSchema(createCollectionFormSchema(t)),
  initialValues: {
    name: props.collection?.name || '',
    description: props.collection?.description || '',
    parent_id: props.collection?.parent_id || props.parentId || null,
    is_published: props.collection?.is_published ?? true,
    sort_order: props.collection?.sort_order || 0
  }
})

onMounted(async () => {
  await fetchAvailableParents()
})

watch(
  () => props.collection,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      form.setValues({
        name: newValues.name || '',
        description: newValues.description || '',
        parent_id: newValues.parent_id || null,
        is_published: newValues.is_published ?? true,
        sort_order: newValues.sort_order || 0
      })
    }
  },
  { immediate: true }
)

watch(
  () => props.locale,
  async () => {
    await fetchAvailableParents()
  }
)

const fetchAvailableParents = async () => {
  try {
    // Filter collections by current locale
    const { data } = await api.getCollections(props.helpCenterId, { locale: props.locale })
    availableParents.value = data.data.filter((collection) => {
      // Exclude self and children from parent options
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
  props.submitForm(values)
})

const handleSubmit = () => {
  if (submitButton.value) {
    submitButton.value.click()
  }
}
</script>