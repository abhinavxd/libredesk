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

        <WhatsAppTemplatePicker
          :approved-templates="approvedTemplates"
          :selected-template="selectedTemplate"
          :template-params="templateParams"
          :placeholders="placeholders"
          :url-button-params="urlButtonParams"
          :rendered-preview="renderedPreview"
          :is-fetching="isFetchingTemplates"
          @pick="pickTemplate"
          @back="selectedTemplate = null"
          @update:param="(key, v) => (templateParams[key] = v)"
        />

        <DialogFooter v-if="selectedTemplate">
          <Button variant="outline" @click="pickerOpen = false" :disabled="isSending">
            {{ $t('globals.messages.cancel') }}
          </Button>
          <Button :is-loading="isSending" :disabled="isSending || !allParamsFilled" @click="sendTemplate">
            {{ $t('conversation.whatsapp.send') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Lightbulb, TriangleAlert } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import ReplyBox from './ReplyBox.vue'
import WhatsAppTemplatePicker from './WhatsAppTemplatePicker.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
import { WHATSAPP_WINDOW_MS as WINDOW_MS, whatsAppWindowInboundAt } from './whatsappTemplate.js'
import { useWhatsAppTemplatePicker } from './useWhatsAppTemplatePicker.js'

const CLOSING_SOON_MS = 4 * 60 * 60 * 1000

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

const lastInboundAt = computed(() => whatsAppWindowInboundAt(conversationStore.current))

const windowRemainingMs = computed(() => {
  const ts = lastInboundAt.value
  if (!ts) return 0
  return WINDOW_MS - (nowTick.value - ts)
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

const {
  templates,
  selectedTemplate,
  templateParams,
  approvedTemplates,
  placeholders,
  urlButtonParams,
  allParamsFilled,
  renderedPreview,
  pickTemplate,
  reset
} = useWhatsAppTemplatePicker()

const pickerOpen = ref(false)
const isFetchingTemplates = ref(false)
const isSending = ref(false)

watch(
  () => conversationStore.current?.uuid,
  () => {
    pickerOpen.value = false
    selectedTemplate.value = null
  }
)

const openTemplatePicker = async () => {
  pickerOpen.value = true
  reset()

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
