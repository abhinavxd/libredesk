<template>
  <div class="flex flex-col h-full min-h-0">
    <div v-if="messages.length" class="flex justify-end border-b border-border px-3 py-2">
      <Button variant="ghost" size="sm" class="gap-1.5 text-muted-foreground" @click="clearChat">
        <Eraser class="h-3.5 w-3.5" />
        {{ $t('copilot.clearChat') }}
      </Button>
    </div>
    <div ref="scrollRef" class="flex-1 overflow-y-auto p-4 space-y-4 min-h-0">
      <div
        v-if="messages.length === 0"
        class="h-full flex flex-col items-center justify-center gap-3 text-center px-4"
      >
        <div class="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center">
          <Bot class="h-6 w-6 text-primary" />
        </div>
        <p class="text-sm text-muted-foreground">
          {{ $t('copilot.emptyState', { name: appSettingsStore.copilotName }) }}
        </p>
        <div class="flex flex-col gap-2 w-full max-w-[85%]">
          <Button
            v-for="preset in presets"
            :key="preset"
            type="button"
            variant="outline"
            size="sm"
            class="w-full whitespace-normal h-auto py-1.5"
            @click="send(preset)"
          >
            {{ preset }}
          </Button>
        </div>
      </div>
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
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { SendHorizontal, Eraser, Bot } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import { useAppSettingsStore } from '@/stores/appSettings'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const conversationStore = useConversationStore()
const appSettingsStore = useAppSettingsStore()
const emitter = useEmitter()
const { t } = useI18n()

const presets = computed(() => [
  t('copilot.preset.summarize'),
  t('copilot.preset.customerAsking')
])

// Chat history lives in the store keyed by conversation uuid so it survives tab
// switches (this panel unmounts) and never leaks across conversations.
const messages = computed(() => conversationStore.getCopilotMessages(conversationStore.current?.uuid || ''))
const input = ref('')
const isThinking = ref(false)
const scrollRef = ref(null)

const clearChat = async () => {
  const uuid = conversationStore.current?.uuid || ''
  conversationStore.clearCopilotMessages(uuid)
  if (!uuid) return
  try {
    await api.clearCopilotMessages(uuid)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

// Load the persisted chat from the server when a conversation opens, so a refresh
// does not lose it. Skip if the store already has messages for it (a live session).
const hydrate = async (uuid) => {
  if (!uuid || conversationStore.getCopilotMessages(uuid).length > 0) return
  try {
    const resp = await api.getCopilotMessages(uuid)
    const loaded = (resp.data.data || []).map((m) => ({ role: m.role, content: m.content }))
    if (loaded.length) {
      conversationStore.setCopilotMessages(uuid, loaded)
      await scrollToBottom()
    }
  } catch {
    // Non-fatal: the panel still works without history.
  }
}

onMounted(() => hydrate(conversationStore.current?.uuid || ''))
watch(
  () => conversationStore.current?.uuid,
  (uuid) => hydrate(uuid || '')
)

const scrollToBottom = async () => {
  await nextTick()
  if (scrollRef.value) scrollRef.value.scrollTop = scrollRef.value.scrollHeight
}

const send = async (preset) => {
  const text = (typeof preset === 'string' ? preset : input.value).trim()
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
