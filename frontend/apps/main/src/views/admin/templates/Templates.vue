<template>
  <div>
    <Spinner v-if="isLoading" />
    <AdminPageWithHelp>
      <template #content>
        <template v-if="router.currentRoute.value.path === '/admin/templates'">
          <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
            <div class="flex justify-between mb-5">
              <div></div>
              <div class="flex justify-end mb-4">
                <Button
                  @click="navigateToNewTemplate"
                  :disabled="templateType !== 'email_outgoing'"
                >
                  {{
                    $t('globals.messages.new', {
                      name: $t('globals.terms.template')
                    })
                  }}
                </Button>
              </div>
            </div>
            <div>
              <Tabs default-value="email_outgoing" v-model="templateType">
                <TabsList class="grid w-full grid-cols-2 mb-5">
                  <TabsTrigger value="email_outgoing">
                    {{ $t('admin.template.outgoingEmailTemplates') }}
                  </TabsTrigger>
                  <TabsTrigger value="email_notification">
                    {{ $t('admin.template.emailNotificationTemplates') }}
                  </TabsTrigger>
                </TabsList>
                <TabsContent value="email_outgoing">
                  <DataTable :columns="createOutgoingEmailTableColumns(t)" :data="templates" :loading="isLoading" />
                </TabsContent>
                <TabsContent value="email_notification">
                  <DataTable :columns="createEmailNotificationTableColumns(t)" :data="templates" :loading="isLoading" />
                </TabsContent>
              </Tabs>
            </div>
          </div>
        </template>
        <template v-else>
          <router-view />
        </template>
      </template>

      <template #help>
        <p>Design templates for customer communications and responses.</p>
        <p>Modify content for internal and external emails.</p>
        <a
          href="https://docs.libredesk.io/configuration/email-templates"
          target="_blank"
          rel="noopener noreferrer"
          class="link-style"
        >
          <p>Learn more</p>
        </a>
      </template>
    </AdminPageWithHelp>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import {
  createOutgoingEmailTableColumns,
  createEmailNotificationTableColumns
} from '../../../features/admin/templates/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import { useRouter, useRoute } from 'vue-router'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useStorage } from '@vueuse/core'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '../../../utils/http'
import api from '../../../api'

const templateType = useStorage('templateType', 'email_outgoing')
const { t } = useI18n()
const templates = ref([])
const isLoading = ref(false)
const router = useRouter()
const route = useRoute()
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

// When back to template list, refetch all items.
watch(
  () => route.name,
  () => {
    if (route.name === 'templates') {
      fetchAll()
    }
  }
)
</script>
