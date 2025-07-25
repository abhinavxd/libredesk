<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
  <input id="fileUpload" type="file" hidden @change="handleFileUpload">
    <div class="flex justify-end mb-5">
     <Button @click="importAgent" class="mr-5" variant="secondary">{{
      $t('globals.messages.import', {
        name: $t('globals.terms.agent', 1)
      })
      }}</Button>
      <router-link :to="{ name: 'new-agent' }">
        <Button>{{
          $t('globals.messages.new', {
            name: $t('globals.terms.agent', 1)
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
import { createColumns } from '@/features/admin/agents/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/datatable/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'
import { useI18n } from 'vue-i18n'

const isLoading = ref(false)
const { t } = useI18n()
const data = ref([])
const emitter = useEmitter()

onMounted(async () => {
  getData()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'agent') getData()
  })
})

const getData = async () => {
  try {
    isLoading.value = true
    const response = await api.getUsers()
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

const importAgent = () => {
  const fileInput = document.getElementById('fileUpload')
  if (fileInput) {
    fileInput.click()
  }
}

const handleFileUpload = async (event) => {
  console.log('Test')
  const file = event.target.files[0]
  if (!file) return

  const formData = new FormData()
  formData.append('file', file)

  try {
    isLoading.value = true
    const response = await api.importAgents(formData)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: t('globals.messages.importSuccess', {
        name: t('globals.terms.agent', 1)
      })
    })
    getData()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
    event.target.value = ''
  }
}
</script>
