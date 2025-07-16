<template>
  <div class="flex flex-col h-full">
    <!-- Chat header -->
    <ChatHeader @goBack="goBack" />

    <!-- Messages container -->
    <ChatMessages ref="chatMessages" />

    <!-- Error display -->
    <WidgetError :errorMessage="errorMessage" />

    <!-- Message input -->
    <MessageInput @error="handleError" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useWidgetStore } from '../store/widget.js'
import WidgetError from '@widget/components/WidgetError.vue'
import ChatHeader from '@widget/components/ChatHeader.vue'
import ChatMessages from '@widget/components/ChatMessages.vue'
import MessageInput from '@widget/components/MessageInput.vue'

const widgetStore = useWidgetStore()
const errorMessage = ref('')

const goBack = () => {
  widgetStore.navigateToMessages()
}

const handleError = (message) => {
  errorMessage.value = message
}
</script>
