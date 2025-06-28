<template>
  <div class="flex flex-col p-4 h-full gap-5">
    <div class="p-2">
      <!-- Logo Section -->
      <div>
        <img v-if="config.logo_url" :src="config.logo_url" alt="Logo" class="max-h-8 max-w-full" />
        <MessageSquare v-else class="w-6 h-6 text-primary" />
      </div>

      <!-- Welcome Message -->
      <div class="mt-24 font-medium text-3xl">
        <h2 class="text-3xl">{{ config.greeting_message || 'Hi there' }}</h2>
        <p class="text-muted-foreground text-3xl mt-2">
          {{ config.introduction_message || 'How can we help?' }}
        </p>
      </div>
    </div>

    <!-- Start Conversation Button -->
    <div class="mt-3">
      <div>
        <Button
          v-if="canStartConversation"
          @click="startConversation"
          class="w-full h-12"
          size="lg"
        >
          {{ getStartButtonText() }}
          <ArrowRight class="w-4 h-4 ml-2" />
        </Button>
      </div>
    </div>

    <!-- External Links -->
    <div v-if="config.external_links?.length" class="mt-2 space-y-2">
      <a
        v-for="link in config.external_links"
        :key="link.url"
        :href="link.url"
        target="_blank"
        rel="noopener noreferrer"
        class="block no-underline"
      >
        <Card class="hover:bg-accent transition-colors cursor-pointer">
          <CardContent class="p-4">
            <div class="flex justify-between items-center">
              <span class="text-sm text-primary font-medium">
                {{ link.text }}
              </span>
              <ExternalLink size="18" class="text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      </a>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { MessageSquare, ArrowRight, ExternalLink } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Card, CardContent } from '@shared-ui/components/ui/card'
import { useWidgetStore } from '../store/widget.js'

const widgetStore = useWidgetStore()

const config = computed(() => widgetStore.config)

const canStartConversation = computed(() => {
  const isAuthenticated = widgetStore.user
  return isAuthenticated
    ? config.value.users?.allow_start_conversation
    : config.value.visitors?.allow_start_conversation
})

const getStartButtonText = () => {
  const isAuthenticated = widgetStore.user
  return isAuthenticated
    ? config.value.users?.start_conversation_button_text || 'Send us a message'
    : config.value.visitors?.start_conversation_button_text || 'Send us a message'
}

const startConversation = () => {
  widgetStore.navigateToChat()
}
</script>
