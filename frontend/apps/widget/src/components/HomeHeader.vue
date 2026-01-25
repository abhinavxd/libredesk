<template>
  <div class="p-4">
    <!-- Logo -->
    <img
      v-if="config.logo_url"
      :src="config.logo_url"
      :alt="config.brand_name"
      class="max-h-8 max-w-full"
    />
    <!-- Greeting and introduction -->
    <div class="mt-24 font-medium text-4xl">
      <h2>{{ parsedGreeting }}</h2>
      <p class="text-muted-foreground mt-2">
        {{ parsedIntroduction }}
      </p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useUserStore } from '@widget/store/user.js'
import { parseTemplateVariables } from '@shared-ui/utils/string.js'

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

const parsedGreeting = computed(() =>
  parseTemplateVariables(props.config.greeting_message, userData.value)
)

const parsedIntroduction = computed(() =>
  parseTemplateVariables(props.config.introduction_message, userData.value)
)
</script>
