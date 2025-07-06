<template>
  <div class="libredesk-widget-app">
    <div class="widget-container">
      <MainLayout />
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useWidgetStore } from './store/widget.js'
import api from '@widget/api/index.js'
import MainLayout from '@widget/layouts/MainLayout.vue'

const widgetStore = useWidgetStore()

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
  overflow: hidden;
}

.widget-container {
  width: 100%;
  height: 100%;
}
</style>
