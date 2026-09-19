<template>
  <div class="flex flex-col h-full">
    <div
      class="flex-1 min-h-0 overflow-y-auto scrollbar-thin scrollbar-track-transparent scrollbar-thumb-muted-foreground/30 hover:scrollbar-thumb-muted-foreground/50"
    >
      <div class="flex flex-col">
        <HomeHeader :config="config">
          <RecentConversationCard
            v-if="mostRecentConversation"
            :conversation="mostRecentConversation"
          />
          <div v-else-if="canStartConversation">
            <Button
              size="lg"
              class="w-full rounded-xl font-semibold shadow-md"
              @click="startConversation"
            >
              {{ startButtonText }}
              <ArrowRight size="16" aria-hidden="true" />
            </Button>
          </div>
        </HomeHeader>

        <div v-if="homeItems.length" class="space-y-3 bg-background px-4 pt-1 pb-5">
          <template v-for="(item, index) in homeItems" :key="index">
            <HomeHelp v-if="item.type === 'help'" />
            <AnnouncementCard v-else-if="item.type === 'announcement'" :announcement="item" />
            <HomeExternalLink v-else-if="item.type === 'external_link'" :link="item" />
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowRight } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useWidgetStore } from '@widget/store/widget.js'
import { useChatStore } from '@widget/store/chat.js'
import { useUserStore } from '@widget/store/user.js'
import { useI18n } from 'vue-i18n'
import HomeHeader from '@widget/components/HomeHeader.vue'
import HomeExternalLink from '@widget/components/HomeExternalLink.vue'
import AnnouncementCard from '@widget/components/AnnouncementCard.vue'
import RecentConversationCard from '@widget/components/RecentConversationCard.vue'
import HomeHelp from '@widget/components/HomeHelp.vue'
import { useHelpStore } from '@widget/store/help.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const userStore = useUserStore()
const { t } = useI18n()
const config = computed(() => widgetStore.config)
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

const canStartConversation = computed(() => {
  const userConfig = userStore.isVisitor ? config.value.visitors : config.value.users
  // Mirrors the server check, else the button is offered and the send fails.
  if (!userConfig?.allow_start_conversation) return false
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
