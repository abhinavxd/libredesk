<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <div class="flex justify-between mb-5">
      <div></div>
      <div class="flex justify-end mb-4">
        <Button @click="navigateToNewTemplate" :disabled="templateType !== 'email_outgoing'">
          {{ $t('template.new') }}
        </Button>
      </div>
    </div>
    <div>
      <Tabs default-value="email_outgoing" v-model="templateType">
        <TabsList class="flex w-full justify-start overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden mb-5 sm:grid sm:grid-cols-2">
          <TabsTrigger value="email_outgoing" :class="TEMPLATE_TAB_CLASS">
            {{ $t('admin.template.outgoingEmailTemplates') }}
          </TabsTrigger>
          <TabsTrigger value="email_notification" :class="TEMPLATE_TAB_CLASS">
            {{ $t('admin.template.emailNotificationTemplates') }}
          </TabsTrigger>
        </TabsList>
        <TabsContent value="email_outgoing">
          <DataTable
            :columns="createOutgoingEmailTableColumns(t)"
            :data="templates"
            :loading="isLoading"
          />
        </TabsContent>
        <TabsContent value="email_notification">
          <DataTable
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
const TEMPLATE_TAB_CLASS = 'shrink-0 sm:min-w-0'

import { ref, onMounted, onUnmounted, watch } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
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
  templates.value = []
  fetchAll()
})
</script>
