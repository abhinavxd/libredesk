<template>
  <div class="mb-5 flex items-center justify-between gap-4">
    <CustomBreadcrumb :links="breadcrumbLinks" />
    <div
      v-if="inbox.channel === 'livechat' && inbox.uuid"
      class="flex items-center gap-1.5 text-xs text-muted-foreground/70"
    >
      <span>UUID:</span>
      <code class="font-mono">{{ inbox.uuid }}</code>
      <CopyButton :text="inbox.uuid" />
    </div>
  </div>
  <Spinner v-if="formLoading"></Spinner>
  <div v-else>
    <EmailInboxForm
      :initialValues="inbox"
      :submitForm="submitForm"
      :isLoading="isLoading"
      :verifyAlias="verifyAlias"
      :aliasVerificationState="aliasVerificationState"
      v-if="inbox.channel === 'email'"
    />
    <LivechatInboxForm
      :initialValues="inbox"
      :submitForm="submitForm"
      :isLoading="isLoading"
      :available-languages="availableLanguages"
      v-else-if="inbox.channel === 'livechat'"
    />
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import api from '../../../api'
import EmailInboxForm from '@/features/admin/inbox/EmailInboxForm.vue'
import LivechatInboxForm from '@/features/admin/inbox/LivechatInboxForm.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb/index.js'
import CopyButton from '@/components/button/CopyButton.vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { AUTH_TYPE_PASSWORD, AUTH_TYPE_OAUTH2 } from '@/constants/auth.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const emitter = useEmitter()
const { t } = useI18n()
const formLoading = ref(false)
const isLoading = ref(false)
const inbox = ref({})
const aliasVerificationState = ref({})
let verificationPollTimer = null
let verificationPollAttempts = 0
const maxVerificationPollAttempts = 12
const availableLanguages = ref([])
const breadcrumbLinks = [
  { path: 'inbox-list', label: t('globals.terms.inbox', 2) },
  { path: '', label: t('inbox.edit') }
]

const submitForm = (values) => {
  let payload

  if (inbox.value.channel === 'email') {
    const config = {
      auth_type: values.auth_type,
      reply_to: values.reply_to,
      enable_plus_addressing: values.enable_plus_addressing,
      imap: [{ ...values.imap }],
      smtp: [{ ...values.smtp }]
    }

    if (values.auth_type === AUTH_TYPE_OAUTH2) {
      config.oauth = values.oauth
    }

    payload = {
      ...values,
      channel: inbox.value.channel,
      config
    }

    if (payload.config.imap[0].password?.includes('•')) {
      payload.config.imap[0].password = ''
    }

    if (payload.config.auth_type === AUTH_TYPE_OAUTH2) {
      if (payload.config.oauth.access_token?.includes('•')) {
        payload.config.oauth.access_token = ''
      }
      if (payload.config.oauth.client_secret?.includes('•')) {
        payload.config.oauth.client_secret = ''
      }
      if (payload.config.oauth.refresh_token?.includes('•')) {
        payload.config.oauth.refresh_token = ''
      }
    }

    payload.config.smtp.forEach((smtp) => {
      if (smtp.password?.includes('•')) {
        smtp.password = ''
      }
    })
  } else if (inbox.value.channel === 'livechat') {
    payload = {
      ...values,
      channel: inbox.value.channel,
      config: values.config
    }
  }

  updateInbox(payload)
}
const updateInbox = async (payload) => {
  try {
    isLoading.value = true
    await api.updateInbox(inbox.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
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

const verifyAlias = async (email) => {
  try {
    await api.verifyInboxAlias(inbox.value.id, { email })
    aliasVerificationState.value[email] = { verification_status: 'pending' }
    startVerificationPolling()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('admin.inbox.aliases.sendingVerificationStarted') })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const stopVerificationPolling = () => {
  if (verificationPollTimer) {
    clearInterval(verificationPollTimer)
    verificationPollTimer = null
  }
}

const pollAliasVerification = async () => {
  const isPending = (alias) =>
    (aliasVerificationState.value[alias.email]?.verification_status || alias.verification_status) === 'pending'

  if (!inbox.value?.aliases?.some(isPending)) {
    stopVerificationPolling()
    return
  }
  if (verificationPollAttempts >= maxVerificationPollAttempts) {
    stopVerificationPolling()
    return
  }

  try {
    verificationPollAttempts += 1
    const response = await api.getInbox(props.id)
    const aliases = response.data.data.aliases || []
    for (const alias of inbox.value.aliases || []) {
      const latest = aliases.find((item) => item.email === alias.email)
      if (latest) {
        aliasVerificationState.value[alias.email] = {
          verification_status: latest.verification_status,
          verified_at: latest.verified_at
        }
      }
    }
    if (!inbox.value.aliases.some(isPending)) {
      stopVerificationPolling()
    }
  } catch {
    // Polling is best-effort; the next interval will retry.
  }
}

const startVerificationPolling = () => {
  stopVerificationPolling()
  verificationPollAttempts = 0
  verificationPollTimer = setInterval(pollAliasVerification, 5000)
}

onMounted(async () => {
  try {
    formLoading.value = true
    const [resp, langsResp] = await Promise.all([
      api.getInbox(props.id),
      api.getAvailableLanguages()
    ])
    availableLanguages.value = langsResp.data.data
    let inboxData = resp.data.data

    // Modify the inbox data as per the zod schema.
    if (inboxData?.config?.imap) {
      inboxData.imap = inboxData?.config?.imap[0]
    }
    if (inboxData?.config?.smtp) {
      inboxData.smtp = inboxData?.config?.smtp[0]
    }
    inboxData.auth_type = inboxData?.config?.auth_type || AUTH_TYPE_PASSWORD
    inboxData.oauth = inboxData?.config?.oauth || {}
    inboxData.enable_plus_addressing = inboxData?.config?.enable_plus_addressing || false
    inboxData.reply_to = inboxData?.config?.reply_to || ''
    inboxData.aliases = inboxData?.aliases || []
    inbox.value = inboxData
    aliasVerificationState.value = {}
    if (inboxData.aliases.some((alias) => alias.verification_status === 'pending')) {
      startVerificationPolling()
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})

onUnmounted(stopVerificationPolling)

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>
