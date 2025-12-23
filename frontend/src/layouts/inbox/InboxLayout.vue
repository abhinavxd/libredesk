<template>
  <div class="flex h-screen w-full" v-if="!isSearchRoute">
    <div class="w-80 2xl:w-[450px] shrink-0 border-r overflow-y-auto">
      <ConversationList />
    </div>
    <div class="flex-1 min-w-0">
      <router-view v-slot="{ Component }">
        <component :is="Component" />
      </router-view>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ConversationList from '@/features/conversation/list/ConversationList.vue'

const route = useRoute()
const isSearchRoute = computed(() => route.name === 'search')
</script>
