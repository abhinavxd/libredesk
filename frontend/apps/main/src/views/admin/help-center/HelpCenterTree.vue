<template>
  <AdminSplitLayout>
    <template #content>
      <Spinner v-if="loading" />
      <div v-else class="h-full flex flex-col">
        <div class="mb-5">
          <div class="flex items-center gap-3">
            <CustomBreadcrumb :links="breadcrumbLinks" />
            <Badge v-if="helpCenter && !helpCenter.is_active" variant="secondary">
              {{ t('globals.terms.paused') }}
            </Badge>
          </div>

          <div class="flex items-center justify-end flex-wrap gap-2 mt-4">
            <Select
              v-if="allowedLocales.length > 1"
              :model-value="props.locale"
              @update:model-value="changeLocale"
            >
              <SelectTrigger class="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="loc in allowedLocales" :key="loc" :value="loc">{{
                  loc
                }}</SelectItem>
              </SelectContent>
            </Select>

            <Button variant="outline" size="icon" @click="toggleExpandAll">
              <component :is="allExpanded ? ChevronsDownUp : ChevronsUpDown" class="h-4 w-4" />
              <span class="sr-only">{{
                allExpanded ? t('globals.messages.collapseAll') : t('globals.messages.expandAll')
              }}</span>
            </Button>

            <Button variant="outline" @click="openInsights">
              <BarChart3 class="h-4 w-4" />
              {{ t('helpCenter.insights') }}
            </Button>

            <Button variant="outline" @click="visitSite">
              <ExternalLink class="h-4 w-4" />
              {{ t('helpCenter.visitSite') }}
            </Button>

            <Button @click="openCreateCollectionModal">
              <Plus class="h-4 w-4" />
              {{ t('helpCenter.newCollection') }}
            </Button>

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
                <DropdownMenuItem
                  @click="deleteHelpCenter"
                  class="text-destructive focus:text-destructive"
                >
                  <Trash class="mr-2 h-4 w-4" />
                  {{ t('globals.messages.delete') }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <div class="flex-1 min-h-0">
          <div class="h-full overflow-y-auto pr-1">
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
              @reorder-collections="reorderCollections"
              @reorder-articles="reorderArticles"
              @move-article="moveArticleToCollection"
            />
          </div>
        </div>
      </div>
    </template>

    <template #help>
      <p>{{ t('admin.helpCenter.treeHelp') }}</p>
    </template>
  </AdminSplitLayout>

  <ArticleEditSheet
    :is-open="showArticleEditSheet"
    @update:open="$event ? (showArticleEditSheet = true) : closeEditSheet()"
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
    @update:open="$event ? (showCollectionEditSheet = true) : closeEditSheet()"
    :collection="editingCollection"
    :help-center-id="parseInt(id)"
    :parent-id="createCollectionParentId"
    :help-center-locales="helpCenter?.allowed_locales || ['en']"
    :default-locale="props.locale"
    :submit-form="handleCollectionSave"
    :is-loading="isSubmittingCollection"
    @cancel="closeEditSheet"
  />

  <Sheet :open="showInsights" @update:open="showInsights = $event">
    <SheetContent class="sm:max-w-lg overflow-y-auto">
      <SheetHeader>
        <SheetTitle>{{ t('helpCenter.insights') }}</SheetTitle>
      </SheetHeader>

      <Spinner v-if="insightsLoading" class="mt-6" />

      <div v-else class="mt-6 space-y-8">
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
        <AlertDialogAction variant="destructive" @click="confirmDelete">{{
          t('globals.messages.delete')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref, onMounted, computed, watch, provide } from 'vue'
import { useRouter } from 'vue-router'
import { useStorage } from '@vueuse/core'
import { useEmitter } from '@/composables/useEmitter.js'
import { useAppSettingsStore } from '@/stores/appSettings'
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
import {
  Folder,
  Plus,
  MoreVertical,
  Pencil,
  Trash,
  BarChart3,
  Power,
  PowerOff,
  ExternalLink,
  ChevronsDownUp,
  ChevronsUpDown
} from 'lucide-vue-next'
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
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import TreeView from '@/features/admin/help-center/TreeView.vue'
import ArticleEditSheet from '@/features/admin/help-center/ArticleEditSheet.vue'
import CollectionEditSheet from '@/features/admin/help-center/CollectionEditSheet.vue'
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
const appSettingsStore = useAppSettingsStore()
const { t } = useI18n()
const loading = ref(true)
const isSubmittingCollection = ref(false)
const isSubmittingArticle = ref(false)
const helpCenter = ref(null)
const treeData = ref([])
const selectedItem = ref(null)

const allowedLocales = computed(() =>
  Array.isArray(helpCenter.value?.allowed_locales) ? helpCenter.value.allowed_locales : []
)

const showDeleteDialog = ref(false)
const showInsights = ref(false)
const insightsLoading = ref(false)
const insights = ref({ top_searches: [], no_result_searches: [] })
const showArticleEditSheet = ref(false)
const showCollectionEditSheet = ref(false)
const editingArticle = ref(null)
const editingCollection = ref(null)
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
  const unsupported = props.locale && !allowedLocales.value.includes(props.locale)
  if ((!props.locale || unsupported) && fallback) {
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

// The tree renders one language at a time; a row saved in another one lands off screen.
const followSavedLocale = (locale) => {
  if (!locale || locale === props.locale) return false
  changeLocale(locale)
  return true
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

// A silent refresh keeps the tree on screen; the spinner would flicker on every reorder.
const fetchTree = async ({ silent = false } = {}) => {
  try {
    loading.value = !silent
    const { data } = await api.getHelpCenterTree(props.id, props.locale)
    helpCenter.value = data.data.help_center || helpCenter.value
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

const visitSite = () => {
  const rootUrl = appSettingsStore.settings?.['app.root_url'] || window.location.origin
  const locale = props.locale || helpCenter.value?.default_locale || ''
  const path = locale ? `/hc/${helpCenter.value?.slug}/${locale}` : `/hc/${helpCenter.value?.slug}`
  window.open(`${rootUrl.replace(/\/$/, '')}${path}`, '_blank', 'noopener')
}

const editHelpCenter = () => {
  router.push({ name: 'help-center-customize', params: { id: props.id } })
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
      await api.createCollection(props.id, formData)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    closeEditSheet()
    if (!followSavedLocale(formData.locale)) fetchTree({ silent: true })
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
  insightsLoading.value = true
  insights.value = { top_searches: [], no_result_searches: [] }
  try {
    const { data } = await api.getHelpCenterInsights(props.id)
    insights.value = data.data || { top_searches: [], no_result_searches: [] }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    insightsLoading.value = false
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
      await api.updateArticle(editingArticle.value.id, formData)
    } else {
      await api.createArticle(formData.collection_id || createArticleCollectionId.value, formData)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    closeEditSheet()
    if (!followSavedLocale(formData.locale)) fetchTree({ silent: true })
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
    fetchTree({ silent: true })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const collapsedIds = useStorage(`helpCenterTreeCollapsed:${props.id}`, [])
const allExpanded = computed(() => collapsedIds.value.length === 0)
const expandSignal = ref({ open: allExpanded.value, n: 0 })

provide('helpCenterTreeExpand', expandSignal)
provide('helpCenterTreeCollapsed', {
  isCollapsed: (itemId) => collapsedIds.value.includes(itemId),
  setCollapsed: (itemId, collapsed) => {
    const next = collapsedIds.value.filter((id) => id !== itemId)
    if (collapsed) next.push(itemId)
    collapsedIds.value = next
  }
})

const collectionIds = (collections) =>
  collections.flatMap((collection) => [collection.id, ...collectionIds(collection.children || [])])

// Collapsed rows unmount their children, so the store is written here for the whole tree
// rather than left to each row to record itself.
const toggleExpandAll = () => {
  const open = !allExpanded.value
  collapsedIds.value = open ? [] : collectionIds(treeData.value)
  expandSignal.value = { open, n: expandSignal.value.n + 1 }
}

const orderMap = (ids) => Object.fromEntries(ids.map((itemId, index) => [itemId, index]))

const reorderCollections = async (ids) => {
  try {
    await api.updateCollectionSortOrders(props.id, orderMap(ids))
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
  fetchTree({ silent: true })
}

const moveArticleToCollection = async ({ articleId, collectionId, ids }) => {
  try {
    await api.moveArticleToCollection(articleId, { collection_id: collectionId })
    await api.updateArticleSortOrders(collectionId, orderMap(ids))
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
  fetchTree({ silent: true })
}

const reorderArticles = async ({ collectionId, ids }) => {
  try {
    await api.updateArticleSortOrders(collectionId, orderMap(ids))
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
  fetchTree({ silent: true })
}

const toggleStatus = async (item) => {
  try {
    if (item.type === 'collection') {
      item.is_published = !item.is_published
      await api.toggleCollection(item.id)
    } else {
      item.status = item.status === 'published' ? 'draft' : 'published'
      await api.updateArticleStatus(item.id, { status: item.status })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
  fetchTree({ silent: true })
}
</script>
