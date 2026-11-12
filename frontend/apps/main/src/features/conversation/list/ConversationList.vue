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
            {{ i18n._('Oldest activity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('newest')">
            {{ i18n._('Newest activity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_first')">
            {{ i18n._('Started first') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_last')">
            {{ i18n._('Started last') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('waiting_longest')">
            {{ i18n._('Waiting longest') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('next_sla_target')">
            {{ i18n._('Next SLA target') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('priority_first')">
            {{ i18n._('Priority first') }}
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
        :title="i18n._('No conversations found')"
        :message="i18n._('Try adjusting filters')"
        :icon="Inbox"
      />

      <EmptyList
        v-if="hasErrored"
        key="error"
        class="px-4 py-8"
        :title="i18n._('Could not fetch conversations')"
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
          type="button"
          variant="ghost"
          size="sm"
          @click="conversationStore.fetchNextConversations"
          :disabled="conversationStore.conversations.fetching"
          :aria-busy="conversationStore.conversations.fetching || undefined"
          class="text-muted-foreground max-md:h-11 [&_svg]:size-3.5"
        >
          <Loader2
            v-if="conversationStore.conversations.fetching"
            class="animate-spin"
            aria-hidden="true"
          />
          {{
            conversationStore.conversations.fetching
              ? i18n._('Loading')
              : i18n._('Load more')
          }}
        </Button>
        <p
          class="text-sm text-muted-foreground"
          v-else-if="conversationStore.conversationsList.length > 10"
        >
          {{ i18n._('All conversations loaded') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
const FILTER_TRIGGER_CLASS = 'min-w-0 justify-start gap-1.5 px-2 text-xs'

import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
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
import { useTeamStore } from '@/stores/team'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import EmptyList from '@/features/conversation/list/ConversationEmptyList.vue'
import ConversationBulkActionToolbar from '@/features/conversation/list/ConversationBulkActionToolbar.vue'
import ConversationListItem from '@/features/conversation/list/ConversationListItem.vue'
import ConversationListItemSkeleton from '@/features/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
const teamStore = useTeamStore()
const { canBulkAct } = useBulkActionPermissions()
const route = useRoute()
const i18n = useI18n()

watch(
  () => conversationStore.conversationsList
    .filter((conversation) => !conversation.assigned_user_id)
    .map((conversation) => conversation.assigned_team_id),
  teamStore.ensureTeamIDs,
  { immediate: true }
)

const hasSelection = computed(() => conversationStore.selectedCount > 0)

const title = computed(() => {
  const typeMessage = route.meta?.typeMessage?.(route)
  if (typeMessage) {
    return i18n._(typeMessage)
  }
  const message = route.meta?.titleMessage
  if (!message) return ''
  return i18n._(message, route.meta?.titleCount || 1)
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
