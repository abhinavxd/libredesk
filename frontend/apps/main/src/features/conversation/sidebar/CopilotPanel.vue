<template>
  <div class="flex flex-col h-full min-h-0">
    <div v-if="messages.length" class="flex justify-end border-b border-border px-3 py-2">
      <Button variant="ghost" size="sm" class="gap-1.5 text-muted-foreground" @click="clearChat">
        <Eraser class="h-3.5 w-3.5" />
        {{ $t('copilot.clearChat') }}
      </Button>
    </div>
    <div ref="scrollRef" class="flex-1 overflow-y-auto p-4 space-y-4 min-h-0">
      <p v-if="messages.length === 0" class="text-sm text-muted-foreground">
        {{ $t('copilot.emptyState') }}
      </p>
      <div
        v-for="(msg, i) in messages"
        :key="i"
        class="flex"
        :class="msg.role === 'user' ? 'justify-end' : 'justify-start'"
      >
        <div
          class="rounded-lg px-3 py-2 text-sm max-w-[85%] whitespace-pre-wrap break-words"
          :class="
            msg.role === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted text-foreground'
          "
        >
          {{ msg.content }}
        </div>
      </div>
      <div v-if="isThinking" class="flex justify-start">
        <div class="rounded-lg px-3 py-2 text-sm bg-muted text-muted-foreground">
          {{ $t('copilot.thinking') }}
        </div>
      </div>
    </div>

    <form class="border-t border-border p-3 flex gap-2 items-end" @submit.prevent="send">
      <Textarea
        v-model="input"
        :placeholder="$t('copilot.placeholder')"
        rows="2"
        class="resize-none"
        @keydown.enter.exact.prevent="send"
      />
      <Button type="submit" size="icon" :disabled="isThinking || !input.trim()">
        <SendHorizontal class="h-4 w-4" />
      </Button>
    </form>
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { SendHorizontal, Eraser } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@/api'

const conversationStore = useConversationStore()
const emitter = useEmitter()

// Chat history lives in the store keyed by conversation uuid so it survives tab
// switches (this panel unmounts) and never leaks across conversations.
const messages = computed(() => conversationStore.getCopilotMessages(conversationStore.current?.uuid || ''))
const input = ref('')
const isThinking = ref(false)
const scrollRef = ref(null)

const clearChat = () => {
  conversationStore.clearCopilotMessages(conversationStore.current?.uuid || '')
}

const scrollToBottom = async () => {
  await nextTick()
  if (scrollRef.value) scrollRef.value.scrollTop = scrollRef.value.scrollHeight
}

const send = async () => {
  const text = input.value.trim()
  if (!text || isThinking.value) return

  const uuid = conversationStore.current?.uuid || ''
  conversationStore.setCopilotMessages(uuid, [...messages.value, { role: 'user', content: text }])
  input.value = ''
  isThinking.value = true
  await scrollToBottom()

  try {
    const resp = await api.aiCopilot({
      conversation_uuid: uuid,
      messages: conversationStore.getCopilotMessages(uuid)
    })
    conversationStore.setCopilotMessages(uuid, [
      ...conversationStore.getCopilotMessages(uuid),
      { role: 'assistant', content: resp.data.data || '' }
    ])
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isThinking.value = false
    await scrollToBottom()
  }
}
</script>
