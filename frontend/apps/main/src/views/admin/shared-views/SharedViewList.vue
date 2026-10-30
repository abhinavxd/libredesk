<template>
  <LoadingOverlay :loading="formLoading" reserve-height>
    <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
      <template #actions>
        <router-link :to="{ name: 'new-shared-view' }">
          <Button>
            {{ i18n._('New shared view') }}
          </Button>
        </router-link>
      </template>
    </AdminToolbar>
    <div>
      <DataTable
        v-model:search="searchTerm"
        :searchable="false"
        :columns="createColumns(i18n)"
        :data="sharedViews"
        :loading="formLoading"
      />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/datatable/DataTable.vue'
import { createColumns } from '@/features/admin/shared-views/dataTableColumns.js'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { Button } from '@shared-ui/components/ui/button'
import { useI18n } from '@shared-ui/i18n'
import api from '@/api'

const searchTerm = ref('')
const i18n = useI18n()
const formLoading = ref(false)
const sharedViews = ref([])
const emit = useEmitter()

onMounted(() => {
  getSharedViews()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'shared-views') getSharedViews()
}

const getSharedViews = async () => {
  try {
    formLoading.value = true
    const resp = await api.getAllSharedViews()
    sharedViews.value = resp.data.data
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
