<template>
  <div class="relative flex flex-col h-full bg-background">
    <div
      class="pointer-events-none absolute inset-x-0 top-0"
      :style="[backgroundStyle, { height: `${headerHeight}px` }]"
    >
      <div v-if="showFade" class="absolute inset-x-0 bottom-0 h-20" :style="fadeStyle"></div>
    </div>

    <div
      class="relative flex-1 min-h-0 overflow-y-auto scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50"
    >
      <HomeHeader ref="headerRef" :config="config">
        <RecentConversationCard
          v-if="mostRecentConversation"
          :conversation="mostRecentConversation"
        />
        <StartConversationButton
          v-else
          arrow
          size="lg"
          fallback-label="globals.messages.sendUsMessage"
          class="w-full font-semibold shadow-md"
        />
      </HomeHeader>

      <div v-if="homeItems.length" class="space-y-3 px-4 pt-1 pb-5">
        <template v-for="(item, index) in homeItems" :key="index">
          <HomeHelp v-if="item.type === 'help'" />
          <AnnouncementCard v-else-if="item.type === 'announcement'" :announcement="item" />
          <HomeExternalLink v-else-if="item.type === 'external_link'" :link="item" />
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useElementSize } from '@vueuse/core'
import { useWidgetStore } from '@widget/store/widget.js'
import { useHeaderTheme } from '@widget/composables/useHeaderTheme.js'
import { useChatStore } from '@widget/store/chat.js'
import HomeHeader from '@widget/components/HomeHeader.vue'
import StartConversationButton from '@widget/components/StartConversationButton.vue'
import HomeExternalLink from '@widget/components/HomeExternalLink.vue'
import AnnouncementCard from '@widget/components/AnnouncementCard.vue'
import RecentConversationCard from '@widget/components/RecentConversationCard.vue'
import HomeHelp from '@widget/components/HomeHelp.vue'
import { useHelpStore } from '@widget/store/help.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const config = computed(() => widgetStore.config)
const { backgroundStyle, showFade, fadeStyle } = useHeaderTheme(config)
const headerRef = ref(null)
const { height: headerHeight } = useElementSize(computed(() => headerRef.value?.$el))
const help = useHelpStore()
const homeItems = computed(() =>
  (config.value.home_apps || []).filter((item) => item.type !== 'help' || help.available)
)

const mostRecentConversation = computed(() => {
  const conversations = chatStore.getConversations
  if (!conversations || conversations.length === 0) return null
  // Get the most recent conversation (already sorted by last_message.created_at in the store)
  return conversations[0]
})

</script>
