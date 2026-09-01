<template>
  <div class="widget-home-header relative" :style="headerStyle">
    <div class="px-5 pt-7 pb-2">
      <!-- Logo -->
      <img
        v-if="config.logo_url"
        :src="config.logo_url"
        :alt="config.brand_name"
        class="max-h-8 max-w-full"
      />
      <!-- Greeting and introduction -->
      <div class="mt-7" :class="textColorClass">
        <h2 class="widget-home-header__greeting break-words">{{ parsedGreeting }}</h2>
        <p class="widget-home-header__intro" :class="subTextColorClass">
          {{ parsedIntroduction }}
        </p>
      </div>
    </div>
    <!-- Primary action area sits on the header so it doesn't cut off visually. -->
    <div class="relative z-10 px-4 pb-4">
      <slot />
    </div>
    <!-- Fade overlay: masks a custom header background into the page surface. -->
    <div
      v-if="showFade"
      class="absolute bottom-0 left-0 right-0 h-16 pointer-events-none"
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

const parseHex = (hex) => {
  if (!hex || typeof hex !== 'string') return null
  let h = hex.replace('#', '')
  if (h.length === 3) h = h.split('').map((c) => c + c).join('')
  if (!/^[0-9a-f]{6}$/i.test(h)) return null
  return {
    r: parseInt(h.slice(0, 2), 16),
    g: parseInt(h.slice(2, 4), 16),
    b: parseInt(h.slice(4, 6), 16)
  }
}

const isHexDark = (hex) => {
  const rgb = parseHex(hex)
  if (!rgb) return false
  const L = (0.2126 * rgb.r + 0.7152 * rgb.g + 0.0722 * rgb.b) / 255
  return L < 0.55
}

const headerIsDark = computed(() => {
  const bg = props.config.home_screen?.background
  if (!bg?.type) return false
  switch (bg.type) {
    case 'solid':
      return isHexDark(bg.color)
    case 'gradient':
      return isHexDark(bg.gradient_start) || isHexDark(bg.gradient_end)
    case 'image':
      return Boolean(bg.image_url)
    default:
      return false
  }
})

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
const isDarkMode = computed(() => Boolean(props.config.dark_mode))

// White-on-white happens when dark mode is turned off but header text stays
// white and the home screen has no dark background. Fall back to body text.
const useDarkText = computed(() => {
  if (headerTextColor.value === 'black') return true
  if (headerTextColor.value === 'white') {
    return !isDarkMode.value && !headerIsDark.value
  }
  return !isDarkMode.value
})

const textColorClass = computed(() => (useDarkText.value ? 'text-foreground' : 'text-white'))

const subTextColorClass = computed(() =>
  useDarkText.value ? 'text-muted-foreground' : 'text-white/70'
)

const showFade = computed(
  () =>
    Boolean(props.config.home_screen?.background?.type) &&
    Boolean(props.config.home_screen?.fade_background)
)

const fadeStyle = { background: 'linear-gradient(to bottom, transparent, hsl(var(--background)))' }
</script>
