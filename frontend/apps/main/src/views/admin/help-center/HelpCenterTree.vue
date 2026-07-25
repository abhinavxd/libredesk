<template>
  <Spinner v-if="loading" />
  <div v-else class="h-full flex flex-col">
    <div class="flex items-center justify-between mb-5">
      <div class="flex items-center gap-3">
        <CustomBreadcrumb :links="breadcrumbLinks" />
        <Badge v-if="helpCenter && !helpCenter.is_active" variant="secondary">
          {{ t('helpCenter.paused') }}
        </Badge>
      </div>

      <div class="flex items-center gap-2">
        <DropdownMenu :modal="false">
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="sm">
              <span class="sr-only">{{ t('globals.terms.openMenu') }}</span>
              <MoreVertical class="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem @click="editHelpCenter">
              <Pencil class="mr-2 h-4 w-4" />
              {{ t('globals.messages.edit') }}
            </DropdownMenuItem>
            <DropdownMenuItem @click="toggleActive">
              <component :is="helpCenter?.is_active ? PowerOff : Power" class="mr-2 h-4 w-4" />
              {{ helpCenter?.is_active ? t('helpCenter.pause') : t('helpCenter.resume') }}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem @click="deleteHelpCenter" class="text-destructive focus:text-destructive">
              <Trash class="mr-2 h-4 w-4" />
              {{ t('globals.messages.delete') }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <Select
          v-if="allowedLocales.length > 1"
          :model-value="props.locale"
          @update:model-value="changeLocale"
        >
          <SelectTrigger class="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="loc in allowedLocales" :key="loc" :value="loc">{{ loc }}</SelectItem>
          </SelectContent>
        </Select>

        <Button variant="outline" @click="openInsights">
          <BarChart3 class="h-4 w-4" />
          {{ t('helpCenter.insights') }}
        </Button>

        <Button @click="openCreateCollectionModal">
          <Plus class="h-4 w-4" />
          {{ t('helpCenter.newCollection') }}
        </Button>
      </div>
    </div>

    <div class="flex-1 min-h-0">
      <div class="border rounded-lg shadow-sm p-6 h-full overflow-y-auto">
        <div v-if="treeData.length === 0 && !loading" class="text-center py-16">
          <div
            class="mx-auto w-24 h-24 bg-muted rounded-full flex items-center justify-center mb-6"
          >
            <Folder class="h-12 w-12 text-muted-foreground" />
          </div>
          <p class="text-muted-foreground mb-6">{{ t('helpCenter.noCollections') }}</p>
          <Button @click="openCreateCollectionModal">
            <Plus class="h-4 w-4 mr-2" />
            {{ t('helpCenter.newCollection') }}
          </Button>
        </div>

        <TreeView
          v-else
          :data="treeData"
          :selected-item="selectedItem"
          @select="selectItem"
          @create-collection="openCreateCollectionModal"
          @create-article="openCreateArticleModal"
          @edit="openEditSheet"
          @delete="deleteItem"
          @toggle-status="toggleStatus"
        />
      </div>
    </div>
  </div>

  <ArticleEditSheet
    :is-open="showArticleEditSheet"
    @update:open="showArticleEditSheet = $event"
    :article="editingArticle"
    :collection-id="editingArticle?.collection_id || createArticleCollectionId"
    :help-center-id="parseInt(id)"
    :help-center-name="helpCenter?.name || ''"
    :help-center-locales="helpCenter?.allowed_locales || ['en']"
    :default-locale="props.locale"
    :submit-form="handleArticleSave"
    :is-loading="isSubmittingArticle"
    @cancel="closeEditSheet"
  />

  <CollectionEditSheet
    :is-open="showCollectionEditSheet"
    @update:open="showCollectionEditSheet = $event"
    :collection="editingCollection"
    :help-center-id="parseInt(id)"
    :parent-id="createCollectionParentId"
    :help-center-locales="helpCenter?.allowed_locales || ['en']"
    :default-locale="props.locale"
    :submit-form="handleCollectionSave"
    :is-loading="isSubmittingCollection"
    @cancel="closeEditSheet"
  />

  <Sheet :open="showHelpCenterEditSheet" @update:open="showHelpCenterEditSheet = false">
    <SheetContent class="sm:max-w-lg overflow-y-auto">
      <SheetHeader>
        <SheetTitle>{{ t('globals.messages.edit') }}</SheetTitle>
      </SheetHeader>

      <HelpCenterForm
        :help-center="editingHelpCenter"
        :submit-form="handleHelpCenterSave"
        :is-loading="isSubmittingHelpCenter"
        @cancel="closeHelpCenterEditSheet"
      />
    </SheetContent>
  </Sheet>

  <Sheet :open="showInsights" @update:open="showInsights = $event">
    <SheetContent class="sm:max-w-lg overflow-y-auto">
      <SheetHeader>
        <SheetTitle>{{ t('helpCenter.insights') }}</SheetTitle>
      </SheetHeader>

      <div class="mt-6 space-y-8">
        <div>
          <h3 class="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-3">
            {{ t('helpCenter.topSearches') }}
          </h3>
          <p v-if="!insights.top_searches?.length" class="text-sm text-muted-foreground">
            {{ t('helpCenter.noSearchData') }}
          </p>
          <table v-else class="w-full text-sm">
            <thead>
              <tr class="text-left text-muted-foreground border-b">
                <th class="py-1 font-medium">{{ t('helpCenter.searchTerm') }}</th>
                <th class="py-1 font-medium text-right">{{ t('helpCenter.searchCount') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in insights.top_searches" :key="s.query" class="border-b last:border-0">
                <td class="py-1.5">{{ s.query }}</td>
                <td class="py-1.5 text-right tabular-nums">{{ s.count }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div>
          <h3 class="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-3">
            {{ t('helpCenter.noResultSearches') }}
          </h3>
          <p v-if="!insights.no_result_searches?.length" class="text-sm text-muted-foreground">
            {{ t('helpCenter.noSearchData') }}
          </p>
          <table v-else class="w-full text-sm">
            <thead>
              <tr class="text-left text-muted-foreground border-b">
                <th class="py-1 font-medium">{{ t('helpCenter.searchTerm') }}</th>
                <th class="py-1 font-medium text-right">{{ t('helpCenter.searchCount') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="s in insights.no_result_searches"
                :key="s.query"
                class="border-b last:border-0"
              >
                <td class="py-1.5">{{ s.query }}</td>
                <td class="py-1.5 text-right tabular-nums">{{ s.count }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </SheetContent>
  </Sheet>

  <AlertDialog :open="showDeleteDialog" @update:open="showDeleteDialog = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ deleteConfirmationText }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="confirmDelete">{{
          t('globals.messages.delete')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
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
import { Folder, Plus, MoreVertical, Pencil, Trash, BarChart3, Power, PowerOff } from 'lucide-vue-next'
import { Badge } from '@shared-ui/components/ui/badge'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@shared-ui/components/ui/sheet'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import TreeView from '@/features/admin/help-center/TreeView.vue'
import ArticleEditSheet from '@/features/admin/help-center/ArticleEditSheet.vue'
import CollectionEditSheet from '@/features/admin/help-center/CollectionEditSheet.vue'
import HelpCenterForm from '@/features/admin/help-center/HelpCenterForm.vue'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  id: {
    type: String,
    required: true
  },
  locale: {
    type: String,
    default: ''
  }
})

const router = useRouter()
const emitter = useEmitter()
const { t } = useI18n()
const loading = ref(true)
const isSubmittingCollection = ref(false)
const isSubmittingArticle = ref(false)
const isSubmittingHelpCenter = ref(false)
const helpCenter = ref(null)
const treeData = ref([])
const selectedItem = ref(null)

const allowedLocales = computed(() =>
  Array.isArray(helpCenter.value?.allowed_locales) ? helpCenter.value.allowed_locales : []
)

const showDeleteDialog = ref(false)
const showInsights = ref(false)
const insights = ref({ top_searches: [], no_result_searches: [] })
const showArticleEditSheet = ref(false)
const showCollectionEditSheet = ref(false)
const showHelpCenterEditSheet = ref(false)
const editingArticle = ref(null)
const editingCollection = ref(null)
const editingHelpCenter = ref(null)
const createCollectionParentId = ref(null)
const createArticleCollectionId = ref(null)
const deletingItem = ref(null)

const breadcrumbLinks = computed(() => [
  { path: 'help-center-list', label: t('globals.terms.helpCenter', 2) },
  { path: '', label: helpCenter.value?.name || '' }
])

const deleteConfirmationText = computed(() => {
  switch (deletingItem.value?.type) {
    case 'collection':
      return t('helpCenter.deleteCollectionConfirmation')
    case 'article':
      return t('helpCenter.deleteArticleConfirmation')
    default:
      return t('helpCenter.deleteConfirmation')
  }
})

onMounted(async () => {
  await fetchHelpCenter()
  const fallback = helpCenter.value?.default_locale || allowedLocales.value[0] || ''
  if (!props.locale && fallback) {
    router.replace({ name: 'help-center-tree', params: { id: props.id, locale: fallback } })
    return
  }
  await fetchTree()
})

watch(
  () => props.locale,
  () => {
    if (helpCenter.value) fetchTree()
  }
)

const changeLocale = (locale) => {
  router.push({ name: 'help-center-tree', params: { id: props.id, locale } })
}

const fetchHelpCenter = async () => {
  try {
    const { data } = await api.getHelpCenter(props.id)
    helpCenter.value = data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const fetchTree = async () => {
  try {
    loading.value = true
    const { data } = await api.getHelpCenterTree(props.id, props.locale)
    treeData.value = data.data.tree || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}

const selectItem = (item) => {
  selectedItem.value = item
  openEditSheet(item)
}

const openEditSheet = (item) => {
  if (item.type === 'article') {
    editingArticle.value = item
    editingCollection.value = null
    showArticleEditSheet.value = true
  } else if (item.type === 'collection') {
    editingCollection.value = item
    editingArticle.value = null
    showCollectionEditSheet.value = true
  }
}

const closeEditSheet = () => {
  showArticleEditSheet.value = false
  showCollectionEditSheet.value = false
  editingArticle.value = null
  editingCollection.value = null
  selectedItem.value = null
  createCollectionParentId.value = null
  createArticleCollectionId.value = null
}

const closeHelpCenterEditSheet = () => {
  showHelpCenterEditSheet.value = false
  editingHelpCenter.value = null
}

const editHelpCenter = () => {
  editingHelpCenter.value = helpCenter.value
  showHelpCenterEditSheet.value = true
}

const deleteHelpCenter = () => {
  deletingItem.value = { ...helpCenter.value, type: 'help_center' }
  showDeleteDialog.value = true
}

const toggleActive = async () => {
  try {
    const { data } = await api.toggleHelpCenter(props.id)
    helpCenter.value = data.data
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const handleHelpCenterSave = async (formData) => {
  isSubmittingHelpCenter.value = true
  try {
    await api.updateHelpCenter(props.id, formData)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    closeHelpCenterEditSheet()
    await fetchHelpCenter()
    await fetchTree()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmittingHelpCenter.value = false
  }
}

const openCreateCollectionModal = (parentId = null) => {
  editingCollection.value = null
  createCollectionParentId.value = typeof parentId === 'number' ? parentId : null
  showCollectionEditSheet.value = true
}

const handleCollectionSave = async (formData) => {
  isSubmittingCollection.value = true
  try {
    if (editingCollection.value) {
      await api.updateCollection(props.id, editingCollection.value.id, formData)
    } else {
      if (createCollectionParentId.value !== null) {
        formData.parent_id = createCollectionParentId.value
      }
      await api.createCollection(props.id, formData)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    closeEditSheet()
    fetchTree()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmittingCollection.value = false
  }
}

const openInsights = async () => {
  showInsights.value = true
  try {
    const { data } = await api.getHelpCenterInsights(props.id)
    insights.value = data.data || { top_searches: [], no_result_searches: [] }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const openCreateArticleModal = (collection) => {
  editingArticle.value = null
  createArticleCollectionId.value = collection.id
  showArticleEditSheet.value = true
}

const handleArticleSave = async (formData) => {
  isSubmittingArticle.value = true
  try {
    if (editingArticle.value) {
      const targetArticle = editingArticle.value
      if (formData.collection_id !== targetArticle.collection_id) {
        await api.updateArticleByID(targetArticle.id, formData)
      } else {
        await api.updateArticle(targetArticle.collection_id, targetArticle.id, formData)
      }
    } else {
      await api.createArticle(createArticleCollectionId.value, formData)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    closeEditSheet()
    fetchTree()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmittingArticle.value = false
  }
}

const deleteItem = (item) => {
  deletingItem.value = item
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  try {
    if (deletingItem.value.type === 'collection') {
      await api.deleteCollection(props.id, deletingItem.value.id)
    } else if (deletingItem.value.type === 'help_center') {
      await api.deleteHelpCenter(deletingItem.value.id)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('globals.messages.deletedSuccessfully')
      })
      router.push({ name: 'help-center-list' })
      return
    } else {
      await api.deleteArticle(deletingItem.value.collection_id, deletingItem.value.id)
    }

    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })

    if (selectedItem.value?.id === deletingItem.value.id) {
      selectedItem.value = null
    }

    showDeleteDialog.value = false
    deletingItem.value = null
    fetchTree()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const toggleStatus = async (item) => {
  try {
    if (item.type === 'collection') {
      await api.toggleCollection(item.id)
    } else {
      const newStatus = item.status === 'published' ? 'draft' : 'published'
      await api.updateArticleStatus(item.id, { status: newStatus })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    fetchTree()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
