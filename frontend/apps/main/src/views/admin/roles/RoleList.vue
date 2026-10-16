<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
      <template #actions>
        <router-link :to="{ name: 'new-role' }">
          <Button>
            {{ $t('role.new') }}
          </Button>
        </router-link>
      </template>
    </AdminToolbar>
    <div>
      <DataTable
        v-model:search="searchTerm"
        :searchable="false"
        :columns="createColumns(t)"
        :data="roles"
        :loading="isLoading"
      />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { onMounted, onUnmounted, ref } from 'vue'
import { createColumns } from '@/features/admin/roles/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import DataTable from '@main/components/datatable/DataTable.vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const emitter = useEmitter()
const searchTerm = ref('')
const { t } = useI18n()
const roles = ref([])
const isLoading = ref(false)

const getRoles = async () => {
  try {
    isLoading.value = true
    const resp = await api.getRoles()
    roles.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const refreshHandler = (data) => {
  if (data?.model === 'team') getRoles()
}

onMounted(async () => {
  getRoles()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
})
</script>
