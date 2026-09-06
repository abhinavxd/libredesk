<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <div class="flex justify-end mb-4">
          <Button @click="router.push({ name: 'new-guided-form' })">{{
            t('admin.guidedForms.new')
          }}</Button>
        </div>
        <DataTable
          :columns="createGuidedFormColumns(t, { onEdit: editForm })"
          :data="forms"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ t('admin.guidedForms.help') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { createGuidedFormColumns } from '@/features/admin/guidedForms/guidedFormColumns.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const router = useRouter()
const isLoading = ref(false)

const forms = ref([])

const refreshHandler = (data) => {
  if (data?.model === 'guided_forms') getForms()
}

const editHandler = (data) => {
  if (data?.model === 'guided_forms') editForm(data.data)
}

const editForm = (item) => {
  router.push({ name: 'edit-guided-form', params: { id: item.id } })
}

onMounted(() => {
  getForms()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const getForms = async () => {
  try {
    isLoading.value = true
    const resp = await api.getGuidedForms()
    forms.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}
</script>
