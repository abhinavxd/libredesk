<template>
  <div>
    <div
      v-if="!windowOpen"
      class="mx-2 mt-2 box p-3 flex items-center justify-between gap-3 border-destructive/40 bg-destructive/5"
    >
      <div class="flex items-start gap-2 text-sm">
        <TriangleAlert class="size-4 mt-0.5 text-destructive shrink-0" />
        <div>
          <p class="font-medium">{{ $t('conversation.whatsapp.windowClosed.title') }}</p>
          <p class="text-muted-foreground text-xs">
            {{ $t('conversation.whatsapp.windowClosed.description') }}
          </p>
        </div>
      </div>
      <Button size="sm" variant="outline" @click="openTemplatePicker">
        {{ $t('conversation.whatsapp.sendTemplate') }}
      </Button>
    </div>
    <div
      v-else-if="windowClosingSoon"
      class="mx-2 mt-2 box px-3 py-2 flex items-center gap-2 text-sm border-border"
    >
      <Lightbulb class="size-4 text-muted-foreground shrink-0" />
      <span>{{ $t('conversation.whatsapp.windowClosing', { time: windowExpiresIn }) }}</span>
    </div>

    <ReplyBox />

    <Dialog v-model:open="pickerOpen">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{{ $t('conversation.whatsapp.sendTemplate') }}</DialogTitle>
          <DialogDescription>
            {{ $t('conversation.whatsapp.sendTemplate.description') }}
          </DialogDescription>
        </DialogHeader>

        <div v-if="isFetchingTemplates" class="py-6 text-sm text-muted-foreground">
          {{ $t('globals.messages.loading') }}
        </div>
        <div v-else-if="!approvedTemplates.length" class="py-6 text-sm text-muted-foreground">
          {{ $t('conversation.whatsapp.noApprovedTemplates') }}
        </div>

        <div v-else-if="!selectedTemplate" class="space-y-2 max-h-80 overflow-y-auto">
          <button
            v-for="tmpl in approvedTemplates"
            :key="tmpl.id"
            type="button"
            class="w-full text-left box p-3 hover:bg-accent transition-colors"
            @click="pickTemplate(tmpl)"
          >
            <div class="flex items-center justify-between gap-2">
              <div class="font-mono text-sm">{{ tmpl.name }}</div>
              <Badge variant="outline">{{ tmpl.language }}</Badge>
            </div>
            <div class="text-xs text-muted-foreground mt-1 line-clamp-2">
              {{ tmpl.body_content }}
            </div>
          </button>
        </div>

        <div v-else class="space-y-4">
          <div class="box p-3">
            <div class="flex items-center justify-between gap-2 mb-2">
              <div class="font-mono text-sm">{{ selectedTemplate.name }}</div>
              <Button variant="ghost" size="sm" @click="selectedTemplate = null">
                {{ $t('globals.messages.back') }}
              </Button>
            </div>
            <div class="text-sm whitespace-pre-wrap text-muted-foreground">
              {{ renderedPreview }}
            </div>
          </div>

          <div
            v-for="key in placeholders"
            :key="key"
            class="grid grid-cols-3 gap-3 items-center"
          >
            <label class="text-sm font-mono">{{ placeholderLabel(key) }}</label>
            <Input v-model="templateParams[key]" class="col-span-2" />
          </div>

          <div
            v-for="btn in urlButtonParams"
            :key="btn.key"
            class="grid grid-cols-3 gap-3 items-center"
          >
            <label class="text-sm truncate" :title="btn.url">{{ btn.label }}</label>
            <Input v-model="templateParams[btn.key]" class="col-span-2" :placeholder="btn.url" />
          </div>
        </div>

        <DialogFooter v-if="selectedTemplate">
          <Button variant="outline" @click="pickerOpen = false" :disabled="isSending">
            {{ $t('globals.messages.cancel') }}
          </Button>
          <Button :is-loading="isSending" :disabled="isSending" @click="sendTemplate">
            {{ $t('conversation.whatsapp.send') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Lightbulb, TriangleAlert } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Input } from '@shared-ui/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import ReplyBox from './ReplyBox.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

const WINDOW_MS = 24 * 60 * 60 * 1000
const CLOSING_SOON_MS = 4 * 60 * 60 * 1000
const PLACEHOLDER_PATTERN = /\{\{([A-Za-z0-9_]+)\}\}/g

const { t } = useI18n()
const conversationStore = useConversationStore()
const emitter = useEmitter()

const nowTick = ref(Date.now())
let tickerHandle = setInterval(() => {
  nowTick.value = Date.now()
}, 60_000)
onBeforeUnmount(() => {
  if (tickerHandle) {
    clearInterval(tickerHandle)
    tickerHandle = null
  }
})

const lastInboundAt = computed(() => conversationStore.current?.last_inbound_at)

const windowRemainingMs = computed(() => {
  const ts = lastInboundAt.value
  if (!ts) return 0
  return WINDOW_MS - (nowTick.value - new Date(ts).getTime())
})

const windowOpen = computed(() => windowRemainingMs.value > 0)

const windowClosingSoon = computed(
  () => windowOpen.value && windowRemainingMs.value <= CLOSING_SOON_MS
)

const windowExpiresIn = computed(() => {
  const remainingMs = windowRemainingMs.value
  if (remainingMs <= 0) return ''
  const hours = Math.floor(remainingMs / (60 * 60 * 1000))
  const minutes = Math.floor((remainingMs % (60 * 60 * 1000)) / (60 * 1000))
  return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`
})

const pickerOpen = ref(false)
const isFetchingTemplates = ref(false)
const templates = ref([])
const selectedTemplate = ref(null)
const templateParams = reactive({})
const isSending = ref(false)

watch(
  () => conversationStore.current?.uuid,
  () => {
    pickerOpen.value = false
    selectedTemplate.value = null
  }
)

// Media-header templates need a media ID the dashboard can't supply, AUTHENTICATION templates need OTP button params, and libredesk_csat_* names are reserved for surveys.
const SENDABLE_HEADER_TYPES = ['', 'NONE', 'TEXT']
const RESERVED_NAME_PREFIX = 'libredesk_csat_'

const approvedTemplates = computed(() =>
  templates.value.filter(
    (tmpl) =>
      tmpl.status === 'APPROVED' &&
      tmpl.category !== 'AUTHENTICATION' &&
      !tmpl.name.startsWith(RESERVED_NAME_PREFIX) &&
      SENDABLE_HEADER_TYPES.includes(tmpl.header_type || '')
  )
)

const extractPlaceholders = (sources) => {
  const seen = new Set()
  const out = []
  for (const src of sources) {
    if (!src) continue
    for (const match of src.matchAll(PLACEHOLDER_PATTERN)) {
      const key = match[1]
      if (!seen.has(key)) {
        seen.add(key)
        out.push(key)
      }
    }
  }
  return out
}

const placeholders = computed(() => {
  if (!selectedTemplate.value) return []
  const sources = [selectedTemplate.value.body_content]
  if (selectedTemplate.value.header_type === 'TEXT' && selectedTemplate.value.header_content) {
    sources.push(selectedTemplate.value.header_content)
  }
  return extractPlaceholders(sources)
})

const placeholderLabel = (key) => `{{${key}}}`

// URL button placeholders are keyed button_url_<index> as the backend component builder expects.
const urlButtonParams = computed(() => {
  const buttons = selectedTemplate.value?.buttons || []
  const out = []
  buttons.forEach((btn, idx) => {
    if ((btn.type || '').toUpperCase() !== 'URL') return
    if (!(btn.url || '').match(PLACEHOLDER_PATTERN)) return
    out.push({ key: `button_url_${idx}`, label: btn.text || btn.url, url: btn.url })
  })
  return out
})

const allParamKeys = computed(() => [
  ...placeholders.value,
  ...urlButtonParams.value.map((btn) => btn.key)
])

const renderedPreview = computed(() => {
  if (!selectedTemplate.value) return ''
  let body = selectedTemplate.value.body_content || ''
  for (const [k, v] of Object.entries(templateParams)) {
    if (v) body = body.split(`{{${k}}}`).join(v)
  }
  return body
})

const openTemplatePicker = async () => {
  pickerOpen.value = true
  selectedTemplate.value = null
  templates.value = []
  Object.keys(templateParams).forEach((k) => delete templateParams[k])

  const inboxID = conversationStore.current?.inbox_id
  if (!inboxID) return

  try {
    isFetchingTemplates.value = true
    const resp = await api.getWhatsAppTemplates(inboxID)
    templates.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isFetchingTemplates.value = false
  }
}

emitter.on(EMITTER_EVENTS.WHATSAPP_TEMPLATE_PICKER_OPEN, openTemplatePicker)
onBeforeUnmount(() => {
  emitter.off(EMITTER_EVENTS.WHATSAPP_TEMPLATE_PICKER_OPEN, openTemplatePicker)
})

const pickTemplate = (tmpl) => {
  selectedTemplate.value = tmpl
  for (const key of allParamKeys.value) {
    if (templateParams[key] === undefined) templateParams[key] = ''
  }
}

watch(selectedTemplate, () => {
  const valid = new Set(allParamKeys.value)
  for (const k of Object.keys(templateParams)) {
    if (!valid.has(k)) delete templateParams[k]
  }
})

const sendTemplate = async () => {
  if (!selectedTemplate.value) return
  const convUUID = conversationStore.current?.uuid
  if (!convUUID) return

  try {
    isSending.value = true
    await api.sendMessage(convUUID, {
      sender_type: 'agent',
      private: false,
      message: '',
      whatsapp_template_id: selectedTemplate.value.id,
      whatsapp_template_params: { ...templateParams }
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('conversation.whatsapp.templateSent')
    })
    pickerOpen.value = false
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSending.value = false
  }
}
</script>
