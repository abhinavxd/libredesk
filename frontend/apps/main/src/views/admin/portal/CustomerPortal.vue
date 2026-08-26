<template>
  <AdminSplitLayout>
    <template #content>
      <Tabs default-value="settings">
        <TabsList class="grid w-full grid-cols-2 mb-5">
          <TabsTrigger value="settings">{{ $t('globals.terms.setting', 2) }}</TabsTrigger>
          <TabsTrigger value="forms">{{ $t('admin.portalForm.tickets', 2) }}</TabsTrigger>
        </TabsList>
        <TabsContent value="settings">
          <LoadingOverlay :loading="isLoading">
            <PortalSettingForm
              v-if="!loadFailed"
              :submitForm="submitForm"
              :initial-values="initialValues"
              :inboxes="inboxes"
              :livechat-inboxes="livechatInboxes"
              :help-centers="helpCenters"
              :ticket-forms="ticketForms"
            />
          </LoadingOverlay>
        </TabsContent>
        <TabsContent value="forms">
          <PortalForms />
        </TabsContent>
      </Tabs>
    </template>
    <template #help>
      <p>{{ $t('admin.portal.help') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import PortalSettingForm from '@/features/admin/portal/PortalSettingForm.vue'
import PortalForms from '@main/views/admin/portal/PortalForms.vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@/api'

const initialValues = ref({})
const inboxes = ref([])
const livechatInboxes = ref([])
const helpCenters = ref([])
const ticketForms = ref([])
const isLoading = ref(false)
const emitter = useEmitter()

// A form that never loaded would save zod defaults over the live settings.
const loadFailed = ref(false)

onMounted(async () => {
  isLoading.value = true
  try {
    const [settingsResp, inboxesResp, helpCentersResp, formsResp] = await Promise.all([
      api.getSettings('portal'),
      api.getInboxes(),
      api.getHelpCenters(),
      api.getPortalForms()
    ])
    ticketForms.value = formsResp.data.data || []
    const allInboxes = inboxesResp.data.data || []
    inboxes.value = allInboxes.filter((inb) => inb.channel === 'email' && inb.enabled)
    livechatInboxes.value = allInboxes.filter((inb) => inb.channel === 'livechat' && inb.enabled)
    helpCenters.value = helpCentersResp.data.data || []
    const data = settingsResp.data.data
    initialValues.value = Object.keys(data).reduce((acc, key) => {
      acc[key.replace(/^portal\./, '')] = data[key]
      return acc
    }, {})
  } catch (error) {
    loadFailed.value = true
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

const submitForm = async (values) => {
  await api.updateSettings('portal', {
    'portal.enabled': values.enabled,
    'portal.tickets_from_article_only': values.tickets_from_article_only,
    'portal.inbox_id': Number(values.inbox_id) || 0,
    'portal.help_center_id': Number(values.help_center_id) || 0,
    'portal.livechat_inbox_id': Number(values.livechat_inbox_id) || 0,
    'portal.form_id': Number(values.form_id) || 0
  })
}
</script>
