<template>
  <div class="flex h-screen w-full" v-if="!isSearchRoute">
    <div class="w-[450px] border-r shrink-0 overflow-y-auto">
      <ConversationList />
    </div>
    <div class="grow">
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
