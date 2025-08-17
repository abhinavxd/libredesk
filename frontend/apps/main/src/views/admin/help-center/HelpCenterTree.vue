<template>
  <Spinner v-if="loading" />
  <div v-else class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-4">
        <Button variant="ghost" size="sm" @click="goBack">
          <ArrowLeftIcon class="h-4 w-4" />
        </Button>
        <div class="h-6 w-px bg-border"></div>
        <div>
          <h1 class="text-2xl font-semibold">{{ helpCenter?.name }}</h1>
          <p class="text-muted-foreground text-sm mt-1">Manage collections and articles</p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <!-- Help Center Actions Dropdown -->
        <DropdownMenu :modal="false">
          <DropdownMenuTrigger as-child>
            <Button variant="ghost" size="sm">
              <MoreVerticalIcon class="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem @click="editHelpCenter">
              <PencilIcon class="mr-2 h-4 w-4" />
              Edit Help Center
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem @click="deleteHelpCenter" class="text-destructive">
              <TrashIcon class="mr-2 h-4 w-4" />
              Delete Help Center
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <div class="h-4 w-px bg-border"></div>
        <Select v-model="selectedLocale" @update:modelValue="handleLocaleChange">
          <SelectTrigger class="w-40">
            <SelectValue placeholder="Language" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="language in sortedLanguages"
              :key="language.code"
              :value="language.code"
            >
              <span class="flex items-center gap-2">
                {{ language.nativeName }}
                <span
                  v-if="language.code === helpCenter?.default_locale"
                  class="text-xs text-muted-foreground"
                  >(Default)</span
                >
              </span>
            </SelectItem>
          </SelectContent>
        </Select>

        <Button @click="openCreateCollectionModal">
          <PlusIcon class="h-4 w-4" />
          New Collection
        </Button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="flex-1 min-h-0">
      <!-- Enhanced Tree Panel -->
      <div class="border rounded-lg shadow-sm p-6 h-full overflow-y-auto">
        <!-- Empty State -->
        <div v-if="treeData.length === 0 && !loading" class="text-center py-16">
          <div
            class="mx-auto w-24 h-24 bg-muted rounded-full flex items-center justify-center mb-6"
          >
            <FolderIcon class="h-12 w-12 text-muted-foreground" />
          </div>
          <h3 class="text-xl font-semibold mb-3">No collections yet</h3>
          <p class="text-muted-foreground mb-6 max-w-md mx-auto">
            Create your first collection to organize your help articles and make them easily
            discoverable by your customers.
          </p>
          <Button @click="openCreateCollectionModal" size="lg" class="px-8">
            <PlusIcon class="h-5 w-5 mr-2" />
            Create Collection
          </Button>
        </div>

        <!-- Tree View -->
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

  <!-- Article Edit Sheet -->
  <ArticleEditSheet
    :is-open="showArticleEditSheet"
    @update:open="showArticleEditSheet = $event"
    :article="editingArticle"
    :collection-id="editingArticle?.collection_id || createArticleCollectionId"
    :locale="selectedLocale"
    :submit-form="handleArticleSave"
    :is-loading="isSubmittingArticle"
    @cancel="closeEditSheet"
  />

  <!-- Collection Edit Sheet -->
  <CollectionEditSheet
    :is-open="showCollectionEditSheet"
    @update:open="showCollectionEditSheet = $event"
    :collection="editingCollection"
    :help-center-id="parseInt(id)"
    :parent-id="createCollectionParentId"
    :locale="selectedLocale"
    :submit-form="handleCollectionSave"
    :is-loading="isSubmittingCollection"
    @cancel="closeEditSheet"
  />

  <!-- Help Center Edit Sheet -->
  <Sheet :open="showHelpCenterEditSheet" @update:open="showHelpCenterEditSheet = false">
    <SheetContent class="sm:max-w-md">
      <SheetHeader>
        <SheetTitle>Edit Help Center</SheetTitle>
        <SheetDescription> Update your help center details. </SheetDescription>
      </SheetHeader>

      <HelpCenterForm
        :help-center="editingHelpCenter"
        :submit-form="handleHelpCenterSave"
        :is-loading="isSubmittingHelpCenter"
        @cancel="closeHelpCenterEditSheet"
      />
    </SheetContent>
  </Sheet>

  <!-- Delete Confirmation Dialog -->
  <AlertDialog :open="showDeleteDialog" @update:open="showDeleteDialog = false">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle
          >Delete
          {{
            deletingItem?.type === 'help_center' ? 'Help Center' : deletingItem?.type
          }}</AlertDialogTitle
        >
        <AlertDialogDescription>
          Are you sure you want to delete "{{ deletingItem?.name || deletingItem?.title }}"?
          {{
            deletingItem?.type === 'collection'
              ? 'This will also delete all articles within this collection.'
              : deletingItem?.type === 'help_center'
                ? 'This will permanently delete the entire help center including all collections and articles. This action cannot be undone.'
                : ''
          }}
          This action cannot be undone.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction
          @click="confirmDelete"
          class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
        >
          Delete
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { LANGUAGES } from '@shared-ui/constants'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
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
  ArrowLeft as ArrowLeftIcon,
  Folder as FolderIcon,
  Plus as PlusIcon,
  MoreVertical as MoreVerticalIcon,
  Pencil as PencilIcon,
  Trash as TrashIcon
} from 'lucide-vue-next'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle
} from '@shared-ui/components/ui/sheet'
import TreeView from '../../../features/admin/help-center/TreeView.vue'
import ArticleEditSheet from '../../../features/admin/help-center/ArticleEditSheet.vue'
import CollectionEditSheet from '../../../features/admin/help-center/CollectionEditSheet.vue'
import HelpCenterForm from '../../../features/admin/help-center/HelpCenterForm.vue'
import api from '../../../api'
import { handleHTTPError } from '../../../utils/http'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  id: {
    type: String,
    required: true
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
const selectedLocale = ref('en') // Will be updated when help center is fetched

const showDeleteDialog = ref(false)
const showArticleEditSheet = ref(false)
const showCollectionEditSheet = ref(false)
const showHelpCenterEditSheet = ref(false)
const editingArticle = ref(null)
const editingCollection = ref(null)
const editingHelpCenter = ref(null)
const createCollectionParentId = ref(null)
const createArticleCollectionId = ref(null)
const deletingItem = ref(null)

// Computed property to sort languages with default locale first
const sortedLanguages = computed(() => {
  if (!helpCenter.value?.default_locale) {
    return LANGUAGES
  }

  const defaultLocale = helpCenter.value.default_locale
  const defaultLang = LANGUAGES.find((lang) => lang.code === defaultLocale)
  const otherLangs = LANGUAGES.filter((lang) => lang.code !== defaultLocale)

  return defaultLang ? [defaultLang, ...otherLangs] : LANGUAGES
})

onMounted(async () => {
  await fetchHelpCenter()
  await fetchTree()
})

const fetchHelpCenter = async () => {
  try {
    const { data } = await api.getHelpCenter(props.id)
    helpCenter.value = data.data

    // Set the selected locale to the help center's default locale
    if (helpCenter.value.default_locale && selectedLocale.value === 'en') {
      selectedLocale.value = helpCenter.value.default_locale
    }
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
    const { data } = await api.getHelpCenterTree(props.id, { locale: selectedLocale.value })
    treeData.value = data.data.tree
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}

const handleLocaleChange = () => {
  selectedItem.value = null
  fetchHelpCenter()
  fetchTree()
}

const goBack = () => {
  router.push({ name: 'help-center-list' })
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

// Help Center operations
const editHelpCenter = () => {
  editingHelpCenter.value = helpCenter.value
  showHelpCenterEditSheet.value = true
}

const deleteHelpCenter = () => {
  deletingItem.value = { ...helpCenter.value, type: 'help_center' }
  showDeleteDialog.value = true
}

const handleHelpCenterSave = async (formData) => {
  isSubmittingHelpCenter.value = true
  try {
    await api.updateHelpCenter(props.id, formData)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: 'Help center updated successfully'
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

// Collection operations
const openCreateCollectionModal = (parentId = null) => {
  // Handle case where parentId might be an event object
  let actualParentId = null
  if (parentId && typeof parentId === 'object' && 'target' in parentId) {
    actualParentId = null
  } else if (typeof parentId === 'number' || typeof parentId === 'string') {
    actualParentId = parentId
  }

  editingCollection.value = null
  createCollectionParentId.value = actualParentId
  showCollectionEditSheet.value = true
}

const handleCollectionSave = async (formData) => {
  if (formData.parent_id === 0) {
    formData.parent_id = null
  }

  isSubmittingCollection.value = true
  try {
    const isEditing = !!editingCollection.value
    const payload = {
      ...formData,
      locale: selectedLocale.value
    }

    if (isEditing) {
      const targetId = editingCollection.value.id
      await api.updateCollection(props.id, targetId, payload)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'success',
        description: 'Collection updated successfully'
      })
    } else {
      if (createCollectionParentId.value !== null) {
        payload.parent_id = createCollectionParentId.value
      }
      await api.createCollection(props.id, payload)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'success',
        description: 'Collection created successfully'
      })
    }

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

// Article operations
const openCreateArticleModal = (collection) => {
  editingArticle.value = null
  createArticleCollectionId.value = collection.id
  showArticleEditSheet.value = true
}

const handleArticleSave = async (formData) => {
  isSubmittingArticle.value = true
  try {
    const isEditing = !!editingArticle.value
    const payload = {
      ...formData,
      locale: selectedLocale.value
    }

    if (isEditing) {
      const targetArticle = editingArticle.value

      // Check if collection is being changed
      const isCollectionChanged = formData.collection_id !== targetArticle.collection_id

      if (isCollectionChanged) {
        // Use the new endpoint that allows collection changes
        await api.updateArticleByID(targetArticle.id, payload)
      } else {
        // Use the original endpoint for same-collection updates
        await api.updateArticle(targetArticle.collection_id, targetArticle.id, payload)
      }

      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'success',
        description: t('globals.messages.updatedSuccessfully', {
          name: t('globals.terms.article')
        })
      })
    } else {
      await api.createArticle(createArticleCollectionId.value, payload)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'success',
        description: t('globals.messages.createdSuccessfully', {
          name: t('globals.terms.article')
        })
      })
    }

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

// Delete operations
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
        variant: 'success',
        description: 'Help center deleted successfully'
      })
      // Navigate back to help center list after deletion
      router.push({ name: 'help-center-list' })
      return
    } else {
      await api.deleteArticle(deletingItem.value.collection_id, deletingItem.value.id)
    }

    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: t('globals.messages.deletedSuccessfully', {
        name: t(`globals.terms.${deletingItem.value.type}`)
      })
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

// Status operations
const toggleStatus = async (item) => {
  try {
    if (item.type === 'collection') {
      await api.toggleCollection(item.id)
    } else {
      const newStatus = item.status === 'published' ? 'draft' : 'published'
      await api.updateArticleStatus(item.id, { status: newStatus })
    }

    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.status')
      })
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
