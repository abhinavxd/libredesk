<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="loading" reserve-height>
        <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
          <template #actions>
            <Button @click="openCreateModal">
              {{ i18n.pgettext('New help center', 'New') }}
            </Button>
          </template>
        </AdminToolbar>

        <DataTable
          v-model:search="searchTerm"
          :searchable="false"
          :columns="columns"
          :data="helpCenters"
          :loading="loading"
        />
      </LoadingOverlay>
    </template>
    <template #help>
      <p>{{ i18n._('Create public help centers where customers browse and search articles. Published articles can also be used by the AI agent to answer conversations.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/help-center"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ i18n._('Learn more') }}
      </a>
    </template>
  </AdminSplitLayout>

  <Sheet :open="showCreateModal" @update:open="closeCreateModal">
    <SheetContent class="sm:max-w-lg overflow-y-auto">
      <SheetHeader>
        <SheetTitle>{{ i18n.pgettext('New help center', 'New') }}</SheetTitle>
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
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

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
import { useI18n } from '@shared-ui/i18n'

const searchTerm = ref('')
const router = useRouter()
const emitter = useEmitter()
const i18n = useI18n()
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
      description: i18n._('Changes saved')
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
      description: i18n._('Changes saved')
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
      description: i18n._('Deleted successfully')
    })
    fetchHelpCenters()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const columns = createHelpCenterColumns(i18n, {
  onOpen: (helpCenter) => goToTree(helpCenter.id),
  onEdit: openEditModal,
  onDelete: handleDelete,
  onToggle: handleToggle
})
</script>
