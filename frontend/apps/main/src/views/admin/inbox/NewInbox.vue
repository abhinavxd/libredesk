<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <div class="space-y-10">
    <div class="mt-10">
      <Stepper class="flex w-full items-start gap-2" v-model="currentStep">
        <StepperItem
          v-for="step in steps"
          :key="step.step"
          v-slot="{ state }"
          class="relative flex w-full flex-col items-center justify-center"
          :step="step.step"
        >
          <StepperSeparator
            v-if="step.step !== steps[steps.length - 1].step"
            class="absolute left-[calc(50%+20px)] right-[calc(-50%+10px)] top-5 block h-0.5 shrink-0 rounded-full bg-muted group-data-[state=completed]:bg-primary"
          />

          <div>
            <Button
              :variant="state === 'completed' || state === 'active' ? 'default' : 'outline'"
              size="icon"
              class="z-10 rounded-full shrink-0"
              :class="[
                state === 'active' && 'ring-2 ring-ring ring-offset-2 ring-offset-background'
              ]"
            >
              <Check v-if="state === 'completed'" class="size-5" />
              <span v-if="state === 'active'">{{ currentStep }}</span>
              <span v-if="state === 'inactive'">{{ step.step }}</span>
            </Button>
          </div>

          <div class="mt-5 flex flex-col items-center text-center">
            <StepperTitle
              :class="[state === 'active' && 'text-primary']"
              class="text-sm font-semibold transition lg:text-base"
            >
              {{ step.title }}
            </StepperTitle>
            <StepperDescription
              :class="[state === 'active' && 'text-primary']"
              class="sr-only text-xs text-muted-foreground transition md:not-sr-only lg:text-sm"
            >
              {{ step.description }}
            </StepperDescription>
          </div>
        </StepperItem>
      </Stepper>
    </div>

    <div>
      <div v-if="currentStep === 1" class="flex space-x-6">
        <MenuCard
          v-for="channel in channels"
          :key="channel.title"
          :onClick="channel.onClick"
          :title="channel.title"
          :subTitle="channel.subTitle"
          :icon="channel.icon"
          class="w-full max-w-sm cursor-pointer"
        >
        </MenuCard>
      </div>

      <div v-else-if="currentStep === 2" class="space-y-6">
        <Button @click="goBack" variant="link" size="xs">← {{ $t('globals.messages.back') }}</Button>
        <div v-if="selectedChannel === 'email'">
          <EmailInboxForm :initial-values="{}" :submitForm="submitForm" :isLoading="isLoading" />
        </div>
        <div v-else-if="selectedChannel === 'livechat'">
          <LivechatInboxForm :initial-values="{}" :submitForm="submitLiveChatForm" :isLoading="isLoading" />
        </div>
      </div>

      <div v-else>
        <Button @click="goInboxList" variant="link" size="xs" class="mt-10"
          >← {{ $t('globals.messages.back') }}</Button
        >
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { useRouter } from 'vue-router'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb/index.js'
import { Check, Mail, MessageCircle } from 'lucide-vue-next'
import MenuCard from '@main/components/layout/MenuCard.vue'
import {
  Stepper,
  StepperDescription,
  StepperItem,
  StepperSeparator,
  StepperTitle
} from '@shared-ui/components/ui/stepper'
import EmailInboxForm from '@/features/admin/inbox/EmailInboxForm.vue'
import LivechatInboxForm from '@/features/admin/inbox/LivechatInboxForm.vue'
import api from '../../../api'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '../../../utils/http'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)
const currentStep = ref(1)
const selectedChannel = ref(null)
const router = useRouter()
const breadcrumbLinks = [
  { path: 'inbox-list', label: t('globals.terms.inbox', 2) },
  { path: '', label: t('globals.messages.new', { name: t('globals.terms.inbox') }) }
]

const steps = [
  {
    step: 1,
    title: t('globals.terms.channel'),
    description: t('admin.inbox.chooseChannel')
  },
  {
    step: 2,
    title: t('globals.terms.configure'),
    description: t('admin.inbox.configureChannel')
  }
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
    title: 'Live Chat',
    subTitle: 'Create a live chat inbox for real-time customer support',
    onClick: selectLiveChatChannel,
    icon: MessageCircle
  }
]

const goBack = () => {
  currentStep.value = 1
  selectedChannel.value = null
}

const goInboxList = () => {
  router.push('/admin/inboxes')
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
    config: values.config
  }
  createInbox(payload)
}

async function createInbox(payload) {
  try {
    isLoading.value = true
    await api.createInbox(payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.inbox')
      })
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
