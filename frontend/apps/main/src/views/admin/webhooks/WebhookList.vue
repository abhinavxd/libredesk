<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
      <template #actions>
        <RouterLink :to="{ name: 'new-webhook' }">
          <Button>{{ i18n._('New webhook') }}</Button>
        </RouterLink>
      </template>
    </AdminToolbar>
    <div>
      <DataTable
        v-model:search="searchTerm"
        :searchable="false"
        :columns="createColumns(i18n)"
        :data="webhooks"
        :loading="isLoading"
      />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createColumns } from '@/features/admin/webhooks/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from '@shared-ui/i18n'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const searchTerm = ref('')
const webhooks = ref([])
const i18n = useI18n()
const isLoading = ref(false)
const emit = useEmitter()

onMounted(() => {
  fetchAll()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'webhook') fetchAll()
}

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getWebhooks()
    webhooks.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}
</script>
