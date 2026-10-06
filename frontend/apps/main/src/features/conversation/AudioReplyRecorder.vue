<template>
  <Popover
    :open="active"
    @update:open="
      (open) => {
        if (!open) cancel()
      }
    "
  >
    <Tooltip>
      <TooltipTrigger as-child>
        <PopoverTrigger as-child>
          <Button
            type="button"
            variant="outline"
            size="icon"
            class="h-9 w-auto border-border/70 bg-background px-2 text-muted-foreground shadow-none max-md:min-h-11 max-md:min-w-11"
            :aria-label="$t('globals.messages.recordAudio')"
            :disabled="active"
            @click="start"
          >
            <Mic aria-hidden="true" class="size-4" />
          </Button>
        </PopoverTrigger>
      </TooltipTrigger>
      <TooltipContent>{{ $t('globals.messages.recordAudio') }}</TooltipContent>
    </Tooltip>
    <PopoverContent
      side="top"
      align="start"
      class="w-72 max-w-[calc(100vw-3rem)] space-y-3"
      :aria-label="$t('globals.messages.recordAudio')"
      @interact-outside.prevent
    >
      <div class="flex items-center justify-between gap-3 text-sm">
        <p role="status" class="font-medium">{{ $t(statusKey) }}</p>
        <span
          v-if="recording"
          class="flex items-center gap-2 font-mono tabular-nums text-muted-foreground"
        >
          <span aria-hidden="true" class="size-2 rounded-full bg-destructive" />{{ duration }}
        </span>
      </div>
      <audio
        v-if="preview"
        :src="preview"
        controls
        class="w-full"
        :aria-label="$t('globals.messages.recordedAudio')"
      />
      <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
      <div class="flex flex-wrap justify-end gap-2">
        <Button type="button" variant="ghost" size="sm" @click="cancel">{{
          $t('globals.messages.cancel')
        }}</Button>
        <Button v-if="recording" type="button" variant="outline" size="sm" @click="stop">{{
          $t('globals.messages.stopRecording')
        }}</Button>
        <Button v-if="file" type="button" size="sm" @click="attach">{{
          $t('globals.messages.attachRecording')
        }}</Button>
      </div>
    </PopoverContent>
  </Popover>
</template>
<script setup>
import { computed, ref, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { Mic } from 'lucide-vue-next'
import { Popover, PopoverTrigger, PopoverContent } from '@shared-ui/components/ui/popover'
import { Tooltip, TooltipTrigger, TooltipContent } from '@shared-ui/components/ui/tooltip'
import { Button } from '@shared-ui/components/ui/button'
import { encodeAudioReply } from './audioReply'
const emit = defineEmits(['recorded', 'busy'])
const { t } = useI18n()
const active = ref(false)
const recording = ref(false)
const preview = ref('')
const error = ref('')
const file = ref(null)
const seconds = ref(0)
const duration = computed(
  () => `${Math.floor(seconds.value / 60)}:${String(seconds.value % 60).padStart(2, '0')}`
)
const statusKey = computed(() =>
  recording.value
    ? 'globals.messages.recordingAudio'
    : file.value
      ? 'globals.messages.recordedAudio'
      : error.value
        ? 'globals.messages.recordingFailed'
        : 'globals.messages.preparingRecording'
)
let recorder
let stream
let timer
let generation = 0
const release = () => {
  clearInterval(timer)
  stream?.getTracks().forEach((track) => track.stop())
  stream = null
}
const cancel = () => {
  generation++
  if (recorder?.state === 'recording') recorder.stop()
  recorder = null
  release()
  if (preview.value) URL.revokeObjectURL(preview.value)
  preview.value = ''
  file.value = null
  error.value = ''
  active.value = false
  recording.value = false
  emit('busy', false)
}
const start = async () => {
  if (active.value) return
  active.value = true
  emit('busy', true)
  const run = ++generation
  try {
    const acquired = await navigator.mediaDevices.getUserMedia({ audio: true })
    if (run !== generation) {
      acquired.getTracks().forEach((track) => track.stop())
      return
    }
    stream = acquired
    const mimeType = ['audio/webm;codecs=opus', 'audio/ogg;codecs=opus', 'audio/mp4'].find((type) =>
      MediaRecorder.isTypeSupported(type)
    )
    recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined)
    const chunks = []
    const type = recorder.mimeType
    recorder.ondataavailable = (event) => {
      if (event.data.size) chunks.push(event.data)
    }
    recorder.onstop = async () => {
      if (run !== generation) return
      release()
      recording.value = false
      try {
        const encoded = await encodeAudioReply(new Blob(chunks, { type }))
        if (run !== generation) return
        if (!encoded.size) throw new Error('empty recording')
        file.value = encoded
        preview.value = URL.createObjectURL(encoded)
      } catch {
        if (run === generation) error.value = t('replyBox.audioEncodingFailed')
      }
    }
    recorder.onerror = () => {
      if (run !== generation) return
      cancel()
      active.value = true
      error.value = t('replyBox.audioRecordingFailed')
    }
    seconds.value = 0
    recorder.start()
    recording.value = true
    timer = setInterval(() => {
      seconds.value++
    }, 1000)
  } catch {
    if (run !== generation) return
    release()
    error.value = t('replyBox.microphoneUnavailable')
  }
}
const stop = () => {
  if (recorder?.state === 'recording') recorder.stop()
}
const attach = () => {
  emit('recorded', [file.value])
  cancel()
}
onBeforeUnmount(cancel)
</script>
