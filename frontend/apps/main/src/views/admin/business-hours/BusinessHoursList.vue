<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <div class="flex justify-between mb-5">
      <div></div>
      <div>
        <router-link :to="{ name: 'new-business-hours' }">
          <Button>
            {{
              $t('businessHour.new')
            }}
          </Button>
        </router-link>
      </div>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="businessHours" :loading="isLoading" />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button'
import { useEmitter } from '../../../composables/useEmitter'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useI18n } from 'vue-i18n'
import { createColumns } from '../../../features/admin/business-hours/dataTableColumns.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import api from '../../../api'

const { t } = useI18n()
const businessHours = ref([])
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
  if (data?.model === 'business_hours') fetchAll()
}

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAllBusinessHours()
    businessHours.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}
</script>
