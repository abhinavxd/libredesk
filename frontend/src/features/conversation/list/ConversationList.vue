<template>
  <div class="h-screen flex flex-col">
    <!-- Header -->
    <div class="flex items-center space-x-4 px-2 h-12 border-b shrink-0">
      <SidebarTrigger class="cursor-pointer" />
      <span class="text-xl font-semibold">{{ title }}</span>
    </div>

    <!-- Filters -->
    <div class="p-2 flex justify-between items-center">
      <!-- Status dropdown-menu, hidden when a view is selected as views are pre-filtered -->
      <DropdownMenu v-if="!route.params.viewID">
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            <div>
              <span class="mr-1">{{ conversationStore.conversations.total }}</span>
              <span>{{ conversationStore.getListStatus }}</span>
            </div>
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            v-for="status in conversationStore.statusOptions"
            :key="status.value"
            @click="handleStatusChange(status)"
          >
            {{ status.label }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <div v-else>
        <Button variant="ghost" class="w-30">
          <span>{{ conversationStore.conversations.total }}</span>
        </Button>
      </div>

      <!-- Sort dropdown-menu -->
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            {{ conversationStore.getListSortField }}
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="handleSortChange('oldest')">
            {{ $t('conversation.sort.oldestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('newest')">
            {{ $t('conversation.sort.newestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_first')">
            {{ $t('conversation.sort.startedFirst') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_last')">
            {{ $t('conversation.sort.startedLast') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('waiting_longest')">
            {{ $t('conversation.sort.waitingLongest') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('next_sla_target')">
            {{ $t('conversation.sort.nextSLATarget') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('priority_first')">
            {{ $t('conversation.sort.priorityFirst') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Content -->
    <div class="flex-grow overflow-y-auto">
      <EmptyList
        v-if="!hasConversations && !hasErrored && !isLoading"
        key="empty"
        class="px-4 py-8"
        :title="t('conversation.noConversationsFound')"
        :message="t('conversation.tryAdjustingFilters')"
        :icon="MessageCircleQuestion"
      />

      <!-- Error State -->
      <EmptyList
        v-if="conversationStore.conversations.errorMessage"
        key="error"
        class="px-4 py-8"
        :title="t('conversation.couldNotFetch')"
        :message="conversationStore.conversations.errorMessage"
        :icon="MessageCircleWarning"
      />

      <!-- Empty State -->
      <TransitionGroup
        enter-active-class="transition-all duration-300 ease-in-out"
        enter-from-class="opacity-0 transform translate-y-4"
        enter-to-class="opacity-100 transform translate-y-0"
        leave-active-class="transition-all duration-300 ease-in-out"
        leave-from-class="opacity-100 transform translate-y-0"
        leave-to-class="opacity-0 transform translate-y-4"
      >
        <!-- Conversation List -->
        <div
          v-if="!conversationStore.conversations.errorMessage"
          key="list"
          class="divide-y divide-gray-200 dark:divide-gray-700"
          :class="{ 'border-b dark:border-gray-700': hasConversations }"
        >
          <ConversationListItem
            v-for="conversation in conversationStore.conversationsList"
            :key="conversation.uuid"
            :conversation="conversation"
            :currentConversation="conversationStore.current"
            :contactFullName="conversationStore.getContactFullName(conversation.uuid)"
            class="transition-colors duration-200 hover:bg-gray-50 dark:hover:bg-gray-600"
          />
        </div>

        <!-- Loading Skeleton -->
        <div v-if="isLoading" key="loading" class="space-y-4">
          <ConversationListItemSkeleton v-for="index in 5" :key="index" />
        </div>
      </TransitionGroup>

      <!-- Load More -->
      <div
        v-if="!hasErrored && (conversationStore.conversations.hasMore || hasConversations)"
        class="flex justify-center items-center p-5"
      >
        <Button
          v-if="conversationStore.conversations.hasMore"
          variant="outline"
          @click="loadNextPage"
          :disabled="isLoading"
          class="transition-all duration-200 ease-in-out transform hover:scale-105"
        >
          <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
          {{ isLoading ? t('globals.terms.loading') : t('globals.terms.loadMore') }}
        </Button>
        <p
          class="text-sm text-gray-500"
          v-else-if="conversationStore.conversationsList.length > 10"
        >
          {{ $t('conversation.allLoaded') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { MessageCircleQuestion, MessageCircleWarning, ChevronDown, Loader2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { SidebarTrigger } from '@/components/ui/sidebar'
import EmptyList from '@/features/conversation/list/ConversationEmptyList.vue'
import ConversationListItem from '@/features/conversation/list/ConversationListItem.vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ConversationListItemSkeleton from '@/features/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
const route = useRoute()
const { t } = useI18n()

const title = computed(() => {
  const typeValue = route.meta?.type?.(route)
  return (
    (typeValue || route.meta?.title || '').charAt(0).toUpperCase() +
    (typeValue || route.meta?.title || '').slice(1)
  )
})

const handleStatusChange = (status) => {
  conversationStore.setListStatus(status.label)
}

const handleSortChange = (order) => {
  conversationStore.setListSortField(order)
}

const loadNextPage = () => {
  conversationStore.fetchNextConversations()
}

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
const isLoading = computed(() => conversationStore.conversations.loading)
</script>
