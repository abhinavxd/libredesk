<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="loading" reserve-height>
        <div class="flex justify-end mb-5">
          <Button @click="openCreateModal">
            {{ $t('globals.messages.new') }}
          </Button>
        </div>

        <div
          v-if="helpCenters.length === 0 && !loading"
          class="text-center py-12 text-muted-foreground"
        >
          <BookOpen class="mx-auto h-12 w-12 mb-4" />
          <p>{{ $t('helpCenter.noHelpCenters') }}</p>
        </div>

        <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <HelpCenterCard
            v-for="helpCenter in helpCenters"
            :key="helpCenter.id"
            :help-center="helpCenter"
            @click="goToTree(helpCenter.id)"
            @edit="openEditModal"
            @delete="handleDelete"
            @toggle="handleToggle"
          />
        </div>
      </LoadingOverlay>
    </template>
    <template #help>
      <p>{{ $t('admin.helpCenter.help') }}</p>
    </template>
  </AdminSplitLayout>

  <Sheet :open="showCreateModal" @update:open="closeCreateModal">
    <SheetContent class="sm:max-w-lg overflow-y-auto">
      <SheetHeader>
        <SheetTitle>
          {{ editingHelpCenter ? $t('globals.messages.edit') : $t('globals.messages.new') }}
        </SheetTitle>
      </SheetHeader>

      <HelpCenterForm
        :help-center="editingHelpCenter"
        :submit-form="handleSave"
        :is-loading="isSubmitting"
        @cancel="closeCreateModal"
      />
    </SheetContent>
  </Sheet>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Button } from '@shared-ui/components/ui/button'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@shared-ui/components/ui/sheet'
import { BookOpen } from 'lucide-vue-next'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import HelpCenterCard from '@/features/admin/help-center/HelpCenterCard.vue'
import HelpCenterForm from '@/features/admin/help-center/HelpCenterForm.vue'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const emitter = useEmitter()
const { t } = useI18n()
const loading = ref(false)
const isSubmitting = ref(false)
const helpCenters = ref([])
const showCreateModal = ref(false)
const editingHelpCenter = ref(null)

onMounted(() => {
  fetchHelpCenters()
})

const fetchHelpCenters = async () => {
  try {
    loading.value = true
    const { data } = await api.getHelpCenters()
    helpCenters.value = data.data || []
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

const openEditModal = (helpCenter) => {
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
    } else {
      await api.createHelpCenter(formData)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
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

const handleToggle = async (helpCenter) => {
  try {
    await api.toggleHelpCenter(helpCenter.id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    fetchHelpCenters()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const handleDelete = async (id) => {
  try {
    await api.deleteHelpCenter(id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })
    fetchHelpCenters()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
