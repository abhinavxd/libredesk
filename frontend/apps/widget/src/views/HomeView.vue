<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 min-h-0 overflow-y-auto scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50">
      <div class="flex flex-col">
        <HomeHeader :config="config">
          <!-- Primary action renders on the gradient so it flows into the header. -->
          <RecentConversationCard
            v-if="mostRecentConversation"
            :conversation="mostRecentConversation"
          />
          <div v-else-if="canStartConversation">
            <Button @click="startConversation" class="w-full flex items-center justify-center">
              {{ startButtonText }}
              <ArrowRight v-if="!widgetStore.isRtl" size="16" />
              <ArrowLeft v-else size="16" />
            </Button>
          </div>
        </HomeHeader>

        <!-- Home Apps (announcements + external links) sit on the normal background. -->
        <div v-if="config.home_apps?.length" class="flex flex-col gap-3 p-4 bg-background">
          <div class="space-y-3">
            <template v-for="(item, index) in config.home_apps" :key="index">
              <AnnouncementCard v-if="item.type === 'announcement'" :announcement="item" />
              <HomeExternalLink v-else-if="item.type === 'external_link'" :link="item" />
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowRight, ArrowLeft } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import { useI18n } from 'vue-i18n'
import HomeHeader from '@widget/components/HomeHeader.vue'
import HomeExternalLink from '@widget/components/HomeExternalLink.vue'
import AnnouncementCard from '@widget/components/AnnouncementCard.vue'
import RecentConversationCard from '@widget/components/RecentConversationCard.vue'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const { t } = useI18n()
const config = computed(() => widgetStore.config)

const mostRecentConversation = computed(() => {
  const conversations = chatStore.getConversations
  if (!conversations || conversations.length === 0) return null
  // Get the most recent conversation (already sorted by last_message.created_at in the store)
  return conversations[0]
})

const canStartConversation = computed(() => {
  const userConfig = userStore.isVisitor ? config.value.visitors : config.value.users
  return userConfig?.prevent_multiple_conversations !== true || !chatStore.hasConversations
})

const startButtonText = computed(() => {
  const isVisitor = userStore.isVisitor
  return isVisitor
    ? config.value.visitors?.start_conversation_button_text || t('globals.messages.sendUsMessage')
    : config.value.users?.start_conversation_button_text || t('globals.messages.sendUsMessage')
})

const startConversation = () => {
  chatStore.setCurrentConversation(null)
  widgetStore.navigateToChat()
}
</script>
