<template>
  <ConversationPlaceholder v-if="!addressID" />
  <router-view v-else />
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ConversationPlaceholder from '@/features/conversation/ConversationPlaceholder.vue'

const route = useRoute()
const addressID = computed(() => route.params.addressID)
</script>
