<template>
  <div class="flex flex-col h-full relative">
    <!-- Header -->
    <WidgetHeader :title="$t('globals.terms.message', 2)" />

    <!-- Messages List -->
    <div class="flex-1 overflow-y-auto pb-20">
      <ConversationsList />
    </div>

    <!-- Floating button with gradient fade -->
    <div class="absolute bottom-0 inset-x-0">
      <!-- Gradient fade overlay -->
      <div
        class="h-20 bg-gradient-to-t from-background via-background/80 to-transparent pointer-events-none"
      ></div>

      <!-- Floating button -->
      <div class="absolute bottom-4 inset-x-0 mx-auto w-fit z-10">
        <StartConversationButton />
      </div>
    </div>
  </div>
</template>

<script setup>
import ConversationsList from '@widget/components/ConversationsList.vue'
import WidgetHeader from '@widget/layouts/WidgetHeader.vue'
import StartConversationButton from '@widget/components/StartConversationButton.vue'
</script>
