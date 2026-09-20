<template>
  <div class="px-4 py-3 text-sm">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="flex min-w-0 flex-[1_1_16rem] items-start gap-2">
        <ImageOff class="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        <div class="min-w-0">
          <p class="font-medium">
            {{ t(message.display?.blocked_images > 0 ? 'conversation.emailImagesBlocked' : 'conversation.senderImagesAllowed') }}
          </p>
          <p v-if="message.display?.sender || showTimestamp" class="mt-1 text-xs text-muted-foreground break-words">
            {{ message.display?.sender }}
            <span v-if="showTimestamp && message.created_at" class="ml-2">{{ formatMessageTimestamp(message.created_at) }}</span>
          </p>
          <p v-if="blockedDomains" class="mt-1 text-xs text-muted-foreground break-words">{{ blockedDomains }}</p>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <template v-if="message.display?.can_allow && message.display?.blocked_images > 0">
          <Button size="sm" variant="outline" :disabled="pending" @click="allow('message')">{{ t('conversation.allowImages') }}</Button>
          <Button v-if="message.display.sender && !message.display.sender_trusted" size="sm" variant="outline" :disabled="pending" :title="message.display.sender" @click="allow('sender')">{{ t('conversation.allowSenderImages') }}</Button>
        </template>
        <Button v-if="message.display?.sender_trusted" size="sm" variant="ghost" :disabled="pending" @click="allow('revoke-sender')">{{ t('conversation.revokeSenderImages') }}</Button>
      </div>
    </div>
    <p v-if="error" role="alert" class="mt-2 text-xs text-destructive">{{ error }}</p>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ImageOff } from 'lucide-vue-next'
import { formatMessageTimestamp } from '@shared-ui/utils/datetime.js'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import api from '@main/api'

const props = defineProps({
  message: { type: Object, required: true },
  showTimestamp: { type: Boolean, default: false }
})
const emit = defineEmits(['updated'])
const { t } = useI18n()
const pending = ref(false)
const error = ref('')
const blockedDomains = computed(() =>
  (props.message.display?.blocked_domains || []).filter(domain => typeof domain === 'string').join(', ')
)

const allow = async (scope) => {
  if (pending.value) return
  pending.value = true
  error.value = ''
  try {
    const response = await api.allowMessageImages(props.message.conversation_uuid, props.message.uuid, scope)
    emit('updated', response.data.data, scope)
  } catch {
    error.value = t('conversation.imagePermissionFailed')
  } finally {
    pending.value = false
  }
}
</script>
