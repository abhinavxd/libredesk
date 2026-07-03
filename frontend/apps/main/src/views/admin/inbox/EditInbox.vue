<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="formLoading"></Spinner>
  <div v-else>
    <EmailInboxForm
      :initialValues="inbox"
      :submitForm="submitForm"
      :isLoading="isLoading"
      v-if="inbox.channel === 'email'"
    />
    <LivechatInboxForm
      :initialValues="inbox"
      :submitForm="submitForm"
      :isLoading="isLoading"
      :available-languages="availableLanguages"
      v-else-if="inbox.channel === 'livechat'"
    />
    <TwitterInboxForm
      :initialValues="inbox"
      :submitForm="submitForm"
      :isLoading="isLoading"
      v-else-if="inbox.channel === 'twitter'"
    />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../../../api'
import EmailInboxForm from '@/features/admin/inbox/EmailInboxForm.vue'
import LivechatInboxForm from '@/features/admin/inbox/LivechatInboxForm.vue'
import TwitterInboxForm from '@/features/admin/inbox/TwitterInboxForm.vue'
import {
  parseTwitterActivityEventTypes,
  parseTwitterStreamRules
} from '@/features/admin/inbox/twitterFormSchema.js'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb/index.js'
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
  } else if (inbox.value.channel === 'twitter') {
    const oauth = { ...values.oauth }
    for (const key of ['access_token', 'client_secret', 'refresh_token']) {
      if (oauth[key]?.includes('•')) {
        oauth[key] = ''
      }
    }
    const webhookConsumerSecret = values.webhook_consumer_secret?.includes('•')
      ? ''
      : values.webhook_consumer_secret
    payload = {
      name: values.name,
      from: values.screen_name,
      channel: inbox.value.channel,
      enabled: values.enabled ?? true,
      csat_enabled: values.csat_enabled ?? false,
      prompt_tags_on_reply: values.prompt_tags_on_reply ?? false,
      config: {
        account_user_id: values.account_user_id,
        screen_name: values.screen_name,
        auth_type: values.auth_type,
        oauth,
        provider: values.provider,
        base_url: values.base_url,
        delivery_mode: values.delivery_mode,
        filtered_stream: {
          rules: parseTwitterStreamRules(values.filtered_stream_rules_json)
        },
        webhook: {
          id: values.webhook_id,
          url: values.webhook_url,
          consumer_secret: webhookConsumerSecret
        },
        activity: {
          subscription_id: values.activity_subscription_id,
          event_types: parseTwitterActivityEventTypes(values.activity_event_types)
        },
        ingest_dms: values.ingest_dms,
        ingest_mentions: values.ingest_mentions,
        poll_interval: values.poll_interval,
        scan_since: values.scan_since,
        dm_cursor: values.dm_cursor,
        mentions_cursor: values.mentions_cursor
      }
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
