<template>
  <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
    <div class="flex justify-between mb-5">
      <div></div>
      <div>
        <router-link :to="{ name: 'new-sla' }">
          <Button>
            {{
              t('globals.messages.new', {
                name: t('globals.terms.slaPolicy')
              })
            }}
          </Button>
        </router-link>
      </div>
    </div>
    <div>
      <Spinner v-if="isLoading"></Spinner>
      <DataTable :columns="createColumns(t)" :data="slas" :loading="isLoading" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createColumns } from '../../../features/admin/sla/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import { useEmitter } from '../../../composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import api from '../../../api'

const { t } = useI18n()
const slas = ref([])
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
  if (data?.model === 'sla') fetchAll()
}

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAllSLAs()
    slas.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}
</script>
