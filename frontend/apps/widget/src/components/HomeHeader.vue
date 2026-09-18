<template>
  <div class="relative" :style="headerStyle">
    <div class="p-7">
      <img
        v-if="config.logo_url"
        :src="config.logo_url"
        :alt="config.brand_name"
        class="max-h-7 max-w-full"
      />
      <div class="mt-20" :class="textColorClass">
        <h2 class="text-4xl font-bold leading-none tracking-tight break-words">
          {{ parsedGreeting }}
        </h2>
        <p class="mt-3 max-w-80 text-2xl font-medium leading-tight" :class="subTextColorClass">
          {{ parsedIntroduction }}
        </p>
      </div>
    </div>
    <div class="relative z-10 px-4 pb-5">
      <slot />
    </div>
    <div
      v-if="config.home_screen?.background?.type"
      class="absolute bottom-0 left-0 right-0 h-20 pointer-events-none"
      :style="fadeStyle"
    ></div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUserStore } from '@widget/store/user.js'
import { renderTemplate } from '@shared-ui/utils/string.js'

const props = defineProps({
  config: {
    type: Object,
    required: true
  }
})

const userStore = useUserStore()

const userData = computed(() => ({
  firstName: userStore.firstName,
  lastName: userStore.lastName
}))

const parsedGreeting = computed(() => renderTemplate(props.config.greeting_message, userData.value))

const parsedIntroduction = computed(() =>
  renderTemplate(props.config.introduction_message, userData.value)
)

const headerStyle = computed(() => {
  const hs = props.config.home_screen
  if (!hs?.background?.type) return {}

  const style = {}
  switch (hs.background.type) {
    case 'solid':
      if (hs.background.color) style.backgroundColor = hs.background.color
      break
    case 'gradient':
      if (hs.background.gradient_start && hs.background.gradient_end) {
        style.background = `linear-gradient(to bottom, ${hs.background.gradient_start}, ${hs.background.gradient_end})`
      }
      break
    case 'image':
      if (hs.background.image_url) {
        style.backgroundImage = `url(${hs.background.image_url})`
        style.backgroundSize = 'cover'
        style.backgroundPosition = 'center'
      }
      break
  }
  return style
})

const headerTextColor = computed(() => props.config.home_screen?.header_text_color)

const textColorClass = computed(() => {
  if (headerTextColor.value === 'black') return 'text-black'
  if (headerTextColor.value === 'white') return 'text-white'
  return ''
})

const subTextColorClass = computed(() => {
  if (headerTextColor.value === 'black') return 'text-black/70'
  if (headerTextColor.value === 'white') return 'text-white/70'
  return 'text-muted-foreground'
})

const fadeStyle = {
  background:
    'linear-gradient(to bottom, transparent 0%, hsl(var(--background) / 0.08) 20%, hsl(var(--background) / 0.32) 45%, hsl(var(--background) / 0.72) 72%, hsl(var(--background)) 100%)'
}
</script>
