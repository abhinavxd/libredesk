<template>
  <div class="flex h-full min-h-0 flex-col">
    <!-- Header -->
    <div class="flex h-14 shrink-0 items-center gap-3 border-b border-border px-3">
      <SidebarTrigger class="cursor-pointer" />
      <h1 class="ui-heading min-w-0 truncate" :title="title">{{ title }}</h1>
    </div>

    <!-- Bulk Action Toolbar (when items selected) -->
    <ConversationBulkActionToolbar v-if="hasSelection && canBulkAct" />

    <!-- Filters (hidden when bulk selecting) -->
    <div v-else class="flex shrink-0 items-center justify-between gap-2 border-b border-border p-2">
      <!-- Status dropdown-menu, hidden when a view is selected as views are pre-filtered -->
      <DropdownMenu v-if="!route.params.viewID">
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="sm" :class="FILTER_TRIGGER_CLASS">
            <div class="min-w-0 truncate">
              <span class="mr-1 tabular-nums">{{ conversationStore.conversations.total }}</span>
              <span>{{ conversationStore.getListStatus }}</span>
            </div>
            <ChevronDown class="size-3.5 shrink-0 opacity-50" aria-hidden="true" />
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
        <Button type="button" variant="ghost" size="sm" class="text-xs">
          <span class="tabular-nums">{{ conversationStore.conversations.total }}</span>
        </Button>
      </div>

      <!-- Sort dropdown-menu -->
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="sm" :class="FILTER_TRIGGER_CLASS">
            <span class="truncate" :title="conversationStore.getListSortField">{{
              conversationStore.getListSortField
            }}</span>
            <ChevronDown class="size-3.5 shrink-0 opacity-50" aria-hidden="true" />
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
    <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain">
      <EmptyList
        v-if="showEmpty"
        key="empty"
        class="px-4 py-8"
        :title="t('conversation.noConversationsFound')"
        :message="t('conversation.tryAdjustingFilters')"
        :icon="Inbox"
      />

      <EmptyList
        v-if="hasErrored"
        key="error"
        class="px-4 py-8"
        :title="t('conversation.couldNotFetch')"
        :message="conversationStore.conversations.errorMessage"
        :icon="MessageCircleWarning"
      />

      <div
        v-if="!hasErrored && !conversationStore.conversations.loading"
        class="divide-y divide-border"
        :class="{ 'border-b border-border': hasConversations }"
      >
        <ConversationListItem
          v-for="conversation in conversationStore.conversationsList"
          :key="conversation.uuid"
          :conversation="conversation"
          :currentConversation="conversationStore.current"
          :contactFullName="conversationStore.getContactFullName(conversation.uuid)"
          class="transition-colors duration-200"
        />
      </div>

      <div v-if="conversationStore.conversations.loading">
        <ConversationListItemSkeleton v-for="i in 12" :key="i" :index="i - 1" />
      </div>

      <!-- Load More -->
      <div
        v-if="!hasErrored && (conversationStore.conversations.hasMore || hasConversations)"
        class="flex justify-center items-center p-5"
      >
        <Button
          v-if="conversationStore.conversations.hasMore"
          variant="outline"
          @click="conversationStore.fetchNextConversations"
          :disabled="conversationStore.conversations.fetching"
          class="max-md:h-11"
        >
          <Loader2
            v-if="conversationStore.conversations.fetching"
            class="mr-2 h-4 w-4 animate-spin"
          />
          {{
            conversationStore.conversations.fetching
              ? t('globals.terms.loading')
              : t('globals.terms.loadMore')
          }}
        </Button>
        <p
          class="text-sm text-muted-foreground"
          v-else-if="conversationStore.conversationsList.length > 10"
        >
          {{ $t('conversation.allLoaded') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
const FILTER_TRIGGER_CLASS = 'min-w-0 justify-start gap-1.5 px-2 text-xs'

import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Inbox, MessageCircleWarning, ChevronDown, Loader2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import EmptyList from '@/features/conversation/list/ConversationEmptyList.vue'
import ConversationBulkActionToolbar from '@/features/conversation/list/ConversationBulkActionToolbar.vue'
import ConversationListItem from '@/features/conversation/list/ConversationListItem.vue'
import ConversationListItemSkeleton from '@/features/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamStore = useTeamStore()
const { canBulkAct } = useBulkActionPermissions()
const route = useRoute()
const { t } = useI18n()

watch(
  () => conversationStore.conversationsList.map((conversation) => conversation.assigned_user_id),
  usersStore.ensureUserIDs,
  { immediate: true }
)

watch(
  () => conversationStore.conversationsList
    .filter((conversation) => !conversation.assigned_user_id)
    .map((conversation) => conversation.assigned_team_id),
  teamStore.ensureTeamIDs,
  { immediate: true }
)

const hasSelection = computed(() => conversationStore.selectedCount > 0)

const title = computed(() => {
  const typeKey = route.meta?.typeKey?.(route)
  if (typeKey) {
    return t(typeKey)
  }
  const key = route.meta?.titleKey
  if (!key) return ''
  return t(key, route.meta?.titleCount || 1)
})

const handleStatusChange = (status) => {
  conversationStore.setListStatus(status.label)
}

const handleSortChange = (order) => {
  conversationStore.setListSortField(order)
}

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
const showEmpty = computed(
  () =>
    !hasConversations.value &&
    !hasErrored.value &&
    !conversationStore.conversations.loading &&
    conversationStore.conversations.initialized
)
</script>
