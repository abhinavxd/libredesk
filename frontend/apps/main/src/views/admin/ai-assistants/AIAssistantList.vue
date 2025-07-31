<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-ai-assistant' }">
        <Button>{{
          $t('globals.messages.new', {
            name: $t('globals.terms.aiAssistant', 1)
          })
        }}</Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="data" />
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { createColumns } from '../../../features/admin/ai-assistants/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import DataTable from '@main/components/datatable/DataTable.vue'
import { handleHTTPError } from '../../../utils/http'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import api from '../../../api'
import { useI18n } from 'vue-i18n'

const isLoading = ref(false)
const { t } = useI18n()
const data = ref([])
const emitter = useEmitter()

onMounted(async () => {
  getData()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'ai_assistant') getData()
  })
})

const getData = async () => {
  try {
    isLoading.value = true
    const response = await api.getAIAssistants()
    data.value = response.data.data
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