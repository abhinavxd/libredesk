<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <Tabs v-model="activeTab" default-value="providers">
          <TabsList class="grid w-full grid-cols-3 mb-5">
            <TabsTrigger value="providers">{{ t('admin.ai.providers') }}</TabsTrigger>
            <TabsTrigger value="snippets">{{ t('admin.ai.snippets') }}</TabsTrigger>
            <TabsTrigger value="tools">{{ t('admin.ai.tools') }}</TabsTrigger>
          </TabsList>

          <TabsContent value="providers">
            <div class="space-y-6">
              <ProviderForm
                type="completion"
                :title="t('admin.ai.completion')"
                :description="t('admin.ai.completionDescription')"
                :show-completion-fields="true"
              />
              <ProviderForm
                type="embedding"
                :title="t('admin.ai.embedding')"
                :description="t('admin.ai.embeddingDescription')"
                :show-embedding-fields="true"
              />
            </div>
          </TabsContent>

          <TabsContent value="snippets">
            <div class="flex justify-end mb-4">
              <Dialog v-model:open="snippetDialogOpen">
                <DialogTrigger as-child @click="newSnippet">
                  <Button>{{ t('admin.ai.snippet.new') }}</Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[560px]">
                  <DialogHeader>
                    <DialogTitle>
                      {{ snippetEditing ? t('admin.ai.snippet.edit') : t('admin.ai.snippet.new') }}
                    </DialogTitle>
                  </DialogHeader>
                  <SnippetForm
                    :initial-values="snippetInitial"
                    :is-editing="snippetEditing"
                    :submit-form="submitSnippet"
                  />
                </DialogContent>
              </Dialog>
            </div>
            <DataTable
              :columns="createSnippetColumns(t, { onEdit: editSnippet })"
              :data="snippets"
              :loading="isSnippetsLoading"
            />
          </TabsContent>

          <TabsContent value="tools">
            <div class="flex justify-end mb-4">
              <Button @click="router.push({ name: 'new-ai-tool' })">{{
                t('admin.ai.tool.new')
              }}</Button>
            </div>
            <DataTable
              :columns="createToolColumns(t, { onEdit: editTool })"
              :data="tools"
              :loading="isToolsLoading"
            />
          </TabsContent>
        </Tabs>
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ t('admin.ai.description') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs/index.js'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
import ProviderForm from '@/features/admin/ai/ProviderForm.vue'
import SnippetForm from '@/features/admin/ai/SnippetForm.vue'
import { createSnippetColumns } from '@/features/admin/ai/snippetColumns.js'
import { createToolColumns } from '@/features/admin/ai/toolColumns.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const route = useRoute()
const router = useRouter()
const isSnippetsLoading = ref(false)
const isToolsLoading = ref(false)
const isLoading = computed(() => isSnippetsLoading.value || isToolsLoading.value)
const activeTab = ref(route.query.tab || 'providers')

const snippets = ref([])
const tools = ref([])

const snippetDialogOpen = ref(false)
const snippetEditing = ref(false)
const snippetInitial = ref({})
const editingSnippetId = ref(null)

const refreshHandler = (data) => {
  if (data?.model === 'ai_snippets') getSnippets()
  if (data?.model === 'ai_tools') getTools()
}

const editHandler = (data) => {
  if (data?.model === 'ai_snippets') editSnippet(data.data)
  if (data?.model === 'ai_tools') editTool(data.data)
}

const editTool = (item) => {
  router.push({ name: 'edit-ai-tool', params: { id: item.id } })
}

onMounted(() => {
  getSnippets()
  getTools()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const getSnippets = async () => {
  try {
    isSnippetsLoading.value = true
    const resp = await api.getAISnippets()
    snippets.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSnippetsLoading.value = false
  }
}

const getTools = async () => {
  try {
    isToolsLoading.value = true
    const resp = await api.getAITools()
    tools.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isToolsLoading.value = false
  }
}

const newSnippet = () => {
  snippetEditing.value = false
  editingSnippetId.value = null
  snippetInitial.value = {}
}

const editSnippet = (item) => {
  snippetEditing.value = true
  editingSnippetId.value = item.id
  snippetInitial.value = { ...item }
  snippetDialogOpen.value = true
}

const submitSnippet = async (values) => {
  try {
    if (snippetEditing.value) {
      await api.updateAISnippet(editingSnippetId.value, values)
    } else {
      await api.createAISnippet(values)
    }
    snippetDialogOpen.value = false
    getSnippets()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
