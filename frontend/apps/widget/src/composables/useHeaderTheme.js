import { computed } from 'vue'

export function useHeaderTheme (config) {
  const homeScreen = computed(() => config.value?.home_screen)
  const choice = computed(() => homeScreen.value?.header_text_color)

  const textColorClass = computed(() => {
    if (choice.value === 'black') return 'text-black'
    if (choice.value === 'white') return 'text-white'
    return ''
  })

  const subTextColorClass = computed(() => {
    if (choice.value === 'black') return 'text-black/70'
    if (choice.value === 'white') return 'text-white/70'
    return 'text-muted-foreground'
  })

  const backgroundStyle = computed(() => {
    const background = homeScreen.value?.background
    if (!background?.type) return {}
    if (background.type === 'solid' && background.color) {
      return { backgroundColor: background.color }
    }
    if (background.type === 'gradient' && background.gradient_start && background.gradient_end) {
      return {
        background: `linear-gradient(to bottom, ${background.gradient_start}, ${background.gradient_end})`
      }
    }
    if (background.type === 'image' && background.image_url) {
      return {
        backgroundImage: `url(${background.image_url})`,
        backgroundSize: 'cover',
        backgroundPosition: 'center'
      }
    }
    return {}
  })

  const showFade = computed(
    () => Boolean(homeScreen.value?.background?.type) && Boolean(homeScreen.value?.fade_background)
  )

  const fadeStyle = {
    background:
      'linear-gradient(to bottom, transparent 0%, hsl(var(--background) / 0.08) 20%, hsl(var(--background) / 0.32) 45%, hsl(var(--background) / 0.72) 72%, hsl(var(--background)) 100%)'
  }

  return { textColorClass, subTextColorClass, backgroundStyle, showFade, fadeStyle }
}
