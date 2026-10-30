<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <div class="space-y-6">
    <div v-if="currentStep === 1" class="space-y-4 mt-10">
      <h3 class="ui-heading">{{ i18n._('Choose channel') }}</h3>
      <FormGrid>
        <MenuCard
          v-for="channel in channels"
          :key="channel.title"
          :onClick="channel.onClick"
          :title="channel.title"
          :subTitle="channel.subTitle"
          :icon="channel.icon"
          :badge="channel.badge"
        >
        </MenuCard>
      </FormGrid>
    </div>

    <div v-else-if="currentStep === 2" class="space-y-6">
      <Button @click="goBack" variant="link" size="xs">← {{ i18n._('Back') }}</Button>
      <div v-if="selectedChannel === 'email'">
        <EmailInboxForm
          :initial-values="{}"
          :submitForm="submitForm"
          :isLoading="isLoading"
          :isNewForm="true"
        />
      </div>
      <div v-else-if="selectedChannel === 'livechat'">
        <LivechatInboxForm
          :initial-values="{}"
          :submitForm="submitLiveChatForm"
          :isLoading="isLoading"
          :isNewForm="true"
          :available-languages="availableLanguages"
        />
      </div>
      <div v-else-if="selectedChannel === 'whatsapp'">
        <WhatsAppInboxForm
          :initial-values="{}"
          :submitForm="submitWhatsAppForm"
          :isLoading="isLoading"
          :isNewForm="true"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb/index.js'
import { Mail, MessageCircle } from 'lucide-vue-next'
import WhatsAppIcon from '@main/components/icons/WhatsAppIcon.vue'
import MenuCard from '@main/components/layout/MenuCard.vue'
import FormGrid from '@shared-ui/components/FormGrid.vue'
import EmailInboxForm from '@/features/admin/inbox/EmailInboxForm.vue'
import LivechatInboxForm from '@/features/admin/inbox/LivechatInboxForm.vue'
import WhatsAppInboxForm from '@/features/admin/inbox/WhatsAppInboxForm.vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)
const availableLanguages = ref([])
const currentStep = ref(1)
const selectedChannel = ref(null)
const router = useRouter()
const breadcrumbLinks = [
  { path: 'inbox-list', label: i18n._('{count, plural, one {Inbox} other {Inboxes}}', 2) },
  { path: '', label: i18n._('New inbox') }
]

const selectChannel = (channel) => {
  selectedChannel.value = channel
  currentStep.value = 2
}

const selectEmailChannel = () => {
  selectChannel('email')
}

const selectLiveChatChannel = () => {
  selectChannel('livechat')
}

const selectWhatsAppChannel = () => {
  selectChannel('whatsapp')
}

const channels = [
  {
    title: i18n.ngettext('Email', 'Emails', 1, {}, 'Communication channel'),
    subTitle: i18n._('Create an email inbox for email-based customer support'),
    onClick: selectEmailChannel,
    icon: Mail
  },
  {
    title: i18n._('Live Chat'),
    subTitle: i18n._('Create a live chat inbox for real-time customer support'),
    onClick: selectLiveChatChannel,
    icon: MessageCircle
  },
  {
    title: i18n._('WhatsApp'),
    subTitle: i18n._('Connect a WhatsApp number through the Meta Cloud API to receive and reply to contact messages on WhatsApp.'),
    onClick: selectWhatsAppChannel,
    icon: WhatsAppIcon,
    badge: i18n._('Beta')
  }
]

onMounted(async () => {
  try {
    const resp = await api.getAvailableLanguages()
    availableLanguages.value = resp.data.data
  } catch (error) {
    console.error('Error fetching available languages:', error)
  }
})

const goBack = () => {
  currentStep.value = 1
  selectedChannel.value = null
}

const submitForm = (values) => {
  const channelName = selectedChannel.value.toLowerCase()
  const payload = {
    name: values.name,
    from: values.from,
    aliases: values.aliases || [],
    from_name_template: values.from_name_template || '',
    channel: channelName,
    enabled: values.enabled ?? true,
    csat_enabled: values.csat_enabled ?? false,
    prompt_tags_on_reply: values.prompt_tags_on_reply ?? false,
    config: {
      reply_to: values.reply_to,
      enable_plus_addressing: values.enable_plus_addressing,
      imap: [values.imap],
      smtp: [values.smtp]
    }
  }
  createInbox(payload)
}

const submitLiveChatForm = (values) => {
  const payload = {
    name: values.name,
    channel: 'livechat',
    enabled: values.enabled ?? true,
    csat_enabled: values.csat_enabled ?? false,
    prompt_tags_on_reply: values.prompt_tags_on_reply ?? false,
    secret: values.secret ?? '',
    linked_email_inbox_id: values.linked_email_inbox_id ?? null,
    config: values.config
  }
  createInbox(payload)
}

const submitWhatsAppForm = (values) => {
  const payload = {
    name: values.name,
    from: values.from,
    channel: 'whatsapp',
    enabled: values.enabled ?? true,
    csat_enabled: values.csat_enabled ?? false,
    prompt_tags_on_reply: values.prompt_tags_on_reply ?? false,
    reopen_window_hours: values.reopen_window_hours ?? 0,
    config: values.config
  }
  createInbox(payload)
}

async function createInbox(payload) {
  try {
    isLoading.value = true
    await api.createInbox(payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
    router.push({ name: 'inbox-list' })
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
