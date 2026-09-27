<template>
  <div class="flex flex-col items-center gap-1">
    <div class="relative size-36">
      <div ref="container" role="img" :aria-label="$t('globals.terms.sticker')" class="size-full" />
      <Spinner v-if="!animation && !failed" size="sm" :aria-label="$t('globals.terms.loading')" />
    </div>
    <p v-if="failed" role="status" class="max-w-48 text-xs text-muted-foreground">
      {{ $t('conversation.telegram.stickerUnavailable') }}
    </p>
    <div class="flex items-center gap-1">
      <Button
        v-if="animation"
        type="button"
        variant="ghost"
        size="icon"
        class="text-muted-foreground max-md:min-h-11 max-md:min-w-11"
        :aria-label="
          playing ? $t('globals.messages.pauseAnimation') : $t('globals.messages.playAnimation')
        "
        @click="toggle"
      >
        <Pause v-if="playing" class="size-4" aria-hidden="true" /><Play
          v-else
          class="size-4"
          aria-hidden="true"
        />
      </Button>
      <DownloadLink
        :url="attachment.url"
        class="size-9 max-md:size-11 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
    </div>
  </div>
</template>
<script setup>
import { ref, shallowRef, watch } from 'vue'
import { Pause, Play } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import DownloadLink from '@main/components/DownloadLink.vue'
import { downloadUrl } from '@shared-ui/utils/file'
import { decodeTelegramSticker } from './telegramSticker'
const props = defineProps({ attachment: { type: Object, required: true } })
const container = ref(null)
const animation = shallowRef(null)
const failed = ref(false)
const playing = ref(false)
const toggle = () => {
  playing.value = !playing.value
  if (playing.value) animation.value.play()
  else animation.value.pause()
}
watch(
  [() => props.attachment.url, container],
  async ([url, element], _, onCleanup) => {
    if (!element) return
    const abort = new AbortController()
    let player
    onCleanup(() => {
      abort.abort()
      player?.destroy()
      animation.value = null
    })
    failed.value = false
    try {
      const response = await fetch(downloadUrl(url).replace('?download=1', ''), {
        signal: abort.signal
      })
      if (!response.ok) throw new Error('Sticker download failed')
      const data = await decodeTelegramSticker(await response.arrayBuffer())
      const { default: lottie } = await import('lottie-web/build/player/lottie_light_canvas')
      if (abort.signal.aborted) return
      playing.value = !window.matchMedia('(prefers-reduced-motion: reduce)').matches
      player = lottie.loadAnimation({
        container: element,
        renderer: 'canvas',
        loop: true,
        autoplay: playing.value,
        animationData: data
      })
      animation.value = player
    } catch {
      if (!abort.signal.aborted) failed.value = true
    }
  },
  { flush: 'post' }
)
</script>
