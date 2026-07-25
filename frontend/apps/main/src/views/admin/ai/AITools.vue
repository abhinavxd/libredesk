<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <div class="flex justify-end mb-4">
          <Button @click="router.push({ name: 'new-ai-tool' })">{{
            t('admin.ai.tool.new')
          }}</Button>
        </div>
        <DataTable
          :columns="createToolColumns(t, { onEdit: editTool })"
          :data="tools"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ t('admin.ai.toolsHelp') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ t('globals.terms.learnMore') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { createToolColumns } from '@/features/admin/ai/toolColumns.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import api from '@/api'

const { t } = useI18n()
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
