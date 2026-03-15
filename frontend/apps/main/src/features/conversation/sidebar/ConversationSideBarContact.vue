<template>
  <div class="space-y-2">
    <div class="flex justify-between items-start">
      <div class="relative">
        <Avatar class="size-20">
          <AvatarImage :src="conversation?.contact?.avatar_url || getGravatarUrl(conversation?.contact?.email)" />
          <AvatarFallback>
            {{ conversation?.contact?.first_name?.toUpperCase().substring(0, 2) }}
          </AvatarFallback>
        </Avatar>
        <StatusDot
          v-if="isLivechat"
          :status="contactStatus"
          size="lg"
          class="absolute bottom-1 right-1 border-2 border-background"
        />
      </div>
      <Button
        variant="ghost"
        size="icon"
        class="h-7 w-7"
        @click="emitter.emit(EMITTER_EVENTS.CONVERSATION_SIDEBAR_TOGGLE)"
      >
        <ViewVerticalIcon />
      </Button>
    </div>

    <div class="h-6 flex items-center gap-2">
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-24 h-4" />
      </span>
      <span v-else>
        {{ conversation?.contact?.first_name + ' ' + conversation?.contact?.last_name }}
      </span>
      <ExternalLink
        v-if="!conversationStore.conversation.loading && userStore.can('contacts:read')"
        size="16"
        class="text-muted-foreground cursor-pointer flex-shrink-0"
        @click="$router.push({ name: 'contact-detail', params: { id: conversation?.contact_id } })"
      />
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Mail size="16" class="flex-shrink-0" />
      <Tooltip v-if="isLivechat && !conversationStore.conversation.loading">
        <TooltipTrigger as-child>
          <ShieldCheck v-if="isVerified" size="14" class="flex-shrink-0 text-green-600" />
          <ShieldQuestion v-else size="14" class="flex-shrink-0 text-amber-500" />
        </TooltipTrigger>
        <TooltipContent>{{ isVerified ? t('contact.identityVerified') : t('contact.identityNotVerified') }}</TooltipContent>
      </Tooltip>
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else-if="conversation?.contact?.email" class="break-all">
        {{ conversation?.contact?.email }}
      </span>
      <span v-else class="text-muted-foreground">
        {{ t('conversation.sidebar.notAvailable') }}
      </span>
    </div>
    <div class="text-sm text-muted-foreground flex gap-2 items-center">
      <Phone size="16" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else>
        {{ phoneNumber }}
      </span>
    </div>
    <div
      class="text-sm text-muted-foreground flex gap-2 items-center"
      v-if="conversation?.contact?.external_user_id"
    >
      <IdCard size="16" class="flex-shrink-0" />
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else>
        {{ conversation.contact.external_user_id }}
      </span>
    </div>

    <!-- Livechat visitor info -->
    <template v-if="isLivechat && !conversationStore.conversation.loading">
      <div
        v-if="conversation?.contact?.country"
        class="text-sm text-muted-foreground flex gap-2 items-center"
      >
        <Globe size="16" class="flex-shrink-0" />
        <span>{{ countryName }}</span>
      </div>
      <div
        v-if="conversation?.meta?.ip"
        class="text-sm text-muted-foreground flex gap-2 items-center"
      >
        <Monitor size="16" class="flex-shrink-0" />
        <span class="break-all">{{ conversation.meta.ip }}</span>
      </div>
      <div
        v-if="conversation?.meta?.user_agent"
        class="text-sm text-muted-foreground flex gap-2 items-center"
      >
        <Smartphone size="16" class="flex-shrink-0" />
        <span class="break-all">{{ parsedUA }}</span>
      </div>
    </template>

  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ViewVerticalIcon } from '@radix-icons/vue'
import { Button } from '@shared-ui/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import StatusDot from '@shared-ui/components/StatusDot.vue'
import { Mail, Phone, ExternalLink, IdCard, Globe, Monitor, Smartphone, ShieldCheck, ShieldQuestion } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import countries from '@/constants/countries.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useConversationStore } from '@/stores/conversation'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { useUserStore } from '@/stores/user'
import { getGravatarUrl } from '@shared-ui/utils/gravatar.js'
import { useI18n } from 'vue-i18n'
const conversationStore = useConversationStore()
const emitter = useEmitter()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()
const userStore = useUserStore()

const phoneNumber = computed(() => {
  const countryCodeValue = conversation.value?.contact?.phone_number_country_code || ''
  const number = conversation.value?.contact?.phone_number || t('conversation.sidebar.notAvailable')
  if (!countryCodeValue) return number

  // Lookup calling code
  const country = countries.find((c) => c.iso_2 === countryCodeValue)
  const callingCode = country ? country.calling_code : countryCodeValue
  return `${callingCode} ${number}`
})

const countryName = computed(() => {
  const code = conversation.value?.contact?.country
  if (!code) return ''
  const c = countries.find((c) => c.iso_2 === code)
  return c ? c.name : code
})

const isLivechat = computed(() => conversation.value?.inbox_channel === 'livechat')
const contactStatus = computed(() => conversation.value?.contact?.availability_status)
const isVerified = computed(() => isLivechat.value && conversation.value?.contact?.type !== 'visitor')

const parsedUA = computed(() => {
  const ua = conversation.value?.meta?.user_agent
  if (!ua) return ''
  const browser = ua.match(/(Chrome|Firefox|Safari|Edge|Opera|MSIE|Trident)[/\s](\d+)/i)
  const os = ua.match(/(Windows|Mac OS X|Linux|Android|iOS|iPhone|iPad)[\s/]?([0-9._]*)/i)
  const parts = []
  if (browser) parts.push(browser[1] + ' ' + browser[2])
  if (os) parts.push(os[1].replace('_', ' '))
  return parts.length > 0 ? parts.join(' / ') : ua.substring(0, 60)
})

</script>
