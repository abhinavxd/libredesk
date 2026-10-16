<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <div>
      <Tabs default-value="email_outgoing" v-model="templateType">
        <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
          <template #left>
            <TabsList class="w-max justify-start">
              <TabsTrigger value="email_outgoing">
                {{ $t('admin.template.outgoingEmailTemplates') }}
              </TabsTrigger>
              <TabsTrigger value="email_notification">
                {{ $t('admin.template.emailNotificationTemplates') }}
              </TabsTrigger>
            </TabsList>
          </template>
          <template #actions>
            <Button
              type="button"
              @click="navigateToNewTemplate"
              :disabled="templateType !== 'email_outgoing'"
            >
              {{ $t('globals.messages.newTemplate') }}
            </Button>
          </template>
        </AdminToolbar>
        <TabsContent value="email_outgoing">
          <DataTable
            v-model:search="searchTerm"
            :searchable="false"
            :columns="createOutgoingEmailTableColumns(t)"
            :data="templates"
            :loading="isLoading"
          />
        </TabsContent>
        <TabsContent value="email_notification">
          <DataTable
            v-model:search="searchTerm"
            :searchable="false"
            :columns="createEmailNotificationTableColumns(t)"
            :data="templates"
            :loading="isLoading"
          />
        </TabsContent>
      </Tabs>
    </div>
  </LoadingOverlay>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import {
  createOutgoingEmailTableColumns,
  createEmailNotificationTableColumns
} from '@main/features/admin/templates/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import { useRouter } from 'vue-router'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useStorage } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

const templateType = useStorage('templateType', 'email_outgoing')
const searchTerm = ref('')
const { t } = useI18n()
const templates = ref([])
const isLoading = ref(false)
const router = useRouter()
const emit = useEmitter()

onMounted(async () => {
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getTemplates(templateType.value)
    templates.value = resp.data.data
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

fetchAll()

const refreshList = (data) => {
  if (data?.model === 'templates') fetchAll()
}

const navigateToNewTemplate = () => {
  router.push({
    name: 'new-template',
    query: { type: templateType.value }
  })
}

watch(templateType, () => {
  searchTerm.value = ''
  templates.value = []
  fetchAll()
})
</script>
