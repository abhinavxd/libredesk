<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
          <template #actions>
            <Button @click="router.push({ name: 'new-ai-tool' })">{{
              i18n._('New tool')
            }}</Button>
          </template>
        </AdminToolbar>
        <DataTable
          v-model:search="searchTerm"
          :searchable="false"
          :columns="createToolColumns(i18n, { onEdit: editTool })"
          :data="tools"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ i18n._('Tools let the assistant call your own APIs to fetch or update data while replying. A tool is shared across every assistant granted access to it, and the model decides when to call it from the name and description, so keep those precise.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai#tools"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ i18n._('Learn more') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { ref, onMounted, onUnmounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { createToolColumns } from '@/features/admin/ai/toolColumns.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import { useRouter } from 'vue-router'
import api from '@/api'

const searchTerm = ref('')
const i18n = useI18n()
const emitter = useEmitter()
const router = useRouter()
const isLoading = ref(false)

const tools = ref([])

const refreshHandler = (data) => {
  if (data?.model === 'ai_tools') getTools()
}

const editHandler = (data) => {
  if (data?.model === 'ai_tools') editTool(data.data)
}

const editTool = (item) => {
  router.push({ name: 'edit-ai-tool', params: { id: item.id } })
}

onMounted(() => {
  getTools()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const getTools = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAITools()
    tools.value = resp.data.data || []
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
