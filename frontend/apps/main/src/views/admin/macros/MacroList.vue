<template>
  <LoadingOverlay :loading="formLoading" reserve-height>
    <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
      <template #actions>
        <router-link :to="{ name: 'new-macro' }">
          <Button>
            {{ $t('macro.new') }}
          </Button>
        </router-link>
      </template>
    </AdminToolbar>
    <div>
      <DataTable
        v-model:search="searchTerm"
        :searchable="false"
        :columns="createColumns(t)"
        :data="macros"
        :loading="formLoading"
      />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createColumns } from '@main/features/admin/macros/dataTableColumns.js'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { Button } from '@shared-ui/components/ui/button'
import { useI18n } from 'vue-i18n'
import api from '@main/api'

const searchTerm = ref('')
const { t } = useI18n()
const formLoading = ref(false)
const macros = ref([])
const emit = useEmitter()

onMounted(() => {
  getMacros()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'macros') getMacros()
}

const getMacros = async () => {
  try {
    formLoading.value = true
    const resp = await api.getMacrosCompact()
    macros.value = resp.data.data
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
