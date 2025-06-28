<template>
  <div class="libredesk-widget-app">
    <div class="widget-container">
      <!-- Welcome View -->
      <WelcomeView v-if="isWelcomeView" />
      <!-- Chat View -->
      <ChatView v-if="isChatView" />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import WelcomeView from './views/WelcomeView.vue'
import ChatView from './views/ChatView.vue'
import { useWidgetStore } from './store/widget.js'
import api from '@widget/api/index.js'

const widgetStore = useWidgetStore()
const isWelcomeView = computed(() => widgetStore.isWelcomeView)
const isChatView = computed(() => widgetStore.isChatView)

onMounted(async () => {
  await fetchWidgetSettings()
  widgetStore.openWidget()
})

// Fetch inbox widget settings from API
const fetchWidgetSettings = async () => {
  try {
    const urlParams = new URLSearchParams(window.location.search)
    const inboxID = urlParams.get('inbox_id')
    if (!inboxID) {
      throw new Error('`inbox_id` is missing in query parameters')
    }
    const response = await api.getWidgetSettings(inboxID)
    widgetStore.updateConfig(response.data.data)
  } catch (error) {
    console.error('Failed to fetch widget settings:', error)
  }
}
</script>

<style scoped>
.libredesk-widget-app {
  width: 100vw;
  height: 100vh;
  font-family: 'Plus Jakarta Sans', sans-serif;
  background: hsl(var(--background));
  overflow: hidden;
}

.widget-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

* {
  box-sizing: border-box;
}

html,
body {
  margin: 0;
  padding: 0;
  overflow: hidden;
}
</style>
