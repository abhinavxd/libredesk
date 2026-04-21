<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <div class="space-y-6">
    <div v-if="currentStep === 1" class="space-y-4 mt-10">
      <h3 class="font-semibold text-lg">{{ $t('admin.inbox.chooseChannel') }}</h3>
      <div class="flex space-x-6">
        <MenuCard
          v-for="channel in channels"
          :key="channel.title"
          :onClick="channel.onClick"
          :title="channel.title"
          :subTitle="channel.subTitle"
          :icon="channel.icon"
          :badge="channel.badge"
          class="w-full max-w-sm cursor-pointer"
        >
        </MenuCard>
      </div>
    </div>

    <div v-else-if="currentStep === 2" class="space-y-6">
      <Button @click="goBack" variant="link" size="xs">← {{ $t('globals.messages.back') }}</Button>
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
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb/index.js'
import { Mail, MessageCircle } from 'lucide-vue-next'
import MenuCard from '@main/components/layout/MenuCard.vue'
import EmailInboxForm from '@/features/admin/inbox/EmailInboxForm.vue'
import LivechatInboxForm from '@/features/admin/inbox/LivechatInboxForm.vue'
import api from '../../../api'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)
const availableLanguages = ref([])
const currentStep = ref(1)
const selectedChannel = ref(null)
const router = useRouter()
const breadcrumbLinks = [
  { path: 'inbox-list', label: t('globals.terms.inbox', 2) },
  { path: '', label: t('inbox.newInbox') }
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

const channels = [
  {
    title: t('globals.terms.email'),
    subTitle: t('admin.inbox.createEmailInbox'),
    onClick: selectEmailChannel,
    icon: Mail
  },
  {
    title: t('globals.terms.liveChat'),
    subTitle: t('admin.inbox.createLiveChatInbox'),
    onClick: selectLiveChatChannel,
    icon: MessageCircle,
    badge: t('globals.terms.beta')
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
    channel: channelName,
    config: {
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
    config: values.config
  }
  createInbox(payload)
}

async function createInbox(payload) {
  try {
    isLoading.value = true
    await api.createInbox(payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
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
