<template>
  <div class="relative">
    <div
      class="px-7 pb-7"
      :class="widgetStore.isMobileFullScreen ? 'pt-[max(3.5rem,env(safe-area-inset-top))]' : 'pt-7'"
    >
      <img
        v-if="widgetStore.branding?.logo_url"
        :src="widgetStore.branding?.logo_url"
        :alt="config.brand_name"
        class="max-h-7 max-w-full"
      />
      <div class="mt-20" :class="textColorClass">
        <h2
          class="text-3xl font-semibold leading-tight tracking-tight break-words"
          :class="subTextColorClass"
        >
          {{ parsedGreeting }}
        </h2>
        <p class="mt-2 text-3xl font-semibold leading-tight tracking-tight break-words">
          {{ parsedIntroduction }}
        </p>
      </div>
    </div>
    <div class="relative z-10 px-4 pb-5">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUserStore } from '@widget/store/user.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { useHeaderTheme } from '@widget/composables/useHeaderTheme.js'
import { renderTemplate } from '@shared-ui/utils/string.js'

const props = defineProps({
  config: {
    type: Object,
    required: true
  }
})

const userStore = useUserStore()
const widgetStore = useWidgetStore()

const userData = computed(() => ({
  firstName: userStore.firstName,
  lastName: userStore.lastName
}))

const parsedGreeting = computed(() => renderTemplate(props.config.greeting_message, userData.value))

const parsedIntroduction = computed(() =>
  renderTemplate(props.config.introduction_message, userData.value)
)

const { textColorClass, subTextColorClass } = useHeaderTheme()
</script>
