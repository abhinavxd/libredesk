<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
          <template #actions>
            <Dialog v-model:open="importDialogOpen">
              <DialogTrigger as-child>
                <Button variant="outline">{{ i18n._('Import from URL') }}</Button>
              </DialogTrigger>
              <DialogContent class="sm:max-w-[560px]">
                <DialogHeader>
                  <DialogTitle>{{ i18n._('Import from URL') }}</DialogTitle>
                </DialogHeader>
                <form class="space-y-4" @submit.prevent="importFromUrl">
                  <Input
                    v-model="importUrl"
                    type="url"
                    :placeholder="i18n._('https://example.com/help/article')"
                  />
                  <div class="flex justify-end">
                    <Button type="submit" :isLoading="isImporting" :disabled="!importUrl.trim()">
                      {{ i18n.pgettext('Action', 'Import') }}
                    </Button>
                  </div>
                </form>
              </DialogContent>
            </Dialog>
            <Dialog v-model:open="snippetDialogOpen">
              <DialogTrigger as-child @click="newSnippet">
                <Button>{{ i18n._('New snippet') }}</Button>
              </DialogTrigger>
              <DialogContent class="sm:max-w-3xl">
                <DialogHeader>
                  <DialogTitle>
                    {{ snippetEditing ? i18n._('Edit snippet') : i18n._('New snippet') }}
                  </DialogTitle>
                </DialogHeader>
                <SnippetForm
                  :initial-values="snippetInitial"
                  :is-editing="snippetEditing"
                  :submit-form="submitSnippet"
                />
              </DialogContent>
            </Dialog>
          </template>
        </AdminToolbar>
        <DataTable
          v-model:search="searchTerm"
          :searchable="false"
          :columns="createSnippetColumns(i18n, { onEdit: editSnippet })"
          :data="snippets"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ i18n._('Snippets are the knowledge base the assistant searches to answer questions.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai#snippets"
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import SnippetForm from '@/features/admin/ai/SnippetForm.vue'
import { createSnippetColumns } from '@/features/admin/ai/snippetColumns.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import api from '@/api'

const searchTerm = ref('')
const i18n = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)

const snippets = ref([])

const snippetDialogOpen = ref(false)
const importDialogOpen = ref(false)
const importUrl = ref('')
const isImporting = ref(false)
const snippetEditing = ref(false)
const snippetInitial = ref({})
const editingSnippetId = ref(null)

const refreshHandler = (data) => {
  if (data?.model === 'ai_snippets') getSnippets()
}

const editHandler = (data) => {
  if (data?.model === 'ai_snippets') editSnippet(data.data)
}

onMounted(() => {
  getSnippets()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const getSnippets = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAISnippets()
    snippets.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const importFromUrl = async () => {
  if (!importUrl.value.trim() || isImporting.value) return
  try {
    isImporting.value = true
    await api.importAISnippetFromURL({ url: importUrl.value.trim() })
    importDialogOpen.value = false
    importUrl.value = ''
    getSnippets()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isImporting.value = false
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
      description: i18n._('Changes saved')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
