<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="formLoading"></Spinner>
  <EmailInboxForm :initialValues="inbox" :submitForm="submitForm" :isLoading="isLoading" v-else />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import EmailInboxForm from '@/features/admin/inbox/EmailInboxForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb/index.js'
import { Spinner } from '@/components/ui/spinner'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useI18n } from 'vue-i18n'

const emitter = useEmitter()
const { t } = useI18n()
const formLoading = ref(false)
const isLoading = ref(false)
const inbox = ref({})
const breadcrumbLinks = [
  { path: 'inbox-list', label: 'Inboxes' },
  { path: '', label: 'Edit Inbox' }
]

const submitForm = (values) => {
  const payload = {
    ...values,
    channel: inbox.value.channel,
    config: {
      imap: [{ ...values.imap }],
      smtp: [{ ...values.smtp }]
    }
  }

  // Set dummy IMAP password to empty string
  if (payload.config.imap[0].password?.includes('•')) {
    payload.config.imap[0].password = ''
  }

  // Set dummy SMTP passwords to empty strings
  payload.config.smtp.forEach((smtp) => {
    if (smtp.password?.includes('•')) {
      smtp.password = ''
    }
  })

  updateInbox(payload)
}
const updateInbox = async (payload) => {
  try {
    isLoading.value = true
    await api.updateInbox(inbox.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.inbox')
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  try {
    formLoading.value = true
    const resp = await api.getInbox(props.id)
    let inboxData = resp.data.data

    // Modify the inbox data as per the zod schema.
    if (inboxData?.config?.imap) {
      inboxData.imap = inboxData?.config?.imap[0]
    }
    if (inboxData?.config?.smtp) {
      inboxData.smtp = inboxData?.config?.smtp[0]
    }
    inbox.value = inboxData
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>
