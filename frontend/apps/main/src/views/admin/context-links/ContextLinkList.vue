<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <div class="flex justify-between mb-5">
      <div></div>
      <div>
        <RouterLink :to="{ name: 'new-context-link' }">
          <Button>{{ $t('contextLink.new') }}</Button>
        </RouterLink>
      </div>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="links" :loading="isLoading" :searchable="false" />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createColumns } from '@/features/admin/context-links/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const links = ref([])
const { t } = useI18n()
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
  if (data?.model === 'context-link') fetchAll()
}

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getContextLinks()
    links.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}
</script>
