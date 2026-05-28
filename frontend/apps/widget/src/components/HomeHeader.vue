<template>
  <div class="relative" :style="headerStyle">
    <div class="p-8">
      <!-- Logo -->
      <img
        v-if="config.logo_url"
        :src="config.logo_url"
        :alt="config.brand_name"
        class="max-h-8 max-w-full"
      />
      <!-- Greeting and introduction -->
      <div class="mt-24" :class="textColorClass">
        <h2 class="break-all font-bold" :class="widgetStore.isRtl ? 'text-2xl' : 'text-4xl'">{{ parsedGreeting }}</h2>
        <p class="mt-2 font-semibold" :class="[subTextColorClass, widgetStore.isRtl ? 'text-sm' : 'text-lg']">
          {{ parsedIntroduction }}
        </p>
      </div>
    </div>
    <!-- Primary action area sits on the gradient so it doesn't cut off visually. -->
    <div class="relative z-10 px-4 pb-4">
      <slot />
    </div>
    <!-- Fade overlay: masks the gradient's bottom into bg-background for a seamless transition. -->
    <div
      v-if="config.home_screen?.background?.type"
      class="absolute bottom-0 left-0 right-0 h-16 pointer-events-none"
      :style="fadeStyle"
    ></div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUserStore } from '@widget/store/user.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { renderTemplate } from '@shared-ui/utils/string.js'

const widgetStore = useWidgetStore()

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

const fadeStyle = { background: 'linear-gradient(to bottom, transparent, hsl(var(--background)))' }
</script>
