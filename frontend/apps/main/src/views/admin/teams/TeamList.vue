<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-team' }">
        <Button> {{ $t('globals.messages.new') }} </Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="columns" :data="data" :loading="isLoading" />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { columns } from '../../../features/admin/teams/TeamsDataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import DataTable from '@main/components/datatable/DataTable.vue'
import api from '../../../api'

const emitter = useEmitter()
const data = ref([])
const isLoading = ref(false)

const getData = async () => {
  try {
    isLoading.value = true
    const response = await api.getTeams()
    data.value = response.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const refreshHandler = (event) => {
  if (event.model === 'team') {
    getData()
  }
}

const listenForRefresh = () => {
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
}

const removeListeners = () => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
}

onMounted(async () => {
  getData()
  listenForRefresh()
})

onUnmounted(() => {
  removeListeners()
})
</script>
