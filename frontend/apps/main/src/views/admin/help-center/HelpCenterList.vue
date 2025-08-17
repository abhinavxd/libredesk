<template>
  <Spinner v-if="loading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': loading }">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-2xl font-semibold">{{ $t('globals.terms.helpCenter') }}</h1>
        <p class="text-muted-foreground mt-1">
          Manage your help centers and knowledge base content
        </p>
      </div>
      <Button @click="openCreateModal">
        {{ $t('globals.messages.new', { name: $t('globals.terms.helpCenter') }) }}
      </Button>
    </div>

    <div v-if="helpCenters.length === 0 && !loading" class="text-center py-12">
      <div class="text-muted-foreground">
        <BookOpenIcon class="mx-auto h-12 w-12 mb-4" />
        <h3 class="text-lg font-medium mb-2">No help centers yet</h3>
        <p class="mb-4">Create your first help center to get started with knowledge management.</p>
        <Button @click="openCreateModal"> Create Help Center </Button>
      </div>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <HelpCenterCard
        v-for="helpCenter in helpCenters"
        :key="helpCenter.id"
        :help-center="helpCenter"
        @click="goToTree(helpCenter.id)"
        @edit="editHelpCenter"
        @delete="deleteHelpCenter"
      />
    </div>
  </div>

  <!-- Create/Edit Help Center Sheet -->
  <Sheet :open="showCreateModal" @update:open="closeCreateModal">
    <SheetContent class="sm:max-w-md">
      <SheetHeader>
        <SheetTitle>
          {{ editingHelpCenter ? 'Edit Help Center' : 'Create Help Center' }}
        </SheetTitle>
        <SheetDescription>
          {{
            editingHelpCenter
              ? 'Update your help center details.'
              : 'Create a new help center for your knowledge base.'
          }}
        </SheetDescription>
      </SheetHeader>

      <HelpCenterForm
        :help-center="editingHelpCenter"
        :submit-form="handleSave"
        :is-loading="isSubmitting"
        @cancel="closeCreateModal"
      />
    </SheetContent>
  </Sheet>

  <!-- Delete Confirmation Dialog -->
  <AlertDialog :open="showDeleteDialog" @update:open="showDeleteDialog = false">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Delete Help Center</AlertDialogTitle>
        <AlertDialogDescription>
          Are you sure you want to delete "{{ deletingHelpCenter?.name }}"? This action cannot be
          undone and will delete all collections and articles within this help center.
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
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { Button } from '@shared-ui/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle
} from '@shared-ui/components/ui/sheet'
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
import { BookOpenIcon } from 'lucide-vue-next'
import HelpCenterCard from '../../../features/admin/help-center/HelpCenterCard.vue'
import HelpCenterForm from '../../../features/admin/help-center/HelpCenterForm.vue'
import api from '../../../api'
import { handleHTTPError } from '../../../utils/http'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const emitter = useEmitter()
const { t } = useI18n()
const loading = ref(false)
const isSubmitting = ref(false)
const helpCenters = ref([])
const showCreateModal = ref(false)
const showDeleteDialog = ref(false)
const editingHelpCenter = ref(null)
const deletingHelpCenter = ref(null)

onMounted(() => {
  fetchHelpCenters()
})

const fetchHelpCenters = async () => {
  try {
    loading.value = true
    const { data } = await api.getHelpCenters()
    helpCenters.value = data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}

const goToTree = (helpCenterId) => {
  router.push({ name: 'help-center-tree', params: { id: helpCenterId } })
}

const openCreateModal = () => {
  editingHelpCenter.value = null
  showCreateModal.value = true
}

const editHelpCenter = (helpCenter) => {
  editingHelpCenter.value = helpCenter
  showCreateModal.value = true
}

const closeCreateModal = () => {
  showCreateModal.value = false
  editingHelpCenter.value = null
}

const handleSave = async (formData) => {
  try {
    isSubmitting.value = true
    if (editingHelpCenter.value) {
      await api.updateHelpCenter(editingHelpCenter.value.id, formData)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'success',
        description: t('globals.messages.updatedSuccessfully', {
          name: t('globals.terms.helpCenter')
        })
      })
    } else {
      await api.createHelpCenter(formData)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'success',
        description: t('globals.messages.createdSuccessfully', {
          name: t('globals.terms.helpCenter')
        })
      })
    }

    closeCreateModal()
    fetchHelpCenters()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmitting.value = false
  }
}

const deleteHelpCenter = (helpCenter) => {
  deletingHelpCenter.value = helpCenter
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  try {
    await api.deleteHelpCenter(deletingHelpCenter.value.id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: t('globals.messages.deletedSuccessfully', {
        name: t('globals.terms.helpCenter')
      })
    })
    showDeleteDialog.value = false
    deletingHelpCenter.value = null
    fetchHelpCenters()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
