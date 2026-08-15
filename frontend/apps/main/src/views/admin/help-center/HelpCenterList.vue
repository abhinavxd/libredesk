<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="loading" reserve-height>
        <div class="flex justify-end mb-4">
          <Button @click="openCreateModal">
            {{ $t('globals.messages.new') }}
          </Button>
        </div>

        <DataTable :columns="columns" :data="helpCenters" :loading="loading" />
      </LoadingOverlay>
    </template>
    <template #help>
      <p>{{ $t('admin.helpCenter.help') }}</p>
    </template>
  </AdminSplitLayout>

  <Sheet :open="showCreateModal" @update:open="closeCreateModal">
    <SheetContent class="sm:max-w-lg overflow-y-auto">
      <SheetHeader>
        <SheetTitle>{{ $t('globals.messages.new') }}</SheetTitle>
      </SheetHeader>

      <HelpCenterBasicsForm
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
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createHelpCenterColumns } from '@/features/admin/help-center/helpCenterColumns.js'
import HelpCenterBasicsForm from '@/features/admin/help-center/HelpCenterBasicsForm.vue'
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
  showCreateModal.value = true
}

const openEditModal = (helpCenter) => {
  router.push({ name: 'help-center-customize', params: { id: helpCenter.id } })
}

const closeCreateModal = () => {
  showCreateModal.value = false
}

const handleSave = async (formData) => {
  try {
    isSubmitting.value = true
    const { data } = await api.createHelpCenter(formData)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    closeCreateModal()
    router.push({ name: 'help-center-customize', params: { id: data.data.id } })
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

const handleDelete = async (helpCenter) => {
  try {
    await api.deleteHelpCenter(helpCenter.id)
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

const columns = createHelpCenterColumns(t, {
  onOpen: (helpCenter) => goToTree(helpCenter.id),
  onEdit: openEditModal,
  onDelete: handleDelete,
  onToggle: handleToggle
})
</script>
