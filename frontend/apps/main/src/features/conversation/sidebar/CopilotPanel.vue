<template>
  <div class="flex flex-col h-full min-h-0">
    <div
      v-if="assistants.length || messages.length"
      class="flex items-center justify-between gap-2 border-b border-border px-3 py-2"
    >
      <Select
        v-if="assistants.length"
        :model-value="selectedAssistantId"
        @update:model-value="persistAssistant"
      >
        <SelectTrigger class="h-8 w-auto gap-1.5 border-0 shadow-none text-muted-foreground focus:ring-0">
          <SelectValue :placeholder="appSettingsStore.copilotName" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem :value="0">{{ appSettingsStore.copilotName }}</SelectItem>
          <SelectItem v-for="a in assistants" :key="a.id" :value="a.id">{{ a.name }}</SelectItem>
        </SelectContent>
      </Select>
      <span v-else />
      <Button
        v-if="messages.length"
        type="button"
        variant="ghost"
        size="sm"
        class="gap-1.5 text-muted-foreground"
        @click="clearChat"
      >
        <Eraser class="h-3.5 w-3.5" />
        {{ $t('copilot.clearChat') }}
      </Button>
    </div>
    <div ref="scrollRef" class="flex-1 overflow-y-auto p-4 space-y-4 min-h-0">
      <div
        v-if="messages.length === 0"
        class="h-full flex flex-col items-center justify-center gap-4 text-center px-4"
      >
        <div class="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center">
          <Bot class="h-6 w-6 text-primary" />
        </div>
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">{{ appSettingsStore.copilotName }}</p>
          <p class="text-xs text-muted-foreground">
            {{ $t('copilot.emptyState', { name: appSettingsStore.copilotName }) }}
          </p>
        </div>
        <div class="flex flex-col gap-1.5 w-full max-w-[85%]">
          <Button
            v-for="preset in presets"
            :key="preset"
            type="button"
            variant="outline"
            size="sm"
            class="w-full justify-start whitespace-normal h-auto py-1.5 font-normal text-muted-foreground"
            @click="send(preset)"
          >
            {{ preset }}
          </Button>
        </div>
      </div>
      <div
        v-for="(msg, i) in messages"
        :key="i"
        class="flex gap-2"
        :class="msg.isUser ? 'justify-end' : 'justify-start'"
      >
        <div
          v-if="!msg.isUser"
          class="mt-0.5 h-6 w-6 shrink-0 rounded-full bg-primary/10 flex items-center justify-center"
        >
          <Bot class="h-3.5 w-3.5 text-primary" />
        </div>
        <div
          class="flex flex-col gap-1 max-w-[85%]"
          :class="msg.isUser ? 'items-end' : 'items-start'"
        >
          <div
            class="rounded-lg px-3 py-2 text-sm break-words"
            :class="
              msg.isUser
                ? 'bg-primary text-primary-foreground rounded-br-sm whitespace-pre-wrap'
                : 'bg-muted text-foreground rounded-bl-sm'
            "
          >
            <Letter v-if="!msg.isUser" :html="msg.content" class="native-html" />
            <template v-else>{{ msg.content }}</template>
          </div>
          <div v-if="!msg.isUser && msg.content" class="flex gap-0.5">
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="h-6 w-6 text-muted-foreground"
                  @click="copyAnswer(msg.content)"
                >
                  <Copy class="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('globals.terms.copy') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="h-6 w-6 text-muted-foreground"
                  @click="insertIntoReply(msg.content)"
                >
                  <Reply class="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('copilot.insertIntoReply') }}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="h-6 w-6 text-muted-foreground"
                  @click="addAsPrivateNote(msg.content)"
                >
                  <StickyNote class="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{{ $t('copilot.addAsPrivateNote') }}</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </div>
      <div v-if="isThinking" class="flex gap-2 justify-start">
        <div
          class="mt-0.5 h-6 w-6 shrink-0 rounded-full bg-primary/10 flex items-center justify-center"
        >
          <Bot class="h-3.5 w-3.5 text-primary" />
        </div>
        <div class="rounded-lg rounded-bl-sm px-3 py-2 bg-muted text-muted-foreground flex items-center">
          <DotLoader />
          <span class="sr-only">{{ $t('copilot.thinking') }}</span>
        </div>
      </div>
    </div>

    <form class="border-t border-border p-3" @submit.prevent="send">
      <div
        class="rounded-lg border border-input bg-background shadow-sm transition-colors focus-within:ring-1 focus-within:ring-ring"
      >
        <Textarea
          v-model="input"
          :placeholder="$t('copilot.placeholder')"
          rows="2"
          class="min-h-[44px] resize-none border-0 bg-transparent shadow-none focus-visible:ring-0"
          @keydown.enter.exact.prevent="send"
        />
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Letter } from 'vue-letter'
import { DotLoader } from '@shared-ui/components/ui/loader'
import { Eraser, Bot, Copy, Reply, StickyNote } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import { useCopilotStore } from '@/stores/copilot'
import { useAppSettingsStore } from '@/stores/appSettings'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { getTextFromHTML } from '@shared-ui/utils/string.js'
import { UserTypeAgent } from '@/constants/user'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const conversationStore = useConversationStore()
const copilotStore = useCopilotStore()
const appSettingsStore = useAppSettingsStore()
const emitter = useEmitter()
const { t } = useI18n()

const presets = computed(() => [
  t('copilot.preset.summarize'),
  t('copilot.preset.customerAsking')
])

// Chat history lives in the store keyed by conversation uuid so it survives tab
// switches (this panel unmounts) and never leaks across conversations.
const messages = computed(() =>
  copilotStore
    .getMessages(conversationStore.current?.uuid || '')
    .map((m) => ({ ...m, isUser: m.role === 'user' }))
)
const input = ref('')
// Thinking state is per conversation so an in-flight send for one conversation does not show or
// block the panel after the agent switches to another.
const thinkingByUUID = ref({})
const isThinking = computed(() => !!thinkingByUUID.value[conversationStore.current?.uuid || ''])
const scrollRef = ref(null)

// Persona selection is global per agent (a stored assistant whose instructions Copilot borrows for
// tone), not per conversation. 0 means the default Copilot.
const ASSISTANT_STORAGE_KEY = 'copilot_assistant_id'
const assistants = ref([])
const selectedAssistantId = ref(0)

const persistAssistant = (value) => {
  const id = Number(value) || 0
  selectedAssistantId.value = id
  localStorage.setItem(ASSISTANT_STORAGE_KEY, String(id))
}

const loadAssistants = async () => {
  const stored = parseInt(localStorage.getItem(ASSISTANT_STORAGE_KEY) || '0', 10)
  if (!Number.isNaN(stored)) selectedAssistantId.value = stored
  try {
    const resp = await api.getAIAssistantsCompact()
    assistants.value = resp.data.data || []
    if (
      selectedAssistantId.value &&
      !assistants.value.some((a) => a.id === selectedAssistantId.value)
    ) {
      persistAssistant(0)
    }
  } catch {
    assistants.value = []
  }
}

// Per-conversation operation revision: a clear bumps it so in-flight hydrate/send
// responses for the old state are discarded instead of restoring cleared history.
const revisions = {}
const revision = (uuid) => revisions[uuid] || 0

const clearChat = async () => {
  const uuid = conversationStore.current?.uuid || ''
  revisions[uuid] = revision(uuid) + 1
  copilotStore.clearMessages(uuid)
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
  if (!uuid || copilotStore.getMessages(uuid).length > 0) return
  const rev = revision(uuid)
  try {
    const resp = await api.getCopilotMessages(uuid)
    if (rev !== revision(uuid) || copilotStore.getMessages(uuid).length > 0) return
    const loaded = (resp.data.data || []).map((m) => ({ role: m.role, content: m.content }))
    if (loaded.length) {
      copilotStore.setMessages(uuid, loaded)
      await scrollToBottom()
    }
  } catch {
    // Non-fatal: the panel still works without history.
  }
}

onMounted(() => {
  hydrate(conversationStore.current?.uuid || '')
  loadAssistants()
})
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
  if (!uuid) return
  const rev = revision(uuid)
  copilotStore.setMessages(uuid, [...copilotStore.getMessages(uuid), { role: 'user', content: text }])
  input.value = ''
  thinkingByUUID.value[uuid] = true
  await scrollToBottom()

  try {
    const payload = { conversation_uuid: uuid, message: text }
    if (selectedAssistantId.value > 0) payload.assistant_id = selectedAssistantId.value
    const resp = await api.aiCopilot(payload)
    if (rev !== revision(uuid)) return
    copilotStore.setMessages(uuid, [
      ...copilotStore.getMessages(uuid),
      { role: 'assistant', content: resp.data.data || '' }
    ])
  } catch (error) {
    // A rejected persona (deleted or disabled since selection) comes back as an input error; fall back
    // to the default Copilot so the next send works.
    if (error?.response?.data?.error?.type === 'InputException' && selectedAssistantId.value > 0) {
      persistAssistant(0)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    delete thinkingByUUID.value[uuid]
    await scrollToBottom()
  }
}

const copyAnswer = async (content) => {
  try {
    await navigator.clipboard.writeText(getTextFromHTML(content))
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.copied') })
  } catch {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('copilot.copyFailed')
    })
  }
}

const insertIntoReply = (content) => {
  emitter.emit(EMITTER_EVENTS.COPILOT_INSERT_REPLY, content)
}

const addAsPrivateNote = async (content) => {
  const uuid = conversationStore.current?.uuid || ''
  if (!uuid) return
  const rev = revision(uuid)
  try {
    await api.sendMessage(uuid, {
      sender_type: UserTypeAgent,
      private: true,
      message: content,
      attachments: [],
      mentions: [],
      cc: [],
      bcc: [],
      to: [],
      echo_id: ''
    })
    if (rev !== revision(uuid)) return
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('copilot.noteAdded') })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
