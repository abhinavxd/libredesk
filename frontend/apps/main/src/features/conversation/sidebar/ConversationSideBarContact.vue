<template>
  <div class="space-y-2">
    <div class="flex justify-between items-start">
      <Avatar class="size-20">
        <AvatarImage :src="conversation?.contact?.avatar_url || ''" />
        <AvatarFallback>
          {{ conversation?.contact?.first_name?.toUpperCase().substring(0, 2) }}
        </AvatarFallback>
      </Avatar>
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
      <span v-if="conversationStore.conversation.loading">
        <Skeleton class="w-32 h-4" />
      </span>
      <span v-else-if="conversation?.contact?.email"  class="break-all">
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
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ViewVerticalIcon } from '@radix-icons/vue'
import { Button } from '@shared-ui/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Mail, Phone, ExternalLink, IdCard } from 'lucide-vue-next'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useConversationStore } from '../../../stores/conversation'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { useUserStore } from '../../../stores/user'
import { useI18n } from 'vue-i18n'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const conversation = computed(() => conversationStore.current)
const { t } = useI18n()
const userStore = useUserStore()

const phoneNumber = computed(() => {
  const callingCode = conversation.value?.contact?.phone_number_calling_code || ''
  const number = conversation.value?.contact?.phone_number || t('conversation.sidebar.notAvailable')
  return callingCode ? `${callingCode} ${number}` : number
})
</script>
